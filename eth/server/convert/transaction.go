package convert

import (
	"log/slog"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt-go/trx/common"
	"github.com/ubtr/ubt/go/api/proto"
)

const Erc20Transfer = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" //Transfer(address,address,uint256)

type TxConverter struct {
	Log *slog.Logger
}

func (c *TxConverter) Convert(ethTx *RpcTx, logs []types.Log) (*proto.Transaction, error) {
	transfers := []*proto.Transfer{}

	c.Log.Debug("TxLogs", "logs", logs)
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

func (c *TxConverter) ConvertNativeTransfer(ethTx *RpcTx) (*proto.Transfer, error) {
	return &proto.Transfer{
		TxId:   ethTx.Tx.Hash().Bytes(),
		From:   ethTx.TxExtraInfo.From.String(),
		To:     ethTx.Tx.To().String(),
		Status: 1,
		Amount: &proto.CurrencyAmount{CurrencyId: "", Value: &proto.Uint256{Data: ethTx.Tx.Value().Bytes()}},
	}, nil
}

func (c *TxConverter) ConvertERC20Transfer(ethTx *RpcTx, logs []types.Log) ([]*proto.Transfer, error) {
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

func (c *TxConverter) DecodeLogAsTransfer(ethTx *RpcTx, log types.Log) (*proto.Transfer, error) {
	currencyId := log.Address.String()

	return &proto.Transfer{
		TxId:   ethTx.Tx.Hash().Bytes(),
		OpId:   log.TxHash.Bytes(),
		From:   common.BytesToAddress(log.Topics[1].Bytes()).String(),
		To:     common.BytesToAddress(log.Topics[2].Bytes()).String(),
		Status: 0,
		Amount: &proto.CurrencyAmount{CurrencyId: currencyId, Value: &proto.Uint256{Data: log.Data}},
	}, nil
}
