package client

import (
	"testing"

	"github.com/ubtr/ubt-go/commons/jsonrpc"
)

func TestImplements(t *testing.T) {
	var _ jsonrpc.IRpcClient = &EthRpcClient{}
}
