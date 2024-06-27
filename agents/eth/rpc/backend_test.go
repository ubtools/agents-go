package rpc

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestImplements(t *testing.T) {
	var _ bind.ContractBackend = &EthRpcBackend{}
}
