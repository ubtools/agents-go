package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ubtr/ubt-go/agent"
	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/blockchain/bnb"
	_ "github.com/ubtr/ubt-go/blockchain/bnb"
	"github.com/ubtr/ubt-go/blockchain/eth"
	_ "github.com/ubtr/ubt-go/blockchain/eth"
	_ "github.com/ubtr/ubt-go/blockchain/trx"

	"github.com/ubtr/ubt-go/cmd/cmdutil"

	_ "github.com/ubtr/ubt-go/eth/server"
	"github.com/ubtr/ubt-go/proxy"
	_ "github.com/ubtr/ubt-go/trx"

	"github.com/ubtr/ubt/go/api/proto/services"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
)

func main() {
	app := &cli.App{
		Name:            "agent-eth",
		Usage:           "UBT Eth-like agent",
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log",
				Value: "INFO",
				Usage: "Log level",
			},
			&cli.StringFlag{
				Name:    "listen",
				Aliases: []string{"L"},
				Value:   ":50051",
				Usage:   "Host+port to listen",
			},
			&cli.StringFlag{
				Name:    "metrics",
				Aliases: []string{"M"},
				Value:   ":2112",
				Usage:   "Host+port to server prometheus metrics",
			},
			&cli.BoolFlag{
				Name:    "reflection",
				Aliases: []string{"r"},
				Value:   false,
				Usage:   "Enable gRPC reflection",
			},
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Required: true,
				Usage:    "Configuration file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			cmdutil.InitLogger(cCtx.String("log"))

			//specific(ethereum)
			//if slog.Default().Enabled(cCtx.Context, slog.LevelDebug) {
			//	ethereum_log.Root().SetHandler(ethereum_log.StreamHandler(os.Stdout, ethereum_log.LogfmtFormat()))
			//}
			// enable eth compatible chains
			agent.AgentFactories[bnb.CODE_STR] = agent.AgentFactories[eth.CODE_STR]
			//end

			supportedChains := []string{}
			for k := range blockchain.Blockchains {
				supportedChains = append(supportedChains, k)
			}
			slog.Debug(fmt.Sprintf("Supported chains: %v", supportedChains))

			lis, err := net.Listen("tcp", cCtx.String("listen"))
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			config := agent.LoadConfig(cCtx.String("config"))
			slog.Debug(fmt.Sprintf("Config: %+v", config))

			srv := InitServerProxy(config.Chains)

			grpcPanicRecoveryHandler := func(p any) (err error) {
				panic(err)
				//return status.Errorf(codes.Internal, "%s", p)
			}

			s := grpc.NewServer(
				grpc.ChainUnaryInterceptor(
					// Order matters e.g. tracing interceptor have to create span first for the later exemplars to work.
					logging.UnaryServerInterceptor(cmdutil.InterceptorLogger(nil)),
					recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
				),
				grpc.ChainStreamInterceptor(
					logging.StreamServerInterceptor(cmdutil.InterceptorLogger(nil)),
					recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
				),
			)
			services.RegisterUbtChainServiceServer(s, srv)
			services.RegisterUbtBlockServiceServer(s, srv)
			services.RegisterUbtCurrencyServiceServer(s, srv)
			services.RegisterUbtConstructServiceServer(s, srv)

			if cCtx.Bool("reflection") {
				slog.Info("Enabling gRPC reflection")
				reflection.Register(s)
			}
			slog.Info(fmt.Sprintf("API listening at %v", lis.Addr()))

			go startMetricsListener(cCtx.String("metrics"))

			if err := s.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startMetricsListener(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	slog.Info(fmt.Sprintf("Metrics listening at %v", addr))
	http.ListenAndServe(addr, nil)
}

func InitServerProxy(configs map[string]agent.ChainTypeConfig) *proxy.ServerProxy {
	servers := make(map[string]agent.UbtAgent)
	for k, v := range configs {
		var foundAgents []struct {
			factory agent.UbtAgentFactory
			config  agent.ChainConfig
		}
		for nk, nv := range v.Networks {
			nv.ChainType = k
			nv.ChainNetwork = nk

			factory, ok := agent.AgentFactories[k]
			if !ok {
				slog.Error(fmt.Sprintf("Unsupported chain type '%s', ignoring", k))
				continue
			}
			foundAgents = append(foundAgents, struct {
				factory agent.UbtAgentFactory
				config  agent.ChainConfig
			}{factory: factory, config: nv})
		}
		if len(foundAgents) == 0 {
			log.Fatalf("No agents found for chain type '%s', ignoring", k)
		}
		for _, agentFactory := range foundAgents {
			agent := agentFactory.factory(context.Background(), &agentFactory.config)
			servers[agentFactory.config.ChainType+":"+agentFactory.config.ChainNetwork] = agent
		}

	}
	return proxy.InitServerProxy(servers)
}
