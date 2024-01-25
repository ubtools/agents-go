package blockchain

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// chain identifier, corresponds to proto.ChainId
type UChainId struct {
	Type    string
	Network string
}

func (c *UChainId) String() string {
	if c.Network == "" {
		return c.Type
	}
	return c.Type + ":" + c.Network
}

func (c *UChainId) Normalize() UChainId {
	net := strings.ToUpper(c.Network)
	if net == MAINNET {
		net = ""
	}
	return UChainId{
		Type:    strings.ToUpper(c.Type),
		Network: net,
	}
}

const MAINNET = "MAINNET"

// unified cross-chain currency id
// format: chainType[:chainNetwork[:address[:token]]]
// special case for MAINNET network - it is normalized to empty string
// so ETH currency on mainnet is just "ETH", USDT on ETH mainnet is "ETH::0xdac17f958d2ee523a2206206994597c13d831ec7"
type UCurrencyId struct {
	Chain      UChainId
	CurrencyId UChainCurrencyId
}

func UCurrencyIdFromString(currencyId string) (UCurrencyId, error) {
	parts := strings.Split(currencyId, ":")
	var ret UCurrencyId
	curLen := len(parts)
	if curLen > 4 || curLen == 0 {
		return ret, status.Errorf(codes.InvalidArgument, "invalid currency id: %s", currencyId)
	}

	if curLen >= 1 {
		ret.Chain.Type = strings.ToUpper(parts[0])
	}
	if curLen >= 2 {
		ret.Chain.Network = strings.ToUpper(parts[1])
		// normalize mainnet to empty string
		if ret.Chain.Network == MAINNET {
			ret.Chain.Network = ""
		}
	}
	if curLen >= 3 {
		ret.CurrencyId.Address = parts[2]
	}
	if curLen == 4 {
		ret.CurrencyId.Token = parts[3]
	}

	return ret, nil
}

func (c *UCurrencyId) IsNative() bool {
	return c.CurrencyId.IsNative()
}

func (c *UCurrencyId) IsERC20() bool {
	return c.CurrencyId.Address != "" && c.CurrencyId.Token == ""
}

func (c *UCurrencyId) Normalize() UCurrencyId {
	return UCurrencyId{
		Chain:      c.Chain.Normalize(),
		CurrencyId: c.CurrencyId,
	}
}

func (c *UCurrencyId) String() string {
	ret := c.Chain.Type
	if c.Chain.Network != "" {
		ret += ":" + c.Chain.Network
	}
	curId := c.CurrencyId.String()
	if curId != "" {
		ret += ":" + curId
	}
	return ret
}

// currency id within some chain
type UChainCurrencyId struct {
	Address string
	Token   string
}

var NATIVE_CURRENCY = UChainCurrencyId{}

func UChainCurrencyIdromString(currencyId string) (UChainCurrencyId, error) {
	stringParts := strings.Split(currencyId, ":")
	var ret UChainCurrencyId
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

func (c *UChainCurrencyId) String() string {
	ret := c.Address
	if c.Token != "" {
		ret += ":" + c.Token
	}
	return ret
}

func (c *UChainCurrencyId) IsNative() bool {
	return c.Address == ""
}

func (c *UChainCurrencyId) IsErc20() bool {
	return c.Address != "" && c.Token == ""
}

func (c *UChainCurrencyId) IsErc1155() bool {
	return c.Address != "" && c.Token != ""
}
