package trx

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shengdoushi/base58"
	"github.com/ubtr/ubt-go/agent"
	"github.com/ubtr/ubt-go/blockchain"
	"github.com/ubtr/ubt-go/blockchain/trx"
	"github.com/ubtr/ubt-go/commons/cache"
	"github.com/ubtr/ubt-go/commons/conv/uint256conv"
	"github.com/ubtr/ubt-go/commons/rpcerrors"
	"github.com/ubtr/ubt-go/eth/contracts/erc20"
	"github.com/ubtr/ubt-go/eth/server"
	"github.com/ubtr/ubt/go/api/proto/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ERC20_FEE_LIMIT = 20000000

func init() {
	agent.AgentFactories[trx.CODE_STR] = func(ctx context.Context, config *agent.ChainConfig) agent.UbtAgent {
		return InitServer(ctx, config)
	}
}

var TrxExtensions = server.Extensions{
	AddressFromString: func(address string) (common.Address, error) {
		addrB58, err := base58.Decode(address, base58.BitcoinAlphabet)
		if err != nil {
			return common.Address{}, err
		}
		addrB58 = addrB58[:len(addrB58)-4]
		return common.BytesToAddress(addrB58[len(addrB58)-20:]), nil
	},
	AddressToString: func(address common.Address) string {
		addressTron := make([]byte, 0)
		addressTron = append(addressTron, trx.TronBytePrefix)
		addressTron = append(addressTron, address.Bytes()...)
		return trx.Address(addressTron).String()
	},
}

func InitServer(ctx context.Context, config *agent.ChainConfig) *TrxAgent {
	agent := &TrxAgent{
		EthServer:      *server.InitServer(ctx, config),
		feePricesCache: cache.NewSimpleExpirationCache[feePrices](10 * time.Second),
	}
	if config.HttpUrls == nil || len(config.HttpUrls) == 0 || config.HttpUrls[0].Url == "" {
		agent.Log.Warn("no http url provided - trx requires http api to create/sign txs")
	} else {
		agent.client = NewTrxApiClient(config.HttpUrls[0].Url, agent.Log)
	}

	agent.EthServer.Extensions = TrxExtensions

	return agent
}

type feePrices struct {
	energyPrice    *big.Int
	bandwidthPrice *big.Int
}

type TrxAgent struct {
	server.EthServer
	client         *TrxApiClient
	feePricesCache *cache.SimpleExpirationCache[feePrices]
}

// get energy and bandwidth prices in suns
func (srv *TrxAgent) GetFeePrices(ctx context.Context) (feePrices, error) {
	prices, ok := srv.feePricesCache.Get()
	if ok {
		return prices, nil
	} else {
		prices, err := srv.client.GetChainParameters(ctx)
		if err != nil {
			return feePrices{}, err
		}
		convertedPrices := feePrices{
			energyPrice:    prices.EnergyPrice,
			bandwidthPrice: prices.BandwidthPrice,
		}
		srv.feePricesCache.Set(convertedPrices)
		return convertedPrices, nil
	}
}

func (srv *TrxAgent) estimateBandwidth(rawDataHexSize uint64, correction uint64) uint64 {
	const max_result_size = 64
	const signature_size = 67
	const protobuf_extra_size = 3
	return rawDataHexSize/2 + max_result_size + signature_size + protobuf_extra_size + correction
}

func (srv *TrxAgent) estimateFee(ctx context.Context, bandwidth uint64, energy uint64) (*big.Int, error) {
	feePrices, err := srv.GetFeePrices(ctx)
	if err != nil {

		return nil, status.Errorf(codes.Unavailable, "failed to get fee prices: %s", err)
	}
	feeEstimate := big.NewInt(0).Mul(big.NewInt(int64(bandwidth)), feePrices.bandwidthPrice)

	return feeEstimate.Add(feeEstimate, big.NewInt(0).Mul(big.NewInt(int64(energy)), feePrices.energyPrice)), nil
}

func (srv *TrxAgent) CreateTransfer(ctx context.Context, req *services.CreateTransferRequest) (*services.TransactionIntent, error) {
	srv.Log.Debug("CreateTransfer", "req", req, "amount", big.NewInt(0).SetBytes(req.Amount.Value.Data))
	if srv.client == nil {
		return nil, errors.ErrUnsupported
	}

	curId, err := blockchain.UChainCurrencyIdromString(req.Amount.CurrencyId)
	if err != nil {
		return nil, err
	}

	if curId.IsNative() {
		res, err := srv.client.CreateTransaction(ctx, CreateTransactionRequest{
			OwnerAddress: req.From,
			ToAddress:    req.To,
			Amount:       big.NewInt(0).SetBytes(req.Amount.Value.Data).Uint64(),
			Visible:      true,
		})

		if err != nil {
			return nil, err
		}

		if res.Error != "" {
			return nil, errors.New(res.Error)
		}

		if err != nil {
			return nil, err
		}

		bandwidthEstimate := srv.estimateBandwidth(uint64(len(res.RawDataHex)), 0)

		srv.Log.Debug("TransferIntent", "bandwidth", bandwidthEstimate)
		feeEstimate, err := srv.estimateFee(ctx, bandwidthEstimate, 0)
		if err != nil {
			return nil, err
		}

		return &services.TransactionIntent{
			Id:            common.Hex2Bytes(res.TxId),
			SignatureType: trx.Instance.SignatureType,
			PayloadToSign: common.Hex2Bytes(res.TxId),
			RawData:       res.RawData,
			EstimatedFee:  uint256conv.FromBigInt(feeEstimate),
		}, nil

	} else if curId.IsErc20() {

		erc20Abi, err := erc20.Erc20MetaData.GetAbi()
		if err != nil {
			return nil, err
		}

		addr, err := srv.Extensions.AddressFromString(req.To)
		if err != nil {
			return nil, err
		}

		data, err := erc20Abi.Pack("transfer", addr, big.NewInt(0).SetBytes(req.Amount.Value.Data))
		if err != nil {
			return nil, err
		}

		estimateRes, err := srv.client.TriggerConstantContract(ctx, TriggerContractRequest{
			OwnerAddress:    req.From,
			ContractAddress: curId.Address,
			FeeLimit:        ERC20_FEE_LIMIT,
			CallValue:       0,
			Data:            common.Bytes2Hex(data),
			Visible:         true,
		})

		if err != nil {
			return nil, err
		}

		if !estimateRes.Result.Result {
			return nil, fmt.Errorf("failed to create tx: %s %s", estimateRes.Result.Code, estimateRes.Result.Message)
		}

		bandwidthEstimate := srv.estimateBandwidth(uint64(len(estimateRes.Transaction.RawDataHex)), 6)
		feeEstimate, err := srv.estimateFee(ctx, bandwidthEstimate, estimateRes.EnergyUsed)
		if err != nil {
			return nil, err
		}

		srv.Log.Debug("TransferIntent", "bandwidth", bandwidthEstimate, "energy", estimateRes.EnergyUsed)

		triggerRes, err := srv.client.TriggerSmartContract(ctx, TriggerContractRequest{
			OwnerAddress:    req.From,
			ContractAddress: curId.Address,
			FeeLimit:        ERC20_FEE_LIMIT,
			CallValue:       0,
			Data:            common.Bytes2Hex(data),
			Visible:         true,
		})

		if err != nil {
			return nil, err
		}

		if !triggerRes.Result.Result {
			return nil, fmt.Errorf("failed to create tx: %s %s", triggerRes.Result.Code, triggerRes.Result.Message)
		}

		return &services.TransactionIntent{
			Id:            common.Hex2Bytes(triggerRes.Transaction.TxId),
			SignatureType: trx.Instance.SignatureType,
			PayloadToSign: common.Hex2Bytes(triggerRes.Transaction.TxId),
			RawData:       triggerRes.Transaction.RawData,
			EstimatedFee:  uint256conv.FromBigInt(feeEstimate),
		}, nil

	} else {
		return nil, rpcerrors.ErrInvalidCurrency
	}
}

func (srv *TrxAgent) Send(ctx context.Context, req *services.TransactionSendRequest) (*services.TransactionSendResponse, error) {
	srv.Log.Debug("Send", "req", req)
	if srv.client == nil {
		return nil, errors.ErrUnsupported
	}

	var signatures []string
	for _, signature := range req.Signatures {
		signatures = append(signatures, common.Bytes2Hex(signature))
	}
	res, err := srv.client.BroadcastTransaction(ctx, BroadcastTransactionRequest{
		Visible:   true,
		RawData:   req.Intent.RawData,
		Signature: signatures,
	})

	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Failed to broadcast tx %s", err.Error())
	}

	if !res.Result {
		return nil, status.Errorf(codes.InvalidArgument, "failed to broadcast tx: %s %s", res.Code, res.Message)
	}

	return &services.TransactionSendResponse{
		Id: req.Intent.Id,
	}, nil
}
