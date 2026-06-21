import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { P2P } from "./p2p.js";
import * as fs from "fs";
import * as path from "path";
import * as crypto from "crypto";

export function createMCPServer(agentID: string, p2p: P2P): McpServer {
  const server = new McpServer({
    name: "lan-agent-bus",
    version: "0.1.0",
  });

  server.tool("lan_get_online_agents", "Get all online AI agents in the local network", {}, async () => {
    const agents = p2p.getOnlinePeers();
    return {
      content: [{ type: "text", text: JSON.stringify({ agents, count: agents.length }) }],
    };
  });

  server.tool(
    "lan_open_connection",
    "Open a WebSocket connection to a peer agent. Must be called before sending messages or files.",
    { peer_id: z.string().describe("Agent ID to connect to") },
    async ({ peer_id }) => {
      try {
        await p2p.connectTo(peer_id);
        return { content: [{ type: "text", text: JSON.stringify({ status: "connected", peer_id }) }] };
      } catch (e: any) {
        return { content: [{ type: "text", text: e.message }], isError: true };
      }
    }
  );

  server.tool(
    "lan_close_connection",
    "Close the WebSocket connection to a peer agent",
    { peer_id: z.string().describe("Agent ID to disconnect from") },
    async ({ peer_id }) => {
      p2p.disconnectFrom(peer_id);
      return { content: [{ type: "text", text: JSON.stringify({ status: "disconnected", peer_id }) }] };
    }
  );

  server.tool(
    "lan_create_channel",
    "Create a communication channel and invite peer agents",
    {
      channel_name: z.string().describe("Name of the channel"),
      peer_ids: z.array(z.string()).describe("List of peer agent IDs to invite"),
    },
    async ({ channel_name, peer_ids }) => {
      for (const pid of peer_ids) {
        if (!p2p.isConnected(pid)) {
          return { content: [{ type: "text", text: `Peer ${pid} not connected` }], isError: true };
        }
      }
      const channelID = `ch-${Date.now()}-${agentID}`;
      const members = [agentID, ...peer_ids];
      const msg = {
        type: "text" as const,
        id: crypto.randomBytes(8).toString("hex"),
        from: agentID,
        channel_id: channelID,
        content: JSON.stringify({ event: "channel_created", channel_id: channelID, channel_name, members }),
        ts: Date.now(),
      };
      p2p.broadcast(peer_ids, msg);
      return {
        content: [{ type: "text", text: JSON.stringify({ channel_id: channelID, members }) }],
      };
    }
  );

  server.tool(
    "lan_leave_channel",
    "Leave a communication channel",
    { channel_id: z.string().describe("ID of the channel to leave") },
    async ({ channel_id }) => {
      const msg = {
        type: "text" as const,
        id: crypto.randomBytes(8).toString("hex"),
        from: agentID,
        channel_id,
        content: JSON.stringify({ event: "peer_left", channel_id, peer_id: agentID }),
        ts: Date.now(),
      };
      p2p.broadcast(p2p.getPeers(), msg);
      return { content: [{ type: "text", text: JSON.stringify({ status: "left", channel_id }) }] };
    }
  );

  server.tool(
    "lan_send_message",
    "Send a message to a channel",
    {
      channel_id: z.string().describe("ID of the channel"),
      message_body: z.string().describe("Message content"),
    },
    async ({ channel_id, message_body }) => {
      const msg = {
        type: "text" as const,
        id: crypto.randomBytes(8).toString("hex"),
        from: agentID,
        channel_id,
        content: message_body,
        ts: Date.now(),
      };
      const peers = p2p.getPeers();
      const failed = p2p.broadcast(peers, msg);
      return {
        content: [{ type: "text", text: JSON.stringify({ status: "sent", channel_id, recipients: peers.length, failed }) }],
      };
    }
  );

  server.tool(
    "lan_share_file",
    "Share a local file to a channel",
    {
      channel_id: z.string().describe("ID of the channel"),
      file_path: z.string().describe("Local file path to share"),
    },
    async ({ channel_id, file_path }) => {
      const absPath = path.resolve(file_path);
      const stat = fs.statSync(absPath);
      const data = fs.readFileSync(absPath);
      const filename = path.basename(absPath);
      const checksum = crypto.createHash("sha256").update(data).digest("hex");
      const chunkSize = 64 * 1024;
      const chunks: Buffer[] = [];
      for (let i = 0; i < data.length; i += chunkSize) {
        chunks.push(data.subarray(i, i + chunkSize));
      }

      const metaID = crypto.randomBytes(8).toString("hex");
      const peers = p2p.getPeers();

      // meta
      p2p.broadcast(peers, {
        type: "file_meta", id: metaID, from: agentID, channel_id,
        filename, file_size: stat.size, checksum, total_chunks: chunks.length, ts: Date.now(),
      });
      // chunks
      for (let i = 0; i < chunks.length; i++) {
        p2p.broadcast(peers, {
          type: "file_data", id: crypto.randomBytes(8).toString("hex"), from: agentID,
          channel_id, chunk_idx: i, ts: Date.now(), data: chunks[i],
        });
      }
      // done
      p2p.broadcast(peers, {
        type: "file_done", id: metaID, from: agentID, channel_id,
        filename, checksum, ts: Date.now(),
      });

      return {
        content: [{ type: "text", text: JSON.stringify({ status: "shared", filename, file_size: stat.size, chunks: chunks.length, checksum }) }],
      };
    }
  );

  return server;
}
