package uint256conv

import (
	"math/big"

	"github.com/ubtr/ubt-go/commons/conv/hexconv"
	"github.com/ubtr/ubt/go/api/proto"
)

const MAX_UINT256 = "115792089237316195423570985008687907853269984665640564039457584007913129639935" // 2^256-1
const MAX_UINT256_HEX = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

var MAX_UINT256_BIGINT = new(big.Int).SetBytes(hexconv.FromHex(MAX_UINT256_HEX))

func ToBigInt(val *proto.Uint256) *big.Int {
	if val == nil {
		return nil
	}
	return big.NewInt(0).SetBytes(val.Data)
}

func FromBigInt(val *big.Int) *proto.Uint256 {
	if val == nil {
		return nil
	}
	if val.Cmp(MAX_UINT256_BIGINT) > 0 {
		panic("value is too large")
	}
	return &proto.Uint256{Data: val.Bytes()}
}
