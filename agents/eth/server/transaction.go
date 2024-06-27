package server

import (
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethtypes "github.com/ubtr/ubt-go/agents/eth/types"
	"github.com/ubtr/ubt/go/api/proto"
)

const Erc20Transfer = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" //Transfer(address,address,uint256)

type TxConverter struct {
	Srv *EthServer

	Log *slog.Logger
}

func (c *TxConverter) Convert(ethTx *ethtypes.RpcTx, logs []types.Log) (*proto.Transaction, error) {
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

	valueBytes := []byte{0}
	if ethTx.Tx.Value() != nil && ethTx.Tx.Value().Sign() > 0 {
		valueBytes = ethTx.Tx.Value().Bytes()
	}

	return &proto.Transaction{
		Id:        ethTx.TxHash.Bytes(),
		From:      c.Srv.AddressToString(ethTx.TxExtraInfo.From),
		To:        c.Srv.AddressToString(ethTx.Tx.To()),
		BlockId:   ethTx.BlockHash.Bytes(),
		Type:      uint32(ethTx.Tx.Type()),
		Fee:       &proto.Uint256{Data: []byte{0}},
		Amount:    &proto.Uint256{Data: valueBytes},
		Idx:       uint32(ethTx.TxExtraInfo.TransactionIndex),
		Transfers: transfers,
	}, nil
}

func (c *TxConverter) ConvertNativeTransfer(ethTx *ethtypes.RpcTx) (*proto.Transfer, error) {
	trfId := append(ethTx.TxHash.Bytes(), 0)
	return &proto.Transfer{
		Id:     trfId,
		TxId:   ethTx.TxHash.Bytes(),
		OpId:   trfId,
		From:   c.Srv.AddressToString(ethTx.TxExtraInfo.From),
		To:     c.Srv.AddressToString(ethTx.Tx.To()),
		Status: 1,
		Amount: &proto.CurrencyAmount{CurrencyId: "", Value: &proto.Uint256{Data: ethTx.Tx.Value().Bytes()}},
	}, nil
}

func (c *TxConverter) ConvertERC20Transfer(ethTx *ethtypes.RpcTx, logs []types.Log) ([]*proto.Transfer, error) {
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

func (c *TxConverter) DecodeLogAsTransfer(ethTx *ethtypes.RpcTx, log types.Log) (*proto.Transfer, error) {
	currencyId := c.Srv.AddressToString(&log.Address)
	fromAddr := common.BytesToAddress(log.Topics[1].Bytes())
	toAddr := common.BytesToAddress(log.Topics[2].Bytes())
	trfId := append(ethTx.TxHash.Bytes(), big.NewInt(int64(log.TxIndex)).Bytes()...)
	return &proto.Transfer{
		Id:     trfId,
		TxId:   ethTx.TxHash.Bytes(),
		OpId:   trfId,
		From:   c.Srv.AddressToString(&fromAddr),
		To:     c.Srv.AddressToString(&toAddr),
		Status: 0,
		Amount: &proto.CurrencyAmount{CurrencyId: currencyId, Value: &proto.Uint256{Data: log.Data}},
	}, nil
}
