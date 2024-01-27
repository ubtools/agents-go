package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	ubt_am "github.com/ubtr/ubt/go/api/proto/services/am"
	"gorm.io/gorm"

	_ "github.com/ubtr/ubt-go/blockchain"
	_ "github.com/ubtr/ubt-go/blockchain/bnb"
	_ "github.com/ubtr/ubt-go/blockchain/eth"
	_ "github.com/ubtr/ubt-go/blockchain/trx"
	"github.com/ubtr/ubt-go/cmd/cmdutil"

	am "github.com/ubtr/ubt-go/am"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

func main() {
	app := &cli.App{
		Name:            "ubt-am",
		Usage:           "UBT Account Manager",
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log",
				Value: "INFO",
				Usage: "Log level",
			},
		},
		Before: func(cCtx *cli.Context) error {
			cmdutil.InitLogger(cCtx.String("log"))
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Starts the server",
				Flags: []cli.Flag{

					&cli.StringFlag{
						Name:  "listen",
						Value: ":50052",
						Usage: "Host+port to listen",
					},
					&cli.StringFlag{
						Name:  "db",
						Value: "host=localhost user=postgres password=postgres dbname=am",
						Usage: "Database connection string",
					},
					&cli.StringFlag{
						Name:  "enckey",
						Usage: "Encryption key. If not provided private keys will be stored unencrypted",
					},
				},
				Action: func(cCtx *cli.Context) error {
					lis, err := net.Listen("tcp", cCtx.String("listen"))
					if err != nil {
						log.Fatalf("failed to listen: %v", err)
					}
					srv := am.InitAMServer(cCtx.String("db"), []byte(cCtx.String("enckey")))

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
					ubt_am.RegisterUbtAccountManagerServer(s, srv)
					slog.Info("server listening at %v", "address", lis.Addr())

					if err := s.Serve(lis); err != nil {
						log.Fatalf("failed to serve: %v", err)
					}
					return nil
				},
			},
			{
				Name:  "gen",
				Usage: "Generate a new key pair",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "chain",
						Value: "ETH",
						Usage: "Chain type",
					},
					&cli.StringFlag{
						Name:  "name",
						Usage: "Optional unique name to be associated with this account",
					},
					&cli.StringFlag{
						Name:  "db",
						Value: "host=localhost user=postgres password=postgres dbname=am",
						Usage: "Database connection string",
					},
					&cli.StringFlag{
						Name:  "enckey",
						Usage: "Encryption key. If not provided private keys will be stored unencrypted",
					},
				},
				Action: func(cCtx *cli.Context) error {
					srv := am.InitAMServer(cCtx.String("db"), []byte(cCtx.String("enckey")))
					res, err := srv.CreateAccount(context.TODO(), &ubt_am.CreateAccountRequest{ChainType: cCtx.String("chain"), Name: cCtx.String("name")})
					if err != nil {
						return err
					}
					fmt.Printf("Name: %s\n", cCtx.String("name"))
					fmt.Printf("Address: %s\n", res.Address)
					return nil
				},
			},
			{
				Name:  "sign",
				Usage: "Sign an arbitrary message. Use - from stdin",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db",
						Value: "host=localhost user=postgres password=postgres dbname=am",
						Usage: "Database connection string",
					},
					&cli.StringFlag{
						Name:  "enckey",
						Usage: "Encryption key. If not provided private keys will be stored unencrypted",
					},
				},
				Action: func(cCtx *cli.Context) error {
					return nil
				},
			},
			{
				Name:  "delete",
				Usage: "Delete an account",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db",
						Value: "host=localhost user=postgres password=postgres dbname=am",
						Usage: "Database connection string",
					},
					&cli.BoolFlag{
						Name:  "all",
						Value: false,
						Usage: "Delete all accounts",
					},
				},
				ArgsUsage: "[address]...",
				Action: func(cCtx *cli.Context) error {
					db, err := am.GormOpenRetry(cCtx.String("db"), &gorm.Config{})
					if err != nil {
						return err
					}
					if cCtx.NArg() == 0 {
						if !cCtx.Bool("all") {
							return errors.New("no addresses provided. Specify --all to remove all accounts")
						}
						db.Exec("DELETE FROM accounts")
						return nil
					}
					db.Exec("DELETE FROM accounts WHERE address in ?", cCtx.Args().Slice())
					return nil
				},
			},
			{
				Name:  "list",
				Usage: "List accounts",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "name",
						Value: "",
						Usage: "Name filter",
					},
					&cli.BoolFlag{
						Name:  "export",
						Value: false,
						Usage: "Export private key as base64 if possible",
					},
					&cli.StringFlag{
						Name:  "db",
						Value: "host=localhost user=postgres password=postgres dbname=am",
						Usage: "Database connection string",
					},
					&cli.StringFlag{
						Name:  "enckey",
						Usage: "Encryption key. If not provided private keys will be stored unencrypted",
					},
				},
				Action: func(cCtx *cli.Context) error {
					db, err := am.GormOpenRetry(cCtx.String("db"), &gorm.Config{})
					if err != nil {
						return err
					}
					var accounts []am.Account
					err = db.Where("name LIKE ?", cCtx.String("name")+"%").Find(&accounts).Error
					//err = db.Find(&accounts).Error
					if err != nil {
						return err
					}

					fmt.Println("Address,Name,PK")

					for _, acc := range accounts {
						fmt.Printf("%s,%s", acc.Address, *acc.Name)
						if cCtx.Bool("export") {
							enc := am.GetEncryption([]byte(cCtx.String("enckey")))
							decodedPk, err := enc.Decrypt(acc.PK)
							if err != nil {
								return err
							}
							fmt.Printf(",%s\n", hexutil.Encode(decodedPk))
						} else {
							fmt.Println()
						}
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
