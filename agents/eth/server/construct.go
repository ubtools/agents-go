package server

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ubtr/ubt-go/agents/eth/rpc"
	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/blockchain/eth"
	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (srv *EthServer) CreateTransfer(ctx context.Context, req *services.CreateTransferRequest) (*services.TransactionIntent, error) {

	nonce, err := rpc.AdoptClient(srv.C).PendingNonceAt(ctx, common.HexToAddress(req.From))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get nonce: %v", err)
	}

	gasPrice, err := rpc.AdoptClient(srv.C).SuggestGasPrice(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get gas price: %v", err)
	}

	currencyId, err := blockchain.UChainCurrencyIdromString(req.Amount.CurrencyId)
	if err != nil {
		return nil, err
	}

	toAddress, err := srv.AddressFromString(req.To)
	if err != nil {
		return nil, err
	}
	fromAddress, err := srv.AddressFromString(req.From)
	if err != nil {
		return nil, err
	}

	var gasEstimate uint64 = 21000 // native transfer

	var tx *types.Transaction

	if currencyId.IsNative() {
		tx = types.NewTransaction(nonce, toAddress, big.NewInt(0).SetBytes(req.Amount.Value.Data), uint64(21000), gasPrice, nil)
		srv.Log.Debug("transfer native", "tx", tx)
	} else if currencyId.IsErc20() {
		transferFnSignature := []byte("transfer(address,uint256)")
		hash := sha3.NewLegacyKeccak256()
		hash.Write(transferFnSignature)
		methodID := hash.Sum(nil)[:4]

		tokenAddress, err := srv.AddressFromString(currencyId.Address)
		if err != nil {
			return nil, err
		}

		paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
		paddedAmount := common.LeftPadBytes(big.NewInt(0).SetBytes(req.Amount.Value.Data).Bytes(), 32)
		var data []byte
		data = append(data, methodID...)
		data = append(data, paddedAddress...)
		data = append(data, paddedAmount...)

		gasLimit, err := rpc.AdoptClient(srv.C).EstimateGas(ctx, ethereum.CallMsg{
			From: fromAddress,
			To:   &tokenAddress,
			Data: data,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to estimate gas: %v", err)
		}
		gasEstimate = gasLimit

		tx = types.NewTransaction(nonce, tokenAddress, big.NewInt(0), gasLimit, gasPrice, data)
		srv.Log.Debug("transfer erc20", "tx", tx)
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "invalid currency id: %s", req.Amount.CurrencyId)
	}

	srv.Log.Debug("Estimating", "gasPrice", gasPrice, "gasEstimate", gasEstimate)

	txId := types.NewEIP155Signer(srv.ChainId).Hash(tx)

	srv.Log.Debug("calculating txId", "txId", txId)
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal tx: %v", err)
	}

	gasCost := big.NewInt(0).Mul(big.NewInt(0).SetUint64(gasEstimate), gasPrice)

	intent := &services.TransactionIntent{
		Id:            txId.Bytes(),
		PayloadToSign: txId.Bytes(),
		SignatureType: eth.Instance.SignatureType,
		RawData:       rawTx,
		EstimatedFee:  &proto.Uint256{Data: gasCost.Bytes()},
	}

	return intent, nil

	//tx := types.NewTransaction(nonce, common.HexToAddress(req.To), big.NewInt(12400000), 10000000, big.NewInt(0), nil)
	//tx2, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1)), nil)
	//tx2.
	//(types.NewEIP155Signer(big.NewInt(1)))
	//srv.C.EstimateGas(ctx, tx)

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
	srv.Log.Debug("sendTx", "req", req)
	tx := &types.Transaction{}

	err := tx.UnmarshalBinary(req.Intent.RawData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal raw tx: %v", err)
	}

	tx, err = tx.WithSignature(types.NewEIP155Signer(srv.ChainId), req.Signatures[0])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sign tx: %v", err)
	}

	srv.Log.Debug("sendTx", "tx", tx)
	err = rpc.AdoptClient(srv.C).SendTransaction(ctx, tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send tx: %v", err)
	}

	return &services.TransactionSendResponse{
		Id: tx.Hash().Bytes(),
	}, nil

}
