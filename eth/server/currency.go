package server

import (
	"context"

	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/eth/contracts/erc20"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ETH_DECIMALS = 18

func (srv *EthServer) GetCurrency(ctx context.Context, req *services.GetCurrencyRequest) (*proto.Currency, error) {
	// naive implementation, no cache
	currencyId, err := blockchain.UChainCurrencyIdromString(req.Id)
	if err != nil {
		return nil, err
	}
	if currencyId.Address == "" {
		// native currency
		return &proto.Currency{
			Id:       req.Id,
			Symbol:   srv.Config.ChainType,
			Decimals: ETH_DECIMALS,
		}, nil
	} else if currencyId.Token == "" {
		// erc20 token
		// retreive token info
		tokenInst, err := erc20.NewErc20(common.HexToAddress(currencyId.Address), srv.C)
		if err != nil {
			return nil, err
		}
		symbol, err := tokenInst.Symbol(nil)
		if err != nil {
			return nil, err
		}
		decimals, err := tokenInst.Decimals(nil)
		if err != nil {
			return nil, err
		}
		return &proto.Currency{
			Id:       req.Id,
			Symbol:   symbol,
			Decimals: uint32(decimals),
		}, nil
	} else {
		// erc-1155 token
		return nil, status.Errorf(codes.Unimplemented, "ERC-1155 not supported yet")
	}

}
