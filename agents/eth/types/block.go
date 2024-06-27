package ethtypes

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt-go/commons"
)

type RpcTx struct {
	TxHash common.Hash
	Tx     *types.Transaction
	TxExtraInfo
}

func (tx *RpcTx) UnmarshalJSON(input []byte) error {
	fixedInput, err := commons.FixJsonFields(input, true,
		[]string{"nonce"}, commons.FixerHexStripLeadingZeros,
		[]string{"r"}, commons.FixerHexStripLeadingZeros,
		[]string{"s"}, commons.FixerHexStripLeadingZeros,
		[]string{"v"}, commons.FixerHexStripLeadingZeros)
	if err != nil {
		return err
	}

	txHash := struct {
		Value common.Hash `json:"hash"`
	}{}
	if err := json.Unmarshal(fixedInput, &txHash); err != nil {
		return err
	}
	tx.TxHash = txHash.Value

	if err := json.Unmarshal(fixedInput, &tx.Tx); err != nil {
		return err
	}
	return json.Unmarshal(fixedInput, &tx.TxExtraInfo)
}

type RpcBody struct {
	Transactions []*RpcTx
}

type HeaderWithBody struct {
	BlockHash common.Hash
	Header    types.Header
	Body      RpcBody
}

func (b *HeaderWithBody) UnmarshalJSON(input []byte) error {
	fixedInput, err := commons.FixJsonFields(input, true, []string{"stateRoot"}, commons.FixerZeroHash)
	if err != nil {
		return err
	}

	blockHash := struct {
		BlockHash common.Hash `json:"hash"`
	}{}
	if err := json.Unmarshal(fixedInput, &blockHash); err != nil {
		return err
	}
	b.BlockHash = blockHash.BlockHash

	if err := b.Header.UnmarshalJSON(fixedInput); err != nil {
		return err
	}

	var txStruct struct {
		Transactions []*RpcTx `json:"transactions"`
	}

	if err := json.Unmarshal(fixedInput, &txStruct); err != nil {
		return err
	}
	b.Body.Transactions = txStruct.Transactions

	return nil
}

type TxExtraInfo struct {
	BlockNumber      *string         `json:"blockNumber,omitempty"`
	BlockHash        *common.Hash    `json:"blockHash,omitempty"`
	From             *common.Address `json:"from,omitempty"`
	TransactionIndex hexutil.Uint64  `json:"transactionIndex"`
}
