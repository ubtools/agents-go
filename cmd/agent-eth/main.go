package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"ubt/agents/eth/config"
	"ubt/agents/eth/server"
	"ubt/agents/trx"

	"github.com/ubtools/ubt/go/api/proto/services"

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

// interceptorLogger adapts go-kit logger to interceptor logger.
func interceptorLogger(logger *slog.Logger) logging.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg, fields...)
		case logging.LevelInfo:
			logger.Info(msg, fields...)
		case logging.LevelWarn:
			logger.Warn(msg, fields...)
		case logging.LevelError:
			logger.Debug(msg, fields...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
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
			var lvl = slogLevelFromString(cCtx.String("log"))
			h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: lvl == slog.LevelDebug, Level: lvl})
			slog.SetDefault(slog.New(h))

			//specific(ethereum)
			if lvl == slog.LevelDebug {
				ethereum_log.Root().SetHandler(ethereum_log.StreamHandler(os.Stdout, ethereum_log.LogfmtFormat()))
			}
			//end

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
					logging.UnaryServerInterceptor(interceptorLogger(nil)),
					recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
				),
				grpc.ChainStreamInterceptor(
					logging.StreamServerInterceptor(interceptorLogger(nil)),
					recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
				),
			)
			services.RegisterUbtChainServiceServer(s, srv)
			services.RegisterUbtBlockServiceServer(s, srv)
			services.RegisterUbtCurrencyServiceServer(s, srv)
			services.RegisterUbtConstructServiceServer(s, srv)

			if cCtx.Bool("reflection") {
				// enable reflection
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

func InitServerProxy(configs map[string]config.ChainTypeConfig) *server.ServerProxy {
	servers := make(map[string]server.IUbtAgentServer)
	for k, v := range configs {
		for nk, nv := range v.Networks {
			nv.ChainType = k
			nv.ChainNetwork = nk
			var ethSrv server.IUbtAgentServer
			// TODO: use mapping
			if nv.ChainType == "TRX" {
				ethSrv = trx.InitServer(context.Background(), &nv)
			} else {
				ethSrv = server.InitServer(context.Background(), &nv)
			}
			servers[k+":"+nk] = ethSrv
		}
	}
	return server.InitServerProxy(servers)
}
