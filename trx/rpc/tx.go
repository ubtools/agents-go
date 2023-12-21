package rpc

import (
	"errors"
	"math/big"

	trxcommon "github.com/ubtr/ubt-go/trx/common"
	trxhexutil "github.com/ubtr/ubt-go/trx/common/hexutil"

	"github.com/ethereum/go-ethereum/core/types"
)

type TrxTx struct {
	Type trxhexutil.Uint64 `json:"type"`

	ChainID              *trxhexutil.Big    `json:"chainId,omitempty"`
	Nonce                *trxhexutil.Uint64 `json:"nonce"`
	To                   *trxcommon.Address `json:"to"`
	Gas                  *trxhexutil.Uint64 `json:"gas"`
	GasPrice             *trxhexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *trxhexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *trxhexutil.Big    `json:"maxFeePerGas"`
	MaxFeePerBlobGas     *trxhexutil.Big    `json:"maxFeePerBlobGas,omitempty"`
	Value                *trxhexutil.Big    `json:"value"`
	Input                *trxhexutil.Bytes  `json:"input"`
	AccessList           *types.AccessList  `json:"accessList,omitempty"`
	BlobVersionedHashes  []trxcommon.Hash   `json:"blobVersionedHashes,omitempty"`
	V                    *trxhexutil.Big    `json:"v"`
	R                    *trxhexutil.Big    `json:"r"`
	S                    *trxhexutil.Big    `json:"s"`
	YParity              *trxhexutil.Uint64 `json:"yParity,omitempty"`

	// Only used for encoding:
	Hash trxcommon.Hash `json:"hash"`
}

// yParityValue returns the YParity value from JSON. For backwards-compatibility reasons,
// this can be given in the 'v' field or the 'yParity' field. If both exist, they must match.
func (tx *TrxTx) yParityValue() (*big.Int, error) {
	if tx.YParity != nil {
		val := uint64(*tx.YParity)
		if val != 0 && val != 1 {
			return nil, errors.New("'yParity' field must be 0 or 1")
		}
		bigval := new(big.Int).SetUint64(val)
		if tx.V != nil && tx.V.ToInt().Cmp(bigval) != 0 {
			return nil, errors.New("'v' and 'yParity' fields do not match")
		}
		return bigval, nil
	}
	if tx.V != nil {
		return tx.V.ToInt(), nil
	}
	return nil, errors.New("missing 'yParity' or 'v' field in transaction")
}

// MarshalJSON marshals as JSON with a hash.

// UnmarshalJSON unmarshals from JSON.
