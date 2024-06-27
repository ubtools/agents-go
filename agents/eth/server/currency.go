package server

import (
	"context"

	"github.com/eko/gocache/lib/v4/store"
	"github.com/ubtr/ubt-go/agents/eth/contracts/erc20"
	"github.com/ubtr/ubt-go/agents/eth/rpc"
	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/commons/rpcerrors"

	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (srv *EthServer) GetCurrency(ctx context.Context, req *services.GetCurrencyRequest) (*proto.Currency, error) {
	srv.Log.Debug("GetCurrency", "req", req)

	// naive implementation, no cache
	currencyId, err := blockchain.UChainCurrencyIdromString(req.Id)
	if err != nil {
		return nil, err
	}
	srv.Log.Debug("Parsed", "currencyId", currencyId)
	if currencyId.Address == "" {

		// native currency
		return &proto.Currency{
			Id:       req.Id,
			Symbol:   srv.Config.ChainType,
			Decimals: uint32(srv.Chain.Decimals),
		}, nil
	} else if currencyId.Token == "" {
		cached, err := srv.CurrencyCache.Get(ctx, req.Id)
		if err == nil {
			srv.Log.Debug("Currency cache hit", "currencyId", req.Id)
			return cached, nil
		} else {
			srv.Log.Debug("Currency cache miss", "currencyId", req.Id)
		}
		// erc20 token
		// retreive token info
		addr, err := srv.AddressFromString(currencyId.Address)
		if err != nil {
			return nil, err
		}
		tokenInst, err := erc20.NewErc20(addr, rpc.AdoptClient(srv.C))
		if err != nil {
			srv.Log.Error("Failed to create token instance", "err", err)
			return nil, rpcerrors.ErrInvalidCurrency
		}
		symbol, err := tokenInst.Symbol(nil)
		if err != nil {
			srv.Log.Error("Failed to get token symbol", "err", err)
			return nil, rpcerrors.ErrInvalidCurrency
		}
		decimals, err := tokenInst.Decimals(nil)
		if err != nil {
			srv.Log.Error("Failed to get token decimals", "err", err)
			return nil, rpcerrors.ErrInvalidCurrency
		}
		var ret = &proto.Currency{
			Id:       req.Id,
			Symbol:   symbol,
			Decimals: uint32(decimals),
		}
		srv.CurrencyCache.Set(ctx, req.Id, ret, store.WithCost(1))
		return ret, nil
	} else {
		// erc-1155 token
		return nil, status.Errorf(codes.Unimplemented, "ERC-1155 not supported yet")
	}

}
