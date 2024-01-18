/*
  JSON RPC abstraction with batchable typed calls
*/

package jsonrpc

import (
	"context"
	"io"
)

type IRpcClient interface {
	io.Closer
	Call(raw *RawCall) error
	CallContext(ctx context.Context, raw *RawCall) error
	BatchCallContext(ctx context.Context, batch *RpcBatch) error
}

type RawCall struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	Result any    `json:"result"`
	Error  error  `json:"error"`
}

type RpcBatch struct {
	Calls []*RawCall
}

func (c *RpcBatch) Add(raw *RawCall) {
	c.Calls = append(c.Calls, raw)
}

func (c *RpcBatch) Call(ctx context.Context, client IRpcClient) error {
	err := client.BatchCallContext(ctx, c)
	if err != nil {
		return err
	}

	// TODO: call ProcessRes when it will be possible
	return nil
}

// typed
type RpcCall[R any] struct {
	raw        RawCall
	resConvert func() error // delayed conversion closure from raw.Result to Response
	Response   *R
	Error      error
}

func (c *RpcCall[R]) Call(ctx context.Context, client IRpcClient) (R, error) {
	err := client.CallContext(ctx, &c.raw)
	if err != nil {
		return *c.Response, err
	}
	if c.resConvert != nil {
		err = c.resConvert()
	}

	return *c.Response, err
}

func (c *RpcCall[R]) AddToBatch(batch *RpcBatch) {
	batch.Add(&c.raw)
}

func (c *RpcCall[R]) ProcessRes(ctx context.Context) error {
	if c.raw.Error != nil {
		return c.raw.Error
	}
	if c.resConvert != nil {
		c.raw.Error = c.resConvert()
	}

	return c.raw.Error
}

func NewRpcCall[R any](method string, params []any, res any, response *R, resConvert func() error) *RpcCall[R] {
	return &RpcCall[R]{
		raw: RawCall{
			Method: method,
			Params: params,
			Result: res,
		},
		Response:   response,
		resConvert: resConvert,
	}
}

func AnyCall(method string, response any, params ...any) *RpcCall[any] {
	return NewRpcCall[any](method, params, response, &response, nil)
}
