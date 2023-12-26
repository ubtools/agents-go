package trx

import (
	"context"

	"github.com/shengdoushi/base58"
	"github.com/ubtr/ubt-go/blockchain"
	rpcerrors "github.com/ubtr/ubt-go/commons/errors"
	"github.com/ubtr/ubt-go/eth/contracts/erc20"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ETH_DECIMALS = 18

func TronAddressToEthAddress(tronAddress string) (common.Address, error) {
	addrB58, err := base58.Decode(tronAddress, base58.BitcoinAlphabet)
	if err != nil {
		return common.Address{}, err
	}
	addrB58 = addrB58[:len(addrB58)-4]
	return common.BytesToAddress(addrB58[len(addrB58)-20:]), nil
}

func (srv *TrxServer) GetCurrency(ctx context.Context, req *services.GetCurrencyRequest) (*proto.Currency, error) {
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
			Decimals: ETH_DECIMALS,
		}, nil
	} else if currencyId.Token == "" {
		// erc20 token
		// retreive token info
		addr, err := TronAddressToEthAddress(currencyId.Address)
		if err != nil {
			return nil, err
		}

		tokenInst, err := erc20.NewErc20(addr, srv.C)
		if err != nil {
			srv.Log.Error("Failed to create token instance", "err", err)
			return nil, rpcerrors.INVALID_CURRENCY
		}
		symbol, err := tokenInst.Symbol(nil)
		if err != nil {
			srv.Log.Error("Failed to get token symbol", "err", err)
			return nil, rpcerrors.INVALID_CURRENCY
		}
		decimals, err := tokenInst.Decimals(nil)
		if err != nil {
			srv.Log.Error("Failed to get token decimals", "err", err)
			return nil, rpcerrors.INVALID_CURRENCY
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
