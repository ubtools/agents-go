import { exec, spawn, ChildProcess } from "child_process";
import { promisify } from "util";
import hre from "hardhat";

import {JsonRpcServer} from "hardhat/internal/hardhat-network/jsonrpc/server"
import { UbtClient } from "./client";

import debug from "debug"

const log = debug("ubt:test:agent")

export class UbtAgent {
  private awaiting: boolean = false;
  proc?: ChildProcess;
  constructor() {
    log("agent created");
  }

  async start(stringToAwait?: string) {
    return new Promise(async (resolve, reject) => {
      try {
        await promisify(exec)("go build -C ../cmd/agent-eth");
      } catch (e) {
        reject(e);
      }
      
      log("Running agents");
      const s = spawn("../cmd/agent-eth/agent-eth", ["-c", "./ubt-config.yaml", "--log", "DEBUG"], {
        stdio: ["ignore", "pipe", "pipe"],
      });
      if (stringToAwait) {
        this.awaiting = true;
      }

      s.stdout.on("data", (data) => {
        log(`stdout: ${data}`);
        if (this.awaiting && data.toString().includes(stringToAwait)) {
          resolve(s);
        }
      });

      s.stderr.on("data", (data) => {
        log(`stderr: ${data}`);
        if (this.awaiting && data.toString().includes(stringToAwait)) {
          resolve(s);
        }
      });

      s.on("close", (code) => {
        log(`child process exited with code ${code}`);
      });
      this.proc = s;
      if (!stringToAwait) {
        resolve(s);
      }
    });
  }

  async stop(): Promise<number> {
    return new Promise((resolve, reject) => {
      if (!this.proc) {
        throw new Error("not started");
      }
      this.proc.on("exit", (code) => {
        resolve(code ?? -1000);
      });
      this.proc.kill("SIGTERM");
    });
  }
}
