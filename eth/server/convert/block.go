package convert

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt-go/eth/client"
	"github.com/ubtr/ubt-go/eth/config"
	"github.com/ubtr/ubt-go/trx/common"
	"github.com/ubtr/ubt-go/trx/common/hexutil"
	"github.com/ubtr/ubt/go/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

	if err := json.Unmarshal(input, &txStruct); err != nil {
		return err
	}
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
	Log    *slog.Logger
}

const slotTimeSec = uint64(12)

func (srv *BlockConverter) getBlockFinalityStatus(block *HeaderWithBody) proto.FinalityStatus {
	if block.Header.Time < uint64(time.Now().Unix())-54*slotTimeSec {
		return proto.FinalityStatus_FINALITY_STATUS_FINALIZED
	} else if block.Header.Time < uint64(time.Now().Unix())-32*slotTimeSec {
		return proto.FinalityStatus_FINALITY_STATUS_SAFE
	} else {
		return proto.FinalityStatus_FINALITY_STATUS_UNSAFE
	}
}

func (c *BlockConverter) loadAndGroupLogs(block *HeaderWithBody) (map[uint][]types.Log, error) {
	c.Log.Debug("Loading logs for block")
	blockId := block.Header.Hash()

	logs, err := c.Client.FilterLogs(c.Ctx, ethereum.FilterQuery{BlockHash: &blockId})
	if err != nil {
		slog.Error("Failed to load logs", "err", err)
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
			FinalityStatus: c.getBlockFinalityStatus(block),
		},
		Transactions: []*proto.Transaction{},
	}

	logs, err := c.loadAndGroupLogs(block)
	if err != nil {
		return nil, err
	}

	for _, tx := range block.Body.Transactions {
		txConverter := &TxConverter{Log: c.Log.With("txId", tx.Tx.Hash().String(), "txIndex", tx.TransactionIndex)}
		txProto, err := txConverter.Convert(tx, logs[uint(tx.TransactionIndex)])
		if err != nil {
			return nil, err
		}
		ret.Transactions = append(ret.Transactions, txProto)
	}

	return ret, nil
}
