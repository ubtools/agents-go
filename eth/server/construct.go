package server

import (
	"context"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
	"github.com/ubtools/ubt/go/blockchain/eth"
	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (srv *EthServer) CreateTransfer(ctx context.Context, req *services.CreateTransferRequest) (*services.TransactionIntent, error) {
	nonce, err := srv.c.PendingNonceAt(ctx, common.HexToAddress(req.From))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get nonce: %v", err)
	}

	gasPrice, err := srv.c.SuggestGasPrice(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get gas price: %v", err)
	}

	currencyId, err := CurrencyIdFromString(req.Amount.CurrencyId)
	if err != nil {
		return nil, err
	}

	var gasEstimate uint64 = 21000

	var tx *types.Transaction

	slog.Debug("gasPrice", "gasPrice", gasPrice.String())
	if currencyId.IsNative() {
		tx = types.NewTransaction(nonce, common.HexToAddress(req.To), big.NewInt(0).SetBytes(req.Amount.Value.Data), uint64(21000), gasPrice, nil)
		slog.Debug("transfer native", "tx", tx)
	} else if currencyId.IsErc20() {
		transferFnSignature := []byte("transfer(address,uint256)")
		hash := sha3.NewLegacyKeccak256()
		hash.Write(transferFnSignature)
		methodID := hash.Sum(nil)[:4]

		tokenAddress := common.HexToAddress(currencyId.Address)
		toAddress := common.HexToAddress(req.To)

		paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
		paddedAmount := common.LeftPadBytes(big.NewInt(0).SetBytes(req.Amount.Value.Data).Bytes(), 32)
		var data []byte
		data = append(data, methodID...)
		data = append(data, paddedAddress...)
		data = append(data, paddedAmount...)

		slog.Debug("estimating gas", "from", req.From, "to", req.To, "tokenAddress", tokenAddress)
		gasLimit, err := srv.c.EstimateGas(ctx, ethereum.CallMsg{
			From: common.HexToAddress(req.From),
			To:   &tokenAddress,
			Data: data,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to estimate gas: %v", err)
		}
		gasEstimate = gasLimit

		tx = types.NewTransaction(nonce, tokenAddress, big.NewInt(0), gasLimit, gasPrice, data)
		slog.Debug("transfer erc20", "tx", tx)
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "invalid currency id: %s", req.Amount.CurrencyId)
	}

	slog.Debug("chainId", "chainId", srv.chainId, "tx", tx)
	txId := types.NewEIP155Signer(srv.chainId).Hash(tx)

	slog.Debug("calculating txId", "txId", txId)
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal tx: %v", err)
	}

	intent := &services.TransactionIntent{
		Id:            txId.Bytes(),
		PayloadToSign: txId.Bytes(),
		SignatureType: "secp256k1",
		RawData:       rawTx,
		EstimatedFee:  &proto.Uint256{Data: big.NewInt(int64(gasEstimate)).Bytes()},
	}

	return intent, nil

	//tx := types.NewTransaction(nonce, common.HexToAddress(req.To), big.NewInt(12400000), 10000000, big.NewInt(0), nil)
	//tx2, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1)), nil)
	//tx2.
	//(types.NewEIP155Signer(big.NewInt(1)))
	//srv.c.EstimateGas(ctx, tx)

	//return nil, status.Errorf(codes.Unimplemented, "method CreateTransfer not implemented")
}
func (srv *EthServer) CombineTransaction(ctx context.Context, req *services.TransactionCombineRequest) (*services.SignedTransaction, error) {
	return &services.SignedTransaction{
		Intent:     req.Intent,
		Signatures: req.Signatures,
	}, nil
}
func (srv *EthServer) SignTransaction(ctx context.Context, req *services.TransactionSignRequest) (*services.SignedTransaction, error) {
	signature, err := eth.SignData(req.Intent.PayloadToSign, req.PrivateKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sign tx: %v", err)
	}

	return &services.SignedTransaction{
		Intent:     req.Intent,
		Signatures: [][]byte{signature},
	}, nil
}
func (srv *EthServer) Send(ctx context.Context, req *services.TransactionSendRequest) (*services.TransactionSendResponse, error) {
	slog.Debug("sendTx", "req", req)
	tx := &types.Transaction{}

	err := tx.UnmarshalBinary(req.Intent.RawData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal raw tx: %v", err)
	}

	tx, err = tx.WithSignature(types.NewEIP155Signer(srv.chainId), req.Signatures[0])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sign tx: %v", err)
	}

	slog.Debug("sendTx", "tx", tx)
	err = srv.c.Client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send tx: %v", err)
	}

	return &services.TransactionSendResponse{
		Id: tx.Hash().Bytes(),
	}, nil

}
