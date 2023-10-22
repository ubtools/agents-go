package main

import (
	"context"
	"crypto/ecdsa"
	"io"

	"github.com/ThalesIgnite/crypto11"
	"github.com/ethereum/go-ethereum/crypto"
	ubt_am "github.com/ubtools/ubt/go/api/proto/services/am"
	"github.com/ubtools/ubt/go/blockchain/eth"
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
	signer, err := s.pkcsCtx.GenerateECDSAKeyPair([]byte(toKeyId(req.NetworkType, req.Name)), crypto.S256())
	if err != nil {
		panic(err)
	}
	pubKey := signer.Public().(*ecdsa.PublicKey)
	addr := eth.AddressFromPublicKey(pubKey.X.Bytes())

	return &ubt_am.CreateAccountResponse{Address: addr.String()}, status.Errorf(codes.Unimplemented, "method CreateAccount not implemented")
}

func (s *AMPKCSServer) HasAccount(ctx context.Context, req *ubt_am.HasAccountRequest) (*ubt_am.HasAccountResponse, error) {
	signer, err := s.pkcsCtx.FindKeyPair([]byte(toKeyId("TBD", req.Name)), nil)
	if err != nil {
		return nil, err
	}
	return &ubt_am.HasAccountResponse{Exists: signer != nil && err != nil}, nil
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
	signer.Sign(s.pkcsCtx.NewRandomReader())
	return nil, status.Errorf(codes.Unimplemented, "method SignPayload not implemented")
}
