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

	"github.com/ubtools/ubt/go/api/proto"
	"github.com/ubtools/ubt/go/api/proto/services"
)

type EthServer struct {
	services.UnimplementedUbtChainServiceServer
	services.UnimplementedUbtBlockServiceServer
	services.UnimplementedUbtConstructServiceServer
	services.UnimplementedUbtCurrencyServiceServer
	C       *client.RateLimitedClient
	Config  config.ChainConfig
	Chain   blockchain.Blockchain
	ChainId *big.Int
}

func (srv *EthServer) String() string {
	return fmt.Sprintf("EthServer{%s:%s}", srv.Config.ChainType, srv.Config.ChainNetwork)
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
	var srv = EthServer{C: client, Config: *config, ChainId: chainId}
	return &srv
}

func (srv *EthServer) GetNetwork(ctx context.Context, netId *proto.ChainId) (*proto.Chain, error) {
	if netId.Type != srv.Chain.Type {
		return nil, status.Errorf(codes.Unimplemented, "method GetNetwork not implemented")
	}
	id := uint32(srv.Chain.TypeNum)
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
	chain := srv.Config
	err := s.Send(&proto.Chain{
		Id:              &proto.ChainId{Type: chain.ChainType, Network: chain.ChainNetwork},
		Bip44Id:         nil, //srv.Chain.TypeNum,
		Testnet:         chain.Testnet,
		FinalizedHeight: 20,
		MsPerBlock:      3000,
		SupportedServices: []proto.Chain_ChainSupportedServices{
			proto.Chain_BLOCK, proto.Chain_CONSTRUCT, proto.Chain_CURRENCIES},
	})
	if err != nil {
		return err
	}
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
	_, err := srv.C.HeaderByHash(ctx, common.Hash(req.Id))
	if err != nil {
		return nil, err
	}
	converter := &BlockConverter{Config: &srv.Config, Client: srv.C, Ctx: ctx}
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

	err := srv.C.BatchCallContext(res.Context(), blockReqs)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("Got %d blocks\n", len(blockReqs)))

	for _, blockReq := range blockReqs {
		if blockReq.Error != nil {
			return blockReq.Error
		}
		blockRes := blockReq.Result.(*HeaderWithBody)
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

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.Version, "web3_clientVersion")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.Listening, "net_listening")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.SyncInfo, "eth_syncing")

	srv.C.Client.Client().CallContext(ctx, &(nodeInfo.ChainId), "eth_chainId")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.PeerCount, "net_peerCount")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.GenesisBlockHash, "net_version")

	var nodeInfoJson, _ = json.Marshal(nodeInfo)

	log.Printf("Node info: %s", nodeInfoJson)
}

func (srv *EthServer) Test2(ctx context.Context) {
	var nodeInfo = TronNodeInfo{}

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.Version, "web3_clientVersion")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.Listening, "net_listening")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.SyncInfo, "eth_syncing")

	srv.C.Client.Client().CallContext(ctx, &(nodeInfo.ChainId), "eth_chainId")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.PeerCount, "net_peerCount")

	srv.C.Client.Client().CallContext(ctx, &nodeInfo.GenesisBlockHash, "net_version")

	var nodeInfoJson, _ = json.Marshal(nodeInfo)

	log.Printf("Node info: %s", nodeInfoJson)
}
