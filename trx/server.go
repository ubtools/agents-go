package trx

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"ubt/agents/eth/client"
	"ubt/agents/eth/config"
	"ubt/agents/eth/server"
	trxrpc "ubt/agents/trx/rpc"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ubtools/ubt/go/api/proto/services"
)

type TrxServer struct {
	server.EthServer
	//services.UnimplementedUbtChainServiceServer
	//services.UnimplementedUbtBlockServiceServer
	//services.UnimplementedUbtConstructServiceServer
	//services.UnimplementedUbtCurrencyServiceServer
	//c       *client.RateLimitedClient
	//config  *config.ChainConfig
	//chain   blockchain.Blockchain
	//chainId *big.Int
}

func (srv *TrxServer) String() string {
	return fmt.Sprintf("TrxServer{%s:%s}", srv.Config.ChainType, srv.Config.ChainNetwork)
}

func InitServer(ctx context.Context, config *config.ChainConfig) *TrxServer {
	slog.Info(fmt.Sprintf("Connecting to chain '%s:%s'", config.ChainType, config.ChainNetwork), "rpcUrl", config.RpcUrl, "limitRps", config.LimitRPS)
	client, err := client.DialContext(context.Background(), config.RpcUrl, config.LimitRPS)
	if err != nil {
		panic(err)
	}
	chainId, err := client.ChainID(ctx)
	if err != nil {
		panic(err)
	}
	slog.Info("Connected", "chainId", chainId)
	var srv = TrxServer{EthServer: server.EthServer{C: client, Config: *config, ChainId: chainId}}
	return &srv
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	if number.IsInt64() {
		return rpc.BlockNumber(number.Int64()).String()
	}
	// It's negative and large, which is invalid.
	return fmt.Sprintf("<invalid %d>", number)
}

func (srv *TrxServer) ListBlocks(req *services.ListBlocksRequest, res services.UbtBlockService_ListBlocksServer) error {
	slog.Debug(fmt.Sprintf("ListBlocks from %d, count = %v\n", req.StartNumber, req.Count))
	blockReqs := []rpc.BatchElem{}
	var endNumber uint64 = 0
	if (req.Count == nil) || (*req.Count == 0) {
		endNumber = req.StartNumber + 10
	} else {
		endNumber = req.StartNumber + *req.Count
	}
	for i := req.StartNumber; i < endNumber; i++ {
		blockReqs = append(blockReqs, rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{toBlockNumArg(big.NewInt(int64(i))), true},
			Result: &trxrpc.HeaderWithBody{},
		})
	}

	err := srv.C.BatchCallContext(res.Context(), blockReqs)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("Got %d blocks\n", len(blockReqs)))

	for _, blockReq := range blockReqs {
		slog.Debug("Block:", "result", blockReq.Result, "error", blockReq.Error)
		if blockReq.Error != nil {
			return blockReq.Error
		}
		blockRes := blockReq.Result.(*trxrpc.HeaderWithBody)
		converter := &BlockConverter{Config: &srv.Config, Client: srv.C, Ctx: res.Context()}
		block, err := converter.EthBlockToProto(blockRes)
		if err != nil {
			slog.Error("Error converting block", "error", err)
			return err
		}
		slog.Debug("TxCount", "count", len(block.Transactions))
		err = res.Send(block)
		if err != nil {
			slog.Error("Error sending block", "error", err)
			return err
		}
	}
	slog.Debug("Done sending blocks")
	return nil
}
