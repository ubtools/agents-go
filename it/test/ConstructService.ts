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
const log = debug("ubt:test:ConstructService")

let agent: UbtAgent;
let jsonRpcSrv: JsonRpcServer
let client: UbtClient;

describe("ConstructService", () => {

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

  it("Should create, sign and send transfer for ERC20", async () => {
    const f = await loadFixture(deployTestERC20Token)

    const intent = await client.constructService().createTransfer({
      chainId: ETH_CHAIN_ID,
      from: getAddress(f.owner.account.address),
      to: getAddress(f.otherAccount.account.address),
      amount: {currencyId: `${getAddress(f.token.address)}`, value: uint256utils.fromBigInt(1000n)}
    })

    expect(intent).not.null
  });


});
