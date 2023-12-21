import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox-viem/network-helpers";
import { expect } from "chai";
import hre from "hardhat";
import { getAddress, parseGwei } from "viem";
import { UbtAgent } from "../scripts/agent";
import { JsonRpcServer } from "hardhat/internal/hardhat-network/jsonrpc/server";
import { UbtClient } from "../scripts/client";
import { arrayFromAsync } from "../scripts/utils";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import debug from "debug"

const log = debug("ubt:test:ChainService")

let agent: UbtAgent;
let jsonRpcSrv: JsonRpcServer
let client: UbtClient;

describe("ChainService", () => {
  before(async () => {
    jsonRpcSrv = new JsonRpcServer({hostname: "localhost", port: 8545, provider: hre.network.provider})
    jsonRpcSrv.listen()
    agent = new UbtAgent();
    const l = await agent.start("Listening at");
    log(`agent started ${agent.proc?.pid}`);
    client = new UbtClient("localhost:50051")
  });
  
  after(async () => {
    const exitCode = await agent.stop();
    log(`ubt agent stopped (${agent.proc?.killed}) exitCode=${exitCode}`);
    await jsonRpcSrv.close()
    log(`JSON RPC server stopped`)
  });

  

  it("Should return list of supported chains", async () => {
    const chains = await arrayFromAsync(client.chainService().listChains({}).responses);
   
    log(chains)
    expect(chains.length).eq(1);
    expect(chains[0].id?.type).eq("ETH");
    expect(chains[0].id?.network).eq("SEPOLIA");
  });
  it("Should return list of chains filtered by type", async () => {
    const chains = await arrayFromAsync(client.chainService().listChains({type: "ETH"}).responses);
   
    expect(chains.length).eq(1);
    expect(chains[0].id?.type).eq("ETH");
    expect(chains[0].id?.network).eq("SEPOLIA");
  });

  it("Should return empty list when filtered by non-existing chain type", async () => {
    const chains = await arrayFromAsync(client.chainService().listChains({type: "SOMETHIN"}).responses);
   
    expect(chains).length(0);
  });

  it("Should return chain by id", async () => {
    const chain = await client.chainService().getChain({type: "ETH", network: "SEPOLIA"});
   
    expect(chain.response.id?.type).eq("ETH");
    expect(chain.response.id?.network).eq("SEPOLIA");
  });

  it("Should throw 'not found' for wrong chain id", async () => {
    try {
      await client.chainService().getChain({type: "ETH", network: "UNKNOWN"});
      expect(true).eq(false);
    } catch (e) {
      log(e)
      expect((e as RpcError).message).eq("chain not supported");
    }

    try {
      await client.chainService().getChain({type: "SOME", network: "UNKNOWN"});
      expect(true).eq(false);
    } catch (e) {
      log(e)
      expect((e as RpcError).message).eq("chain not supported");
    }
  });
});
