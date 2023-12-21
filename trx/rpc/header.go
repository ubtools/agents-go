package rpc

import (
	"encoding/json"
	"errors"
	"math/big"

	trxcommon "github.com/ubtr/ubt-go/trx/common"
	trxhexutil "github.com/ubtr/ubt-go/trx/common/hexutil"

	"github.com/ethereum/go-ethereum/core/types"
)

type Header struct {
	Hash        trxcommon.Hash    `json:"hash"       gencodec:"required"`
	ParentHash  trxcommon.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   trxcommon.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    trxcommon.Address `json:"miner"`
	Root        trxcommon.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      trxcommon.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash trxcommon.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       types.Bloom       `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int          `json:"difficulty"       gencodec:"required"`
	Number      *big.Int          `json:"number"           gencodec:"required"`
	GasLimit    uint64            `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64            `json:"gasUsed"          gencodec:"required"`
	Time        uint64            `json:"timestamp"        gencodec:"required"`
	Extra       []byte            `json:"extraData"        gencodec:"required"`
	MixDigest   trxcommon.Hash    `json:"mixHash"`
	Nonce       types.BlockNonce  `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`

	// WithdrawalsHash was added by EIP-4895 and is ignored in legacy headers.
	WithdrawalsHash *trxcommon.Hash `json:"withdrawalsRoot" rlp:"optional"`

	// BlobGasUsed was added by EIP-4844 and is ignored in legacy headers.
	BlobGasUsed *uint64 `json:"blobGasUsed" rlp:"optional"`

	// ExcessBlobGas was added by EIP-4844 and is ignored in legacy headers.
	ExcessBlobGas *uint64 `json:"excessBlobGas" rlp:"optional"`

	// ParentBeaconRoot was added by EIP-4788 and is ignored in legacy headers.
	ParentBeaconRoot *trxcommon.Hash `json:"parentBeaconBlockRoot" rlp:"optional"`
}

// UnmarshalJSON unmarshals from JSON.
func (h *Header) UnmarshalJSON(input []byte) error {
	type Header struct {
		Hash             *trxcommon.Hash    `json:"hash"       gencodec:"required"`
		ParentHash       *trxcommon.Hash    `json:"parentHash"       gencodec:"required"`
		UncleHash        *trxcommon.Hash    `json:"sha3Uncles"       gencodec:"required"`
		Coinbase         *trxcommon.Address `json:"miner"`
		Root             *trxcommon.Hash    `json:"stateRoot"        gencodec:"required"`
		TxHash           *trxcommon.Hash    `json:"transactionsRoot" gencodec:"required"`
		ReceiptHash      *trxcommon.Hash    `json:"receiptsRoot"     gencodec:"required"`
		Bloom            *types.Bloom       `json:"logsBloom"        gencodec:"required"`
		Difficulty       *trxhexutil.Big    `json:"difficulty"       gencodec:"required"`
		Number           *trxhexutil.Big    `json:"number"           gencodec:"required"`
		GasLimit         *trxhexutil.Uint64 `json:"gasLimit"         gencodec:"required"`
		GasUsed          *trxhexutil.Uint64 `json:"gasUsed"          gencodec:"required"`
		Time             *trxhexutil.Uint64 `json:"timestamp"        gencodec:"required"`
		Extra            *trxhexutil.Bytes  `json:"extraData"        gencodec:"required"`
		MixDigest        *trxcommon.Hash    `json:"mixHash"`
		Nonce            *types.BlockNonce  `json:"nonce"`
		BaseFee          *trxhexutil.Big    `json:"baseFeePerGas" rlp:"optional"`
		WithdrawalsHash  *trxcommon.Hash    `json:"withdrawalsRoot" rlp:"optional"`
		BlobGasUsed      *trxhexutil.Uint64 `json:"blobGasUsed" rlp:"optional"`
		ExcessBlobGas    *trxhexutil.Uint64 `json:"excessBlobGas" rlp:"optional"`
		ParentBeaconRoot *trxcommon.Hash    `json:"parentBeaconBlockRoot" rlp:"optional"`
	}
	var dec Header
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Hash == nil {
		return errors.New("missing required field 'hash' for Header")
	}
	h.Hash = *dec.Hash
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for Header")
	}
	h.ParentHash = *dec.ParentHash
	if dec.UncleHash == nil {
		return errors.New("missing required field 'sha3Uncles' for Header")
	}
	h.UncleHash = *dec.UncleHash
	if dec.Coinbase != nil {
		h.Coinbase = *dec.Coinbase
	}
	if dec.Root == nil {
		return errors.New("missing required field 'stateRoot' for Header")
	}
	h.Root = *dec.Root
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	h.TxHash = *dec.TxHash
	if dec.ReceiptHash == nil {
		return errors.New("missing required field 'receiptsRoot' for Header")
	}
	h.ReceiptHash = *dec.ReceiptHash
	if dec.Bloom == nil {
		return errors.New("missing required field 'logsBloom' for Header")
	}
	h.Bloom = *dec.Bloom
	if dec.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Header")
	}
	h.Difficulty = (*big.Int)(dec.Difficulty)
	if dec.Number == nil {
		return errors.New("missing required field 'number' for Header")
	}
	h.Number = (*big.Int)(dec.Number)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	h.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	h.GasUsed = uint64(*dec.GasUsed)
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint64(*dec.Time)
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	h.Extra = *dec.Extra
	if dec.MixDigest != nil {
		h.MixDigest = *dec.MixDigest
	}
	if dec.Nonce != nil {
		h.Nonce = *dec.Nonce
	}
	if dec.BaseFee != nil {
		h.BaseFee = (*big.Int)(dec.BaseFee)
	}
	if dec.WithdrawalsHash != nil {
		h.WithdrawalsHash = dec.WithdrawalsHash
	}
	if dec.BlobGasUsed != nil {
		h.BlobGasUsed = (*uint64)(dec.BlobGasUsed)
	}
	if dec.ExcessBlobGas != nil {
		h.ExcessBlobGas = (*uint64)(dec.ExcessBlobGas)
	}
	if dec.ParentBeaconRoot != nil {
		h.ParentBeaconRoot = dec.ParentBeaconRoot
	}
	return nil
}
