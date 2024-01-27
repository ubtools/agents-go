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
import { Block, CurrencyId, FinalityStatus, ListBlocksRequest_IncludeFlags, hexutils, uint256, uint256utils } from "@ubtr/sdk";
import { ETH_CHAIN_ID } from "../scripts/testfixtures";
const log = debug("ubt:test:BlockService")

let agent: UbtAgent;
let jsonRpcSrv: JsonRpcServer
let client: UbtClient;

describe("BlockService", () => {

  async function deployTestERC20Token() {
    // Contracts are deployed using the first signer/account by default
    const [owner, otherAccount] = await hre.viem.getWalletClients();

    const ownerBalance = 10000n;

    const token = await hre.viem.deployContract("TestERC20", ["UBTToken", "UBTR"]);

    const publicClient = await hre.viem.getPublicClient();

    return {
      token,
      ownerBalance,
      owner,
      otherAccount,
      publicClient,
    };
  }

  before(async () => {
    jsonRpcSrv = new JsonRpcServer({hostname: "localhost", port: 8545, provider: hre.network.provider})
    await jsonRpcSrv.listen()
    agent = new UbtAgent();
    const l = await agent.start("API listening at");
    log(`agent started ${agent.proc?.pid}`);
    client = new UbtClient("localhost:50051")
  });
  
  after(async () => {
    const exitCode = await agent.stop();
    log(`ubt agent stopped (${agent.proc?.killed}) exitCode=${exitCode}`);
    await jsonRpcSrv.close()
    log(`JSON RPC server stopped`)
  });

  it("Should return list of blocks", async () => {
    const f = await loadFixture(deployTestERC20Token)
    const blocks = await arrayFromAsync(client.blockService().listBlocks({chainId: ETH_CHAIN_ID,
      startNumber: 0n, finalityStatus: FinalityStatus.UNSPECIFIED, includes: ListBlocksRequest_IncludeFlags.FULL as number, count: 2n}).responses);
   
    log(blocks)
    log("Block timestamp", new Date(new Number(blocks[0].header?.timestamp?.seconds).valueOf() * 1000))
    expect(blocks.length).eq(2);
    
    expect(blocks[0]).to.deep.include({})
    expect(blocks[0].header?.number).eq(0n);
    expect(blocks[0].transactions.length).eq(0); // genesis block has no transactions
    expect(blocks[1].header?.number).eq(1n);
    expect(blocks[1].transactions.length).eq(1); // contract deployment

    const tx = blocks[1].transactions[0]
    expect(tx.from).eq(getAddress(f.owner.account.address))
    expect(tx.to).eq("")
    expect(tx.transfers.length).eq(1)

    // validate mint transfer
    const mintTr = tx.transfers[0]
    expect(mintTr.from).eq("0x0000000000000000000000000000000000000000")
    expect(mintTr.to).eq(getAddress(f.owner.account.address))
    expect(mintTr.amount?.currencyId).eq(`${getAddress(f.token.address)}`)
    expect(uint256utils.toBigInt(mintTr.amount?.value!)).eq(f.ownerBalance)
    log(JSON.stringify(blocks[1].transactions[0]))
  });

  it("Should fail on list block if start more than head", async () => {
    const f = await loadFixture(deployTestERC20Token)
    await expect(arrayFromAsync(client.blockService().listBlocks({chainId: ETH_CHAIN_ID,
      startNumber: 3n, finalityStatus: FinalityStatus.UNSPECIFIED, includes: ListBlocksRequest_IncludeFlags.FULL as number, count: 2n}).responses))
      .rejectedWith(RpcError)
  });

  it("Should return block by id", async () => {
    const f = await loadFixture(deployTestERC20Token)
    const blocks = await arrayFromAsync(client.blockService().listBlocks({chainId: ETH_CHAIN_ID,
      startNumber: 0n, finalityStatus: FinalityStatus.UNSPECIFIED, includes: ListBlocksRequest_IncludeFlags.FULL as number, count: 2n}).responses);
   
    log(blocks)
    log("Block timestamp", new Date(new Number(blocks[0].header?.timestamp?.seconds).valueOf() * 1000))
    expect(blocks.length).eq(2);

    expect(blocks[0].header?.number).eq(0n);
    expect(blocks[0].transactions.length).eq(0); // genesis block has no transactions
    expect(blocks[1].header?.number).eq(1n);
    expect(blocks[1].transactions.length).eq(1); // contract deployment
    const tx = blocks[1].transactions[0]
    expect(tx.from).eq(getAddress(f.owner.account.address))
    expect(tx.to).eq("")
    expect(tx.transfers.length).to.eq(1)
    // validate mint transfer
    const mintTr = tx.transfers[0]
    expect(mintTr.from).eq("0x0000000000000000000000000000000000000000")
    expect(mintTr.to).eq(getAddress(f.owner.account.address))
    expect(mintTr.amount?.currencyId).eq(`${getAddress(f.token.address)}`)
    expect(uint256utils.toBigInt(mintTr.amount?.value!)).eq(f.ownerBalance)
    log(JSON.stringify(blocks[1].transactions[0]))
  });

});
