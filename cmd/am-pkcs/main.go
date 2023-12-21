package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	am_pkcs "github.com/ubtr/ubt-go/pkcs"

	"github.com/ThalesIgnite/crypto11"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	ubt_am "github.com/ubtr/ubt/go/api/proto/services/am"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
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

func randomBytes() []byte {
	result := make([]byte, 32)
	rand.Read(result)
	return result
}

func readConfig(configLocation string) *crypto11.Config {
	if configLocation == "" {
		configLocation = "config.yaml"
	}
	var conf crypto11.Config
	data, err := os.ReadFile(configLocation)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	err = yaml.Unmarshal([]byte(data), &conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return &conf
}

func initPKCSContext(configLocation string) *crypto11.Context {
	conf := readConfig(configLocation)
	//conf.UseGCMIVFromHSM = true
	//conf.UserType = crypto11.CryptoUser

	slog.Info("Config", "conf", conf)
	ctx, err := crypto11.Configure(conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("PKCS11 context initialized: %v\n", ctx)
	return ctx
}

func main() {
	app := &cli.App{
		Name:            "ubt-am-pkcs",
		Usage:           "UBT Account Manager (PKCS11)",
		HideHelpCommand: true,
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the UBT Account Manager server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "log",
						Value: "INFO",
						Usage: "Log level",
					},
					&cli.StringFlag{
						Name:  "listen",
						Value: ":50052",
						Usage: "Host+port to listen",
					},
					&cli.StringFlag{
						Name:  "db",
						Value: "am.db",
						Usage: "Database connection string",
					},
				},
				Action: func(cCtx *cli.Context) error {
					var lvl = slogLevelFromString(cCtx.String("log"))
					h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
					slog.SetDefault(slog.New(h))

					lis, err := net.Listen("tcp", cCtx.String("listen"))
					if err != nil {
						log.Fatalf("failed to listen: %v", err)
					}
					srv, err := am_pkcs.InitAMPKCSServer(am_pkcs.Config{})
					if err != nil {
						slog.Error("Failed to start server", "err", err)
						panic(err)
					}
					defer srv.Close()

					s := grpc.NewServer()
					ubt_am.RegisterUbtAccountManagerServer(s, srv)
					log.Printf("server listening at %v", lis.Addr())

					if err := s.Serve(lis); err != nil {
						log.Fatalf("failed to serve: %v", err)
					}
					return nil
				},
			},
			{
				Name:  "gen",
				Usage: "Generate a new key pair",
				Flags: []cli.Flag{},
				Action: func(cCtx *cli.Context) error {
					pkcs := initPKCSContext("")

					s, err := pkcs.GenerateECDSAKeyPairWithLabel(randomBytes(), []byte("test"), secp256k1.S256())
					//s, err := pkcs.GenerateECDSAKeyPair([]byte("test"), crypto.S256())
					if err != nil {
						return err
					}
					fmt.Println(s)
					return nil
				},
			},
			{
				Name:  "list",
				Usage: "List all key pairs",
				Flags: []cli.Flag{},
				Action: func(cCtx *cli.Context) error {
					fmt.Println("list")
					pkcs := initPKCSContext("")
					keys, err := pkcs.FindAllKeyPairs()
					if err != nil {
						return err
					}
					fmt.Printf("keys: %d\n", len(keys))
					for _, key := range keys {
						fmt.Println(key)
						//fmt.Println(key.Public().(*ecdsa.PublicKey).Params().Name)
					}

					return nil
				},
			},
			{
				Name:  "sign",
				Usage: "Sign arbitrary data using stored key",
				Flags: []cli.Flag{},
				Action: func(cCtx *cli.Context) error {
					fmt.Println("sign")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
