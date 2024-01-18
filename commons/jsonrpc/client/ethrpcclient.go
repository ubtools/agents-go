package client

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ubtr/ubt-go/commons/jsonrpc"
)

// temporary implementation using eth rpc.Client

type EthRpcClient struct {
	client *rpc.Client
}

func DialContext(ctx context.Context, rawUrl string) (*EthRpcClient, error) {
	c, err := rpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, err
	}
	return &EthRpcClient{client: c}, nil
}

func (c *EthRpcClient) Call(raw *jsonrpc.RawCall) error {
	return c.CallContext(context.Background(), raw)
}

func (c *EthRpcClient) CallContext(ctx context.Context, raw *jsonrpc.RawCall) error {
	return c.client.CallContext(ctx, raw.Result, raw.Method, raw.Params...)
}

func (c *EthRpcClient) BatchCallContext(ctx context.Context, batch *jsonrpc.RpcBatch) error {
	var elems []rpc.BatchElem
	for _, call := range batch.Calls {
		elems = append(elems, rpc.BatchElem{
			Method: call.Method,
			Args:   call.Params,
			Result: call.Result,
		})
	}

	res := c.client.BatchCallContext(ctx, elems)
	for i, call := range batch.Calls {
		call.Error = elems[i].Error
	}
	slog.With(ctx).Debug("EthBatchResponse", "result", res)
	return res
}

func (c *EthRpcClient) BatchCall(batch *jsonrpc.RpcBatch) error {
	return c.BatchCallContext(context.Background(), batch)
}

func (c *EthRpcClient) Close() error {
	c.client.Close()
	return nil
}
