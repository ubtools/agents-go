package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"ubt/agents/commons"
	"ubt/agents/eth/client"
	"ubt/agents/eth/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ubtools/ubt/go/blockchain"
	"github.com/ubtools/ubt/go/blockchain/eth"

	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
)

type EthServer struct {
	services.UnimplementedUbtChainServiceServer
	services.UnimplementedUbtBlockServiceServer
	services.UnimplementedUbtConstructServiceServer
	services.UnimplementedUbtCurrencyServiceServer
	c       *client.RateLimitedClient
	config  *config.ChainConfig
	chain   blockchain.Blockchain
	chainId *big.Int
}

func InitServer(ctx context.Context, config *config.ChainConfig) *EthServer {
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
	var srv = EthServer{c: client, config: config, chainId: chainId}
	return &srv
}

func (srv *EthServer) GetNetwork(ctx context.Context, netId *proto.ChainId) (*proto.Chain, error) {
	if netId.Type != srv.chain.Type {
		return nil, status.Errorf(codes.Unimplemented, "method GetNetwork not implemented")
	}
	id := uint32(srv.chain.TypeNum)
	return &proto.Chain{
		Id:              netId,
		Bip44Id:         &id,
		Testnet:         false,
		FinalizedHeight: 20,
		MsPerBlock:      3000,
		SupportedServices: []proto.Chain_ChainSupportedServices{
			proto.Chain_BLOCK, proto.Chain_CONSTRUCT, proto.Chain_CURRENCIES},
	}, nil
}

func (srv *EthServer) ListChains(req *services.ListChainsRequest, s services.UbtChainService_ListChainsServer) error {
	id := uint32(eth.CODE_NUM)
	net := &proto.Chain{
		Id:              &proto.ChainId{Type: eth.CODE_STR, Network: commons.MAINNET},
		Bip44Id:         &id,
		Testnet:         false,
		FinalizedHeight: 20,
		MsPerBlock:      3000,
		SupportedServices: []proto.Chain_ChainSupportedServices{
			proto.Chain_BLOCK, proto.Chain_CONSTRUCT, proto.Chain_CURRENCIES},
	}
	s.Send(net)
	return nil
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

func (srv *EthServer) GetBlock(ctx context.Context, req *services.BlockRequest) (*proto.Block, error) {
	_, err := srv.c.HeaderByHash(ctx, common.Hash(req.Id))
	if err != nil {
		return nil, err
	}
	converter := &BlockConverter{Config: srv.config, Client: srv.c, Ctx: ctx}
	b, err := converter.EthBlockToProto(nil)
	if err != nil {
		return nil, err
	}
	b.Transactions = []*proto.Transaction{}
	return b, nil
}

func (srv *EthServer) ListBlocks(req *services.ListBlocksRequest, res services.UbtBlockService_ListBlocksServer) error {
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
			Result: &HeaderWithBody{},
		})
	}

	err := srv.c.BatchCallContext(res.Context(), blockReqs)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("Got %d blocks\n", len(blockReqs)))

	for _, blockReq := range blockReqs {
		slog.Debug("Block:", "result", blockReq.Result, "error", blockReq.Error)
		if blockReq.Error != nil {
			return blockReq.Error
		}
		blockRes := blockReq.Result.(*HeaderWithBody)
		converter := &BlockConverter{Config: srv.config, Client: srv.c, Ctx: res.Context()}
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

type NodeSyncInfo struct {
	StartingBlock commons.UInt64HexString
	CurrentBlock  commons.UInt64HexString
	HighestBlock  commons.UInt64HexString
}

type TronNodeInfo struct {
	Listening bool
	ChainId   string
	Version   string

	SyncInfo NodeSyncInfo

	PeerCount        commons.UInt64HexString
	GenesisBlockHash string
}

func (srv *EthServer) Test(ctx context.Context) {
	var nodeInfo = TronNodeInfo{}

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.Version, "web3_clientVersion")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.Listening, "net_listening")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.SyncInfo, "eth_syncing")

	srv.c.Client.Client().CallContext(ctx, &(nodeInfo.ChainId), "eth_chainId")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.PeerCount, "net_peerCount")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.GenesisBlockHash, "net_version")

	var nodeInfoJson, _ = json.Marshal(nodeInfo)

	log.Printf("Node info: %s", nodeInfoJson)
}

func (srv *EthServer) Test2(ctx context.Context) {
	var nodeInfo = TronNodeInfo{}

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.Version, "web3_clientVersion")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.Listening, "net_listening")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.SyncInfo, "eth_syncing")

	srv.c.Client.Client().CallContext(ctx, &(nodeInfo.ChainId), "eth_chainId")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.PeerCount, "net_peerCount")

	srv.c.Client.Client().CallContext(ctx, &nodeInfo.GenesisBlockHash, "net_version")

	var nodeInfoJson, _ = json.Marshal(nodeInfo)

	log.Printf("Node info: %s", nodeInfoJson)
}
