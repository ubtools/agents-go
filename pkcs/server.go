package pkcs

import (
	"context"
	"crypto/ecdsa"
	"io"

	"github.com/ThalesIgnite/crypto11"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/blockchain/eth"
	ubt_am "github.com/ubtr/ubt/go/api/proto/services/am"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AMPKCSServer struct {
	io.Closer
	ubt_am.UnimplementedUbtAccountManagerServer
	pkcsCtx *crypto11.Context
}

type Config struct {
	Path       string
	TokenLabel string
	Pin        string
}

func InitAMPKCSServer(conf Config) (*AMPKCSServer, error) {
	c, err := crypto11.Configure(&crypto11.Config{
		Path:       conf.Path,
		TokenLabel: conf.TokenLabel,
		Pin:        conf.Pin,
	})
	if err != nil {
		return nil, err
	}
	return &AMPKCSServer{pkcsCtx: c}, nil
}

func (s *AMPKCSServer) Close() error {
	return s.pkcsCtx.Close()
}

func toKeyId(networkType string, name string) string {
	return networkType + ":" + name
}

func (s *AMPKCSServer) CreateAccount(ctx context.Context, req *ubt_am.CreateAccountRequest) (*ubt_am.CreateAccountResponse, error) {
	signer, err := s.pkcsCtx.GenerateECDSAKeyPair([]byte(toKeyId(req.ChainType, req.Name)), crypto.S256())
	if err != nil {
		panic(err)
	}
	pubKey := signer.Public().(*ecdsa.PublicKey)
	addr := eth.AddressFromPublicKey(pubKey.X.Bytes())

	return &ubt_am.CreateAccountResponse{Address: addr.String()}, status.Errorf(codes.Unimplemented, "method CreateAccount not implemented")
}

func (s *AMPKCSServer) GetAccount(ctx context.Context, req *ubt_am.GetAccountRequest) (*ubt_am.GetAccountResponse, error) {
	signer, err := s.pkcsCtx.FindKeyPair([]byte(toKeyId("TBD", req.Name)), nil)
	if err != nil {
		return nil, err
	}

	if signer != nil {
		bc := blockchain.GetBlockchain("ETH")
		if bc == nil {
			return nil, status.Errorf(codes.Unimplemented, "Unsupported chain type '%s'", "ETH")
		}

		address, err := bc.RecoverAddress(signer.Public().(*ecdsa.PublicKey).X.Bytes(), nil)
		if err != nil {
			return nil, err
		}

		return &ubt_am.GetAccountResponse{Address: address}, nil
	}

	return nil, status.Errorf(codes.NotFound, "Account not found")
}

func (s *AMPKCSServer) ListAccounts(context.Context, *ubt_am.ListAccountsRequest) (*ubt_am.ListAccountsResponse, error) {
	signers, err := s.pkcsCtx.FindAllKeyPairs()
	if err != nil {
		return nil, err
	}
	var accounts []*ubt_am.ListAccountsResponse_Account
	for _, signer := range signers {
		pubKey := signer.Public().(*ecdsa.PublicKey)
		addr := eth.AddressFromPublicKey(pubKey.X.Bytes())
		accounts = append(accounts, &ubt_am.ListAccountsResponse_Account{
			Name:    pubKey.Params().Name,
			Address: addr.String(),
		})
	}
	return &ubt_am.ListAccountsResponse{Accounts: accounts}, nil
}

func (s *AMPKCSServer) SignPayload(ctx context.Context, req *ubt_am.SignPayloadRequest) (*ubt_am.SignPayloadResponse, error) {
	signer, err := s.pkcsCtx.FindKeyPair([]byte(toKeyId("TBD", req.Name)), nil)
	if err != nil {
		return nil, err
	}
	randReader, err := s.pkcsCtx.NewRandomReader()
	if err != nil {
		return nil, err
	}
	signer.Sign(randReader, req.Data, nil)
	return nil, status.Errorf(codes.Unimplemented, "method SignPayload not implemented")
}
