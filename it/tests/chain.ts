import { startNode } from "../src/nodeprovider";

let node: any;
beforeEach(async () => {
  node = await startNode();
});

afterEach(async () => {
  await node.stop();
});

describe("example test", () => {
  it("should pass 1", () => {
    expect(true).toBe(true);
  });
  it("should pass 2", () => {
    expect(true).toBe(true);
  });
});
