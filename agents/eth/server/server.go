package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/big"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/ubtr/ubt-go/agent"
	ethrpc "github.com/ubtr/ubt-go/agents/eth/rpc"
	ethtypes "github.com/ubtr/ubt-go/agents/eth/types"
	"github.com/ubtr/ubt-go/blockchain/eth"
	"github.com/ubtr/ubt-go/commons"
	"github.com/ubtr/ubt-go/commons/jsonrpc"
	"github.com/ubtr/ubt-go/commons/jsonrpc/client"
	"github.com/ubtr/ubt-go/commons/rpcerrors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ubtr/ubt-go/blockchain"
	ubtcache "github.com/ubtr/ubt-go/commons/cache"

	"github.com/ubtr/ubt/go/api/proto"
	"github.com/ubtr/ubt/go/api/proto/services"
)

func init() {
	agent.AgentFactories[eth.CODE_STR] = func(ctx context.Context, config *agent.ChainConfig) agent.UbtAgent {
		return InitServer(ctx, config)
	}
}

// extension hooks to tune behaviour of different eth-like chains
type Extensions struct {
	AddressFromString   func(address string) (common.Address, error)
	AddressToString     func(address common.Address) string
	BlockFinalityStatus func(block *proto.Block) proto.FinalityStatus
}

type EthServer struct {
	services.UnimplementedUbtChainServiceServer
	services.UnimplementedUbtBlockServiceServer
	services.UnimplementedUbtConstructServiceServer
	services.UnimplementedUbtCurrencyServiceServer
	C             *client.BalancedClient
	Config        agent.ChainConfig
	Chain         blockchain.Blockchain
	ChainId       *big.Int
	CurrencyCache cache.CacheInterface[*proto.Currency]
	Log           *slog.Logger
	Extensions    Extensions
}

func InitServer(ctx context.Context, config *agent.ChainConfig) *EthServer {

	chainIdStr := config.ChainType + ":" + config.ChainNetwork
	logger := slog.With("chain", chainIdStr)

	logger.Info("Connecting")

	var peers []*client.ClientConfig
	for _, url := range config.RpcUrls {
		upstreamLabel := commons.EitherStr(url.Name, url.Url)
		peers = append(peers, &client.ClientConfig{Url: url.Url, LimitRps: url.LimitRps, Labels: []any{"chain", chainIdStr, "upstream", upstreamLabel}})
		logger.Info(fmt.Sprintf("Upstream %s rps: %v", url.Url, url.LimitRps))
	}
	if len(peers) == 0 {
		panic("No peers configured")
	}
	client := client.NewBalancedClient(peers, []any{"chain", chainIdStr}) //client.DialContext(ctx, config.LimitRPS, commons.EitherStr())
	client.Start()

	chainId, err := ethrpc.ChainId().Call(ctx, client)
	if err != nil {
		panic(err)
	}
	blockchain := blockchain.GetBlockchain(config.ChainType)
	if blockchain == nil {
		panic(fmt.Sprintf("Unsupported chain type '%s'", config.ChainType))
	}

	var srv = EthServer{C: client, Config: *config, ChainId: chainId, Chain: *blockchain, CurrencyCache: ubtcache.NewCache[*proto.Currency](), Log: logger}

	srv.Log.Info("Connected")
	return &srv
}

func (srv *EthServer) String() string {
	return fmt.Sprintf("EthServer{%s:%s}", srv.Config.ChainType, srv.Config.ChainNetwork)
}

func (srv *EthServer) AddressFromString(address string) (common.Address, error) {
	if srv.Extensions.AddressFromString != nil {
		return srv.Extensions.AddressFromString(address)
	}
	addrBytes, err := hexutil.Decode(address)
	if err != nil {
		return common.Address{}, rpcerrors.ArgError("address", err)
	}
	if len(addrBytes) != common.AddressLength {
		return common.Address{}, rpcerrors.ArgError("address", fmt.Errorf("invalid address length %d", len(addrBytes)))
	}
	return common.BytesToAddress(addrBytes), nil
}

func (srv *EthServer) AddressToString(address *common.Address) string {
	if address == nil {
		return ""
	}
	if srv.Extensions.AddressToString != nil {
		return srv.Extensions.AddressToString(*address)
	}
	return address.Hex()
}

func (srv *EthServer) GetChain(ctx context.Context, chainId *proto.ChainId) (*proto.Chain, error) {
	srv.Log.Debug("GetChain")
	if chainId.Type != srv.Chain.Type {
		return nil, rpcerrors.ErrInvalidChainId
	}
	bip44Id := uint32(srv.Chain.TypeNum)
	return &proto.Chain{
		Id:              chainId,
		Bip44Id:         &bip44Id,
		Testnet:         false,
		FinalizedHeight: 20, //FIXME: this is wrong assumption
		MsPerBlock:      3000,
		SupportedServices: []proto.Chain_ChainSupportedServices{
			proto.Chain_BLOCK, proto.Chain_CONSTRUCT, proto.Chain_CURRENCIES},
	}, nil
}

func (srv *EthServer) ListChains(req *services.ListChainsRequest, s services.UbtChainService_ListChainsServer) error {
	if req.Type != nil && *req.Type != srv.Chain.Type {
		return nil
	}
	chain := srv.Config
	err := s.Send(&proto.Chain{
		Id:              &proto.ChainId{Type: chain.ChainType, Network: chain.ChainNetwork},
		Bip44Id:         nil, //FIXME: change api type &srv.Chain.TypeNum,
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
	block, err := ethrpc.GetBlockByHash(common.Hash(req.Id), true).Call(ctx, srv.C)
	if err != nil {
		return nil, err
	}
	converter := &BlockConverter{Config: &srv.Config, Client: srv.C, Srv: srv, Ctx: ctx, Log: srv.Log.With("block", req.Id)}
	b, err := converter.EthBlockToProto(block)
	if err != nil {
		return nil, err
	}
	b.Transactions = []*proto.Transaction{}
	return b, nil
}

func (srv *EthServer) ListBlocks(req *services.ListBlocksRequest, res services.UbtBlockService_ListBlocksServer) error {
	srv.Log.Debug(fmt.Sprintf("ListBlocks from %d, count = %v\n", req.StartNumber, req.Count))

	// get top block number
	topBlockNumber, err := ethrpc.GetBlockNumber().Call(res.Context(), srv.C)
	if err != nil {
		return err
	}

	var endNumber uint64 = 0
	if (req.Count == nil) || (*req.Count == 0) {
		endNumber = req.StartNumber + 10
	} else {
		endNumber = req.StartNumber + *req.Count
	}
	endNumber = min(endNumber, topBlockNumber+1)
	srv.Log.Debug("Block range", "startNumber", req.StartNumber, "endNumber", endNumber)
	if req.StartNumber >= endNumber {
		return rpcerrors.ErrBlockOutOfRange
	}

	blockReqs := []*jsonrpc.RpcCall[ethtypes.HeaderWithBody]{}
	var batch jsonrpc.RpcBatch
	for i := req.StartNumber; i < endNumber; i++ {
		c := ethrpc.GetBlockByNumber(big.NewInt(int64(i)), true)
		c.AddToBatch(&batch)
		blockReqs = append(blockReqs, c)
	}

	err = batch.Call(res.Context(), srv.C)
	if err != nil {
		return err
	}

	srv.Log.Debug("Blocks received", "count", len(blockReqs))

	for idx, blockReq := range blockReqs {
		err := blockReq.ProcessRes(res.Context())
		if err != nil {
			return err
		}
		blockRes := blockReq.Response
		converter := &BlockConverter{Config: &srv.Config, Client: srv.C, Ctx: res.Context(), Srv: srv, Log: srv.Log.With("block", blockRes.Header.Hash())}
		block, err := converter.EthBlockToProto(blockRes)
		if err != nil {
			srv.Log.Error("Error converting block", "error", err)
			return err
		}
		if block.Header.FinalityStatus < req.FinalityStatus {
			if idx > 0 {
				break
			} else {
				return rpcerrors.ErrBlockOutOfRange
			}
		}
		srv.Log.Debug("Send processed block", "txCount", len(block.Transactions))
		err = res.Send(block)
		if err != nil {
			srv.Log.Error("Error sending block", "error", err)
			return err
		}
	}
	srv.Log.Debug("Done sending blocks")
	return nil
}

type NodeSyncInfo struct {
	StartingBlock commons.UInt64HexString
	CurrentBlock  commons.UInt64HexString
	HighestBlock  commons.UInt64HexString
}

type NodeInfo struct {
	Listening bool
	ChainId   string
	Version   string

	SyncInfo NodeSyncInfo

	PeerCount        commons.UInt64HexString
	GenesisBlockHash string
}

func (srv *EthServer) Info(ctx context.Context) {
	var nodeInfo = NodeInfo{}

	jsonrpc.AnyCall("web3_clientVersion", &nodeInfo.Version).Call(ctx, srv.C)
	jsonrpc.AnyCall("net_listening", &nodeInfo.Listening).Call(ctx, srv.C)
	jsonrpc.AnyCall("eth_syncing", &nodeInfo.SyncInfo).Call(ctx, srv.C)
	jsonrpc.AnyCall("eth_chainId", &nodeInfo.ChainId).Call(ctx, srv.C)
	jsonrpc.AnyCall("net_peerCount", &nodeInfo.PeerCount).Call(ctx, srv.C)
	jsonrpc.AnyCall("net_version", &nodeInfo.GenesisBlockHash).Call(ctx, srv.C)

	var nodeInfoJson, _ = json.Marshal(nodeInfo)

	log.Printf("Node info: %s", nodeInfoJson)
}
