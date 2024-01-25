package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt-go/agent"
	"github.com/ubtr/ubt-go/commons/jsonrpc/client"
	"github.com/ubtr/ubt-go/eth/rpc"
	ethtypes "github.com/ubtr/ubt-go/eth/types"
	"github.com/ubtr/ubt/go/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BlockConverter struct {
	Config *agent.ChainConfig
	Client *client.BalancedClient
	Ctx    context.Context
	Srv    *EthServer
	Log    *slog.Logger
}

const slotTimeSec = uint64(12)

func (srv *BlockConverter) getBlockFinalityStatus(block *ethtypes.HeaderWithBody) proto.FinalityStatus {
	if block.Header.Time < uint64(time.Now().Unix())-54*slotTimeSec {
		return proto.FinalityStatus_FINALITY_STATUS_FINALIZED
	} else if block.Header.Time < uint64(time.Now().Unix())-32*slotTimeSec {
		return proto.FinalityStatus_FINALITY_STATUS_SAFE
	} else {
		return proto.FinalityStatus_FINALITY_STATUS_UNSAFE
	}
}

func (c *BlockConverter) loadAndGroupLogs(block *ethtypes.HeaderWithBody) (map[uint][]types.Log, error) {
	c.Log.Debug("Loading logs for block")
	blockId := block.BlockHash

	logs, err := rpc.AdoptClient(c.Client).FilterLogs(c.Ctx, ethereum.FilterQuery{BlockHash: &blockId})
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

func (c *BlockConverter) EthBlockToProto(block *ethtypes.HeaderWithBody) (*proto.Block, error) {
	ret := &proto.Block{
		Header: &proto.BlockHeader{
			Id:             block.BlockHash.Bytes(),
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
		txConverter := &TxConverter{Srv: c.Srv, Log: c.Log.With("txId", tx.Tx.Hash().String(), "txIndex", uint64(tx.TransactionIndex))}
		txProto, err := txConverter.Convert(tx, logs[uint(tx.TransactionIndex)])
		if err != nil {
			return nil, err
		}
		ret.Transactions = append(ret.Transactions, txProto)
	}

	return ret, nil
}
