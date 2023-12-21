package server

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/ubtr/ubt-go/eth/client"
	"github.com/ubtr/ubt-go/eth/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt/go/api/proto"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const Erc20Transfer = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" //Transfer(address,address,uint256)

type RpcTx struct {
	Tx *types.Transaction
	TxExtraInfo
}

func (tx *RpcTx) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.Tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.TxExtraInfo)
}

type RpcBody struct {
	Transactions []*RpcTx
}

type HeaderWithBody struct {
	Header types.Header
	Body   RpcBody
}

func (b *HeaderWithBody) UnmarshalJSON(input []byte) error {
	if err := b.Header.UnmarshalJSON(input); err != nil {
		return err
	}

	var txStruct struct {
		Transactions []*RpcTx `json:"transactions"`
	}

	slog.Debug("Suka", "input", string(input))
	if err := json.Unmarshal(input, &txStruct); err != nil {
		return err
	}
	slog.Debug("Suka2")
	b.Body.Transactions = txStruct.Transactions

	return nil
}

type TxExtraInfo struct {
	BlockNumber      *string         `json:"blockNumber,omitempty"`
	BlockHash        *common.Hash    `json:"blockHash,omitempty"`
	From             *common.Address `json:"from,omitempty"`
	TransactionIndex hexutil.Uint64  `json:"transactionIndex"`
}

type BlockConverter struct {
	Config *config.ChainConfig
	Client *client.RateLimitedClient
	Ctx    context.Context
}

func (c *BlockConverter) loadAndGroupLogs(block *HeaderWithBody) (map[uint][]types.Log, error) {
	slog.Debug("Loading logs for block", "block", block.Header.Number)
	blockId := block.Header.Hash()

	logs, err := c.Client.FilterLogs(c.Ctx, ethereum.FilterQuery{BlockHash: &blockId})
	if err != nil {
		slog.Error("Failed to load logs", "block", block.Header.Number, "err", err)
		return nil, err
	}

	res := make(map[uint][]types.Log)
	for _, log := range logs {
		logGroup, ok := res[log.TxIndex]
		if !ok {
			logGroup = []types.Log{log}
			res[log.TxIndex] = logGroup
		} else {
			res[log.TxIndex] = append(logGroup, log)
		}
	}
	return res, nil
}

func (c *BlockConverter) EthBlockToProto(block *HeaderWithBody) (*proto.Block, error) {
	ret := &proto.Block{
		Header: &proto.BlockHeader{
			Id:             block.Header.TxHash.Bytes(),
			Number:         block.Header.Number.Uint64(),
			ParentId:       block.Header.ParentHash.Bytes(),
			Timestamp:      timestamppb.New(time.Unix(int64(block.Header.Time), 0)),
			FinalityStatus: proto.FinalityStatus_FINALITY_STATUS_UNSPECIFIED,
		},
		Transactions: []*proto.Transaction{},
	}

	logs, err := c.loadAndGroupLogs(block)
	if err != nil {
		return nil, err
	}

	for _, tx := range block.Body.Transactions {
		txProto, err := c.EthTransactionToProto(tx, logs[uint(tx.TransactionIndex)])
		if err != nil {
			return nil, err
		}
		ret.Transactions = append(ret.Transactions, txProto)
	}

	return ret, nil
}

func (c *BlockConverter) EthTransactionToProto(ethTx *RpcTx, logs []types.Log) (*proto.Transaction, error) {
	transfers := []*proto.Transfer{}

	log.Printf("TxLogs %s : %+v", ethTx.Tx.Hash().String(), logs)
	if ethTx.Tx.Value().Sign() > 0 {
		transfer, err := c.ConvertNativeTransfer(ethTx)
		if err != nil {
			return nil, err
		}
		if transfer != nil {
			transfers = append(transfers, transfer)
		}
	}

	if len(logs) > 0 {
		erc20Transfers, err := c.ConvertERC20Transfer(ethTx, logs)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, erc20Transfers...)
	}

	toString := ""
	if ethTx.Tx.To() != nil {
		toString = ethTx.Tx.To().String()
	}

	fromString := ""
	if ethTx.TxExtraInfo.From != nil {
		fromString = ethTx.TxExtraInfo.From.String()
	}

	valueBytes := []byte{0}
	if ethTx.Tx.Value() != nil && ethTx.Tx.Value().Sign() > 0 {
		valueBytes = ethTx.Tx.Value().Bytes()
	}

	return &proto.Transaction{
		Id:        ethTx.Tx.Hash().Bytes(),
		To:        toString,
		From:      fromString,
		BlockId:   ethTx.BlockHash.Bytes(),
		Type:      uint32(ethTx.Tx.Type()),
		Fee:       &proto.Uint256{Data: []byte{0}},
		Amount:    &proto.Uint256{Data: valueBytes},
		Idx:       uint32(ethTx.TxExtraInfo.TransactionIndex),
		Transfers: transfers,
	}, nil
}

func (c *BlockConverter) getCurrencyId() string {
	return c.Config.ChainType + ":" + c.Config.ChainNetwork
}

func (c *BlockConverter) ConvertNativeTransfer(ethTx *RpcTx) (*proto.Transfer, error) {
	return &proto.Transfer{
		TxId:   ethTx.Tx.Hash().Bytes(),
		From:   ethTx.TxExtraInfo.From.String(),
		To:     ethTx.Tx.To().String(),
		Status: 1,
		Amount: &proto.CurrencyAmount{CurrencyId: c.getCurrencyId(), Value: &proto.Uint256{Data: ethTx.Tx.Value().Bytes()}},
	}, nil
}

func (c *BlockConverter) ConvertERC20Transfer(ethTx *RpcTx, logs []types.Log) ([]*proto.Transfer, error) {
	//logs, err := c.Client.FilterLogs(*c.Ctx, ethereum.FilterQuery{BlockHash: ethTx.BlockHash})
	//if err != nil {
	//	return nil, err
	//}

	var transfers []*proto.Transfer
	for _, log := range logs {
		if len(log.Topics) > 2 && strings.HasSuffix(log.Topics[0].Hex(), Erc20Transfer) {
			transfer, err := c.DecodeLogAsTransfer(ethTx, log)
			if err != nil {
				return nil, err
			}
			transfers = append(transfers, transfer)
		}
	}

	return transfers, nil
}

func (c *BlockConverter) DecodeLogAsTransfer(ethTx *RpcTx, log types.Log) (*proto.Transfer, error) {
	currencyId := c.getCurrencyId() + ":" + log.Address.String()

	return &proto.Transfer{
		TxId:   ethTx.Tx.Hash().Bytes(),
		OpId:   log.TxHash.Bytes(),
		From:   common.BytesToAddress(log.Topics[1].Bytes()).String(),
		To:     common.BytesToAddress(log.Topics[2].Bytes()).String(),
		Status: 0,
		Amount: &proto.CurrencyAmount{CurrencyId: currencyId, Value: &proto.Uint256{Data: log.Data}},
	}, nil
}
