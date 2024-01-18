package agent

import (
	"context"

	"github.com/ubtr/ubt-go/eth/config"
	"github.com/ubtr/ubt/go/api/proto/services"
)

type UbtAgent interface {
	services.UbtChainServiceServer
	services.UbtBlockServiceServer
	services.UbtConstructServiceServer
	services.UbtCurrencyServiceServer
	String() string
}

type UbtAgentFactory func(ctx context.Context, config *config.ChainConfig) UbtAgent

var AgentFactories = make(map[string]UbtAgentFactory)
