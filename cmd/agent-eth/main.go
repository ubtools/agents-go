package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ubtr/ubt-go/blockchain"
	_ "github.com/ubtr/ubt-go/blockchain/bnb"
	_ "github.com/ubtr/ubt-go/blockchain/eth"
	_ "github.com/ubtr/ubt-go/blockchain/trx"
	"github.com/ubtr/ubt-go/cmd/cmdutil"
	"github.com/ubtr/ubt-go/eth/config"
	"github.com/ubtr/ubt-go/eth/server"
	"github.com/ubtr/ubt-go/proxy"
	"github.com/ubtr/ubt-go/trx"

	"github.com/ubtr/ubt/go/api/proto/services"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	ethereum_log "github.com/ethereum/go-ethereum/log"
)

func slogLevelFromString(lvl string) (programLevel slog.Level) {
	switch strings.ToUpper(lvl) {
	case "DEBUG":
		programLevel = slog.LevelDebug
	case "INFO":
		programLevel = slog.LevelInfo
	case "WARN":
		programLevel = slog.LevelWarn
	case "ERROR":
		programLevel = slog.LevelError
	default:
		log.Fatalf("Invalid log level %s", lvl)
	}
	return programLevel
}

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
			if slog.Default().Enabled(cCtx.Context, slog.LevelDebug) {
				ethereum_log.Root().SetHandler(ethereum_log.StreamHandler(os.Stdout, ethereum_log.LogfmtFormat()))
			}
			//end

			supportedChains := []string{}
			for k := range blockchain.Blockchains {
				supportedChains = append(supportedChains, k)
			}
			slog.Debug("Supported chains", "chains", supportedChains)

			lis, err := net.Listen("tcp", cCtx.String("listen"))
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			config := config.LoadConfig(cCtx.String("config"))
			log.Printf("Config: %+v", config)

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
			log.Printf("Listening at %v", lis.Addr())

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

func InitServerProxy(configs map[string]config.ChainTypeConfig) *proxy.ServerProxy {
	servers := make(map[string]proxy.IUbtAgentServer)
	for k, v := range configs {
		for nk, nv := range v.Networks {
			nv.ChainType = k
			nv.ChainNetwork = nk
			var ethSrv proxy.IUbtAgentServer
			// TODO: use mapping
			if nv.ChainType == "TRX" {
				ethSrv = trx.InitServer(context.Background(), &nv)
			} else {
				ethSrv = server.InitServer(context.Background(), &nv)
			}
			servers[k+":"+nk] = ethSrv
		}
	}
	return proxy.InitServerProxy(servers)
}
