import { ChannelCredentials } from "@grpc/grpc-js";
import { GrpcTransport } from "@protobuf-ts/grpc-transport";
import { UbtChainServiceClient, UbtBlockServiceClient, UbtConstructServiceClient, UbtCurrencyServiceClient, UbtAccountManagerClient } from "@ubtr/sdk";

export class UbtClient {
  readonly transport: GrpcTransport;
  constructor(readonly host: string) {
    this.transport = new GrpcTransport({
      host: host,
      channelCredentials: ChannelCredentials.createInsecure(),
    });
  }

  chainService(): UbtChainServiceClient {
    return new UbtChainServiceClient(this.transport);
  }

  blockService(): UbtBlockServiceClient {
    return new UbtBlockServiceClient(this.transport);
  }

  constructService(): UbtConstructServiceClient {
    return new UbtConstructServiceClient(this.transport);
  }

  currencyService(): UbtCurrencyServiceClient {
    return new UbtCurrencyServiceClient(this.transport);
  }

  accountManager(): UbtAccountManagerClient {
    return new UbtAccountManagerClient(this.transport);
  }
}