import * as Viam from '@viamrobotics/sdk';
// import type { Resource, Options } from '@viamrobotics/sdk';
import { Struct, type JsonValue } from '@bufbuild/protobuf';
import { type Client } from '@connectrpc/connect';
import { videostoreService } from './grpc/src/proto/videostore_connect.js';
import * as pb from './grpc/src/proto/videostore_pb.js';

export interface Videostore extends Viam.Resource {
    fetch(from: string, to: string): Promise<Uint8Array>;
    save(from: string, to: string): Promise<string>;
    fetchStream(
        from: string,
        to: string,
        onChunk: (chunk: Uint8Array) => void
    ): Promise<void>;
    doCommand(command: Struct): Promise<JsonValue>;
}


export class VideostoreClient extends Viam.Client {
    private client: Client<typeof videostoreService>;
    private readonly name: string;
    private readonly options: Viam.Options;

    constructor(client: Viam.RobotClient, name: string, options: Viam.Options = {}) {
        super(name);
        this.client = client.createServiceClient(videostoreService);
        this.name = name;
        this.options = options;
    }

    async fetch(from: string, to: string, container: string): Promise<Uint8Array> {
        const req = new pb.FetchRequest({
            name: this.name,
            from: from,
            to: to,
            container: container,
        });

        this.options.requestLogger?.(req);
        const res = await this.client.fetch(req);
        return res.videoData;
    }

    async save(from: string, to: string, container: string): Promise<string> {
        const req = new pb.SaveRequest({
            name: this.name,
            from: from,
            to: to,
            container: container,
        });

        this.options.requestLogger?.(req);
        const res = await this.client.save(req);
        return res.filename;
    }

    async fetchStream(
        from: string,
        to: string,
        container: string,
        onChunk: (chunk: Uint8Array) => void
    ): Promise<void> {
        const req = new pb.FetchStreamRequest({
            name: this.name,
            from: from,
            to: to,
            container: container,
        });
        console.log("fetchStream request:", req);
        this.options.requestLogger?.(req);
        try {
            const stream = this.client.fetchStream(req);
            for await (const res of stream) {
                // console.log(`fetchStream response #${res}`); // Log each response
                onChunk(res.videoData);
            }
        } catch (error) {
            this.options.requestLogger?.(error);
            throw error;
        }
    }

    async doCommand(command: Struct): Promise<JsonValue> {
        return {};
    }
}
