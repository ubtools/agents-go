import ganache, { Server } from "ganache";

export class TestNode {
  private node: Server;
  constructor() {
    this.node = ganache.server();
  }

  async start() {
    await this.node.listen(8545);
    console.log(`server started ${JSON.stringify(this.node.address())}`);
    return this;
  }

  async stop() {
    await this.node.close();
    console.log(`server stopped ${JSON.stringify(this.node.status)}`);
  }
}

export async function startNode() {
  return await new TestNode().start();
}
