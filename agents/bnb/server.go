package bnb

import (
	"context"

	"github.com/ubtr/ubt-go/agent"
	ethagent "github.com/ubtr/ubt-go/agents/eth/server"
	"github.com/ubtr/ubt-go/blockchain/bnb"
)

func init() {
	agent.AgentFactories[bnb.CODE_STR] = func(ctx context.Context, config *agent.ChainConfig) agent.UbtAgent {
		return ethagent.InitServer(ctx, config)
	}
}
