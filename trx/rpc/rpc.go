package rpc

import (
	"encoding/json"
	"log/slog"

	trxcommon "github.com/ubtr/ubt-go/trx/common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type RpcTx struct {
	Tx *TrxTx
	TxExtraInfo
}

func (tx *RpcTx) UnmarshalJSON(msg []byte) error {
	slog.Debug("+UNRpcTx")
	if tx.Tx == nil {
		tx.Tx = &TrxTx{}
	}
	if err := json.Unmarshal(msg, tx.Tx); err != nil {
		return err
	}
	slog.Debug("-UNRpcTx")
	return json.Unmarshal(msg, &tx.TxExtraInfo)
}

type RpcBody struct {
	Transactions []*RpcTx `json:"transactions"`
}

type HeaderWithBody struct {
	Header Header
	Body   RpcBody
}

func (b *HeaderWithBody) UnmarshalJSON(input []byte) error {
	if err := b.Header.UnmarshalJSON(input); err != nil {
		return err
	}

	var txStruct struct {
		Transactions []*RpcTx `json:"transactions"`
		//Transactions2 []RpcTx  `json:"transactions3"`
	}

	slog.Debug("Suka", "input", string(input))
	if err := json.Unmarshal(input, &txStruct); err != nil {
		return err
	}
	slog.Debug("Suka2")
	b.Body.Transactions = txStruct.Transactions

	return nil
}

type TxExtraInfo struct {
	BlockNumber      *string         `json:"blockNumber,omitempty"`
	BlockHash        *trxcommon.Hash `json:"blockHash,omitempty"`
	From             *common.Address `json:"from,omitempty"`
	TransactionIndex hexutil.Uint64  `json:"transactionIndexx"`
}
