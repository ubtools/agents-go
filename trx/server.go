package trx

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shengdoushi/base58"
	"github.com/ubtr/ubt-go/agent"
	"github.com/ubtr/ubt-go/blockchain/trx"
	"github.com/ubtr/ubt-go/eth/config"
	"github.com/ubtr/ubt-go/eth/server"
)

func init() {
	agent.AgentFactories[trx.CODE_STR] = func(ctx context.Context, config *config.ChainConfig) agent.UbtAgent {
		return InitServer(ctx, config)
	}
}

var TrxExtensions = server.Extensions{
	AddressFromString: func(address string) (common.Address, error) {
		addrB58, err := base58.Decode(address, base58.BitcoinAlphabet)
		if err != nil {
			return common.Address{}, err
		}
		addrB58 = addrB58[:len(addrB58)-4]
		return common.BytesToAddress(addrB58[len(addrB58)-20:]), nil
	},
	AddressToString: func(address common.Address) string {
		addressTron := make([]byte, 0)
		addressTron = append(addressTron, trx.TronBytePrefix)
		addressTron = append(addressTron, address.Bytes()...)
		return trx.Address(addressTron).String()
	},
}

func InitServer(ctx context.Context, config *config.ChainConfig) *TrxAgent {
	agent := &TrxAgent{
		EthServer: *server.InitServer(ctx, config),
	}
	agent.EthServer.Extensions = TrxExtensions

	return agent
}

type TrxAgent struct {
	server.EthServer
}
