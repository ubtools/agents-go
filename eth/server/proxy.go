package server

import (
	"context"
	"errors"
	"ubt/agents/commons"

	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
)

type IUbtAgentServer interface {
	services.UbtChainServiceServer
	services.UbtBlockServiceServer
	services.UbtConstructServiceServer
	services.UbtCurrencyServiceServer
}

type ServerProxy struct {
	services.UnimplementedUbtChainServiceServer
	services.UnimplementedUbtBlockServiceServer
	services.UnimplementedUbtConstructServiceServer
	services.UnimplementedUbtCurrencyServiceServer

	servers map[string]IUbtAgentServer
}

func InitServerProxy(servers map[string]IUbtAgentServer) *ServerProxy {
	var srv = ServerProxy{servers: servers}
	return &srv
}

var ErrChainNotSupported = errors.New("chain not supported")

func (s *ServerProxy) GetChain(ctx context.Context, in *proto.ChainId) (*proto.Chain, error) {
	chainId := commons.ChainIdToString(in)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetChain(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) ListChains(in *services.ListChainsRequest, srv services.UbtChainService_ListChainsServer) error {
	//FIXME
	return ErrChainNotSupported
}

func (s *ServerProxy) GetBlock(ctx context.Context, in *services.BlockRequest) (*proto.Block, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetBlock(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) ListBlocks(in *services.ListBlocksRequest, res services.UbtBlockService_ListBlocksServer) error {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.ListBlocks(in, res)
	}
	return ErrChainNotSupported
}

func (s *ServerProxy) GetAccount(ctx context.Context, in *services.GetAccountRequest) (*proto.Account, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetAccount(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) DeriveAccount(ctx context.Context, in *services.DeriveAccountRequest) (*proto.Account, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.DeriveAccount(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) GetCurrency(ctx context.Context, in *services.GetCurrencyRequest) (*proto.Currency, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetCurrency(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) CreateTransfer(ctx context.Context, in *services.CreateTransferRequest) (*services.TransactionIntent, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.CreateTransfer(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) CombineTransaction(ctx context.Context, in *services.TransactionCombineRequest) (*services.SignedTransaction, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.CombineTransaction(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) Send(ctx context.Context, in *services.TransactionSendRequest) (*services.TransactionSendResponse, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.Send(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) SignTransaction(ctx context.Context, in *services.TransactionSignRequest) (*services.SignedTransaction, error) {
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.SignTransaction(ctx, in)
	}
	return nil, ErrChainNotSupported
}
