package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
)

type TClient struct {
}

func (c *TClient) Call(raw *RawCall) error {
	return c.CallContext(context.Background(), raw)
}

type jsonreq struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	Id      int    `json:"id"`
}

type jsonRes struct {
	Id     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *jsonError      `json:"error"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}

func (err *jsonError) ErrorData() interface{} {
	return err.Data
}

var ErrNoResult = errors.New("RPC response has no result")

func (c *TClient) CallContext(ctx context.Context, raw *RawCall) error {
	_, err := json.Marshal(jsonreq{
		Jsonrpc: "2.0",
		Method:  raw.Method,
		Params:  raw.Params,
		Id:      1,
	})
	if err != nil {
		return err
	}
	//fmt.Println(string(data))
	json.Unmarshal([]byte("{\"result\":1,\"error\":null}"), raw)
	return nil
}

func generateBatchResponse(size int) string {
	res := "["
	for i := 0; i < size; i++ {
		res += "{\"result\":1, \"id\": " + strconv.Itoa(i) + "},"
	}
	//fmt.Println(res)
	return res[:len(res)-1] + "]"
}

var batchResponse = []byte(generateBatchResponse(100))

func (c *TClient) BatchCallContext(ctx context.Context, batch *RpcBatch) error {
	// exec batch request
	var reqs []jsonreq
	//var byId = make(map[int]*RawCall)
	for i, call := range batch.Calls {
		reqs = append(reqs, jsonreq{
			Jsonrpc: "2.0",
			Method:  call.Method,
			Params:  call.Params,
			Id:      i,
		})
		//byId[i] = call
	}
	_, err := json.Marshal(reqs)
	if err != nil {
		return err
	}
	var responses []jsonRes
	err = json.Unmarshal(batchResponse, &responses)
	if err != nil {
		return err
	}

	for _, res := range responses {
		call := batch.Calls[res.Id]
		//delete(byId, res.Id)

		switch {
		case res.Error != nil:
			call.Error = res.Error
		case res.Result == nil:
			call.Error = ErrNoResult
		default:
			call.Error = json.Unmarshal(res.Result, call.Result)
		}
	}

	return nil
}

func (c *TClient) Close() error {
	return nil
}

func BenchmarkStruct(b *testing.B) {

	client := &TClient{}
	for i := 0; i < b.N; i++ {
		rpcCall1 := rpcCall(true)
		rpcCall1.Call(context.Background(), client)
	}
}

func BenchmarkBatchStruct(b *testing.B) {

	client := &TClient{}
	ctx := context.Background()
	for n := 0; n < b.N; n++ {
		var batch RpcBatch
		var res []*RpcCall[int]
		for i := 0; i < 100; i++ {
			c := rpcCall(i%2 == 0)
			res = append(res, &c)
			c.AddToBatch(&batch)
		}

		client.BatchCallContext(ctx, &batch)
		for _, c := range res {
			c.ProcessRes(context.Background())
		}
	}
}

func BenchmarkBatchStructSimple(b *testing.B) {
	client := &TClient{}
	ctx := context.Background()
	var batch RpcBatch
	var res []*RpcCall[int]

	for i := 0; i < 100; i++ {
		c := rpcCall(i%2 == 0)
		res = append(res, &c)
		c.AddToBatch(&batch)
	}

	for i := 0; i < b.N; i++ {
		err := batch.Call(ctx, client)
		if err != nil {
			panic(err)
		}
		for _, call := range res {
			call.ProcessRes(ctx)
		}
	}
}

func rpcCall(arg bool) RpcCall[int] {
	var rawres int
	var res int
	return RpcCall[int]{
		raw: RawCall{
			Method: "eth_chainId",
			Params: []any{arg},
			Result: &rawres,
		},
		Response: &res,
		resConvert: func() error {
			res = rawres
			return nil
		},
	}
}

func BenchmarkStructIface(b *testing.B) {
	rpcCall(true)

	for i := 0; i < b.N; i++ {
		rpcCall(true)
	}
}
