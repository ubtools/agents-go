package rpc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ubtr/ubt-go/commons/jsonrpc"
	ethtypes "github.com/ubtr/ubt-go/eth/types"
)

func GetBlockNumber() *jsonrpc.RpcCall[uint64] {
	var res string
	var response uint64
	return jsonrpc.NewRpcCall[uint64](
		"eth_blockNumber",
		[]any{},
		&res,
		&response,
		func() error {
			dec, err := hexutil.DecodeUint64(res)
			if err != nil {
				return err
			}
			response = dec
			return nil
		},
	)
}

func GetBlockByHash(hash common.Hash, txDetails bool) *jsonrpc.RpcCall[*ethtypes.HeaderWithBody] {
	var res ethtypes.HeaderWithBody
	var response *ethtypes.HeaderWithBody = &res
	return jsonrpc.NewRpcCall[*ethtypes.HeaderWithBody](
		"eth_getBlockByHash",
		[]any{hash, txDetails},
		&res,
		&response,
		nil,
	)
}

func GetBlockByNumber(number *big.Int, txDetails bool) *jsonrpc.RpcCall[ethtypes.HeaderWithBody] {
	var res ethtypes.HeaderWithBody
	return jsonrpc.NewRpcCall[ethtypes.HeaderWithBody](
		"eth_getBlockByNumber",
		[]any{toBlockNumArg(number), txDetails},
		&res,
		&res,
		nil,
	)
}

func ChainId() *jsonrpc.RpcCall[*big.Int] {
	var res hexutil.Big
	var response *big.Int = (*big.Int)(&res)
	return jsonrpc.NewRpcCall[*big.Int](
		"eth_chainId",
		[]any{},
		&res,
		&response,
		nil,
	)
}
