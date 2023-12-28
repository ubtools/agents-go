package proxy

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ubtr/ubt-go/commons"

	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
)

type IUbtAgentServer interface {
	services.UbtChainServiceServer
	services.UbtBlockServiceServer
	services.UbtConstructServiceServer
	services.UbtCurrencyServiceServer
	String() string
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
var ErrChainIdRequired = errors.New("chain id is required")

func (s *ServerProxy) GetChain(ctx context.Context, in *proto.ChainId) (*proto.Chain, error) {
	if in == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in)
	slog.Debug("GetChain", "chainId", chainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetChain(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) ListChains(in *services.ListChainsRequest, srv services.UbtChainService_ListChainsServer) error {
	for sn, downSrv := range s.servers {
		slog.Debug("ss", "sn", sn, "downSrv", downSrv.String())
		err := downSrv.ListChains(in, srv)
		if err != nil {
			slog.Debug("err", "err", err)
			return err
		}
	}
	return nil
}

func (s *ServerProxy) GetBlock(ctx context.Context, in *services.BlockRequest) (*proto.Block, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetBlock(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) ListBlocks(in *services.ListBlocksRequest, res services.UbtBlockService_ListBlocksServer) error {
	if in.ChainId == nil {
		return ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	slog.Debug("ListBlocks", "chainId", chainId)
	if srv, ok := s.servers[chainId]; ok {
		slog.Debug("ListBlocks", "server", srv.String())
		return srv.ListBlocks(in, res)
	}
	return ErrChainNotSupported
}

func (s *ServerProxy) GetAccount(ctx context.Context, in *services.GetAccountRequest) (*proto.Account, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetAccount(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) DeriveAccount(ctx context.Context, in *services.DeriveAccountRequest) (*proto.Account, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.DeriveAccount(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) GetCurrency(ctx context.Context, in *services.GetCurrencyRequest) (*proto.Currency, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.GetCurrency(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) CreateTransfer(ctx context.Context, in *services.CreateTransferRequest) (*services.TransactionIntent, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.CreateTransfer(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) CombineTransaction(ctx context.Context, in *services.TransactionCombineRequest) (*services.SignedTransaction, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.CombineTransaction(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) Send(ctx context.Context, in *services.TransactionSendRequest) (*services.TransactionSendResponse, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.Send(ctx, in)
	}
	return nil, ErrChainNotSupported
}

func (s *ServerProxy) SignTransaction(ctx context.Context, in *services.TransactionSignRequest) (*services.SignedTransaction, error) {
	if in.ChainId == nil {
		return nil, ErrChainIdRequired
	}
	chainId := commons.ChainIdToString(in.ChainId)
	if srv, ok := s.servers[chainId]; ok {
		return srv.SignTransaction(ctx, in)
	}
	return nil, ErrChainNotSupported
}
