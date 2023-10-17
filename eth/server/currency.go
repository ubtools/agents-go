package server

import (
	"context"
	"strings"
	"ubt/agents/eth/contracts/erc20"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ETH_DECIMALS = 18

type CurrencyId struct {
	Address string
	Token   string
}

func CurrencyIdFromString(currencyId string) (CurrencyId, error) {
	stringParts := strings.Split(currencyId, ":")
	var ret CurrencyId
	if len(stringParts) == 2 {
		ret.Address = stringParts[0]
		ret.Token = stringParts[1]
	} else if len(stringParts) == 1 {
		ret.Address = stringParts[0]
		ret.Token = ""
	} else if len(stringParts) > 2 {
		return ret, status.Errorf(codes.InvalidArgument, "invalid currency id: %s", currencyId)
	}

	return ret, nil
}

func (c *CurrencyId) String() string {
	res := c.Address
	if c.Token != "" {
		res += ":" + c.Token
	}
	return res
}

func (c *CurrencyId) IsNative() bool {
	return c.Address == ""
}

func (c *CurrencyId) IsErc20() bool {
	return c.Address != "" && c.Token == ""
}

func (c *CurrencyId) IsErc1155() bool {
	return c.Address != "" && c.Token != ""
}

func (srv *EthServer) GetCurrency(ctx context.Context, req *services.GetCurrencyRequest) (*proto.Currency, error) {
	// naive implementation, no cache
	currencyId, err := CurrencyIdFromString(req.Id)
	if err != nil {
		return nil, err
	}
	if currencyId.Address == "" {
		// native currency
		return &proto.Currency{
			Id:       req.Id,
			Symbol:   srv.config.ChainType,
			Decimals: ETH_DECIMALS,
		}, nil
	} else if currencyId.Token == "" {
		// erc20 token
		// retreive token info
		tokenInst, err := erc20.NewErc20(common.HexToAddress(currencyId.Address), srv.c)
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
