#!/usr/bin/env node
import * as os from "os";
import { P2P } from "./p2p.js";
import { createMCPServer } from "./mcp.js";

function main() {
  const agentID = process.argv.includes("--id")
    ? process.argv[process.argv.indexOf("--id") + 1]
    : `${os.hostname()}-${process.pid}`;

  const portArg = process.argv.includes("--port")
    ? parseInt(process.argv[process.argv.indexOf("--port") + 1])
    : 19100 + (process.pid % 1000);

  const shortID = agentID.length > 8 ? agentID.slice(0, 8) : agentID;
  const log = (tag: string, msg: string) => console.log(`[agent:${shortID}] [${tag}] ${msg}`);

  const p2p = new P2P(agentID, portArg);

  // Handle incoming messages
  p2p.setMessageHandler((msg, from) => {
    if (msg.type === "text" && msg.channel_id) {
      log("recv", `ch=${msg.channel_id.slice(0, 12)} from=${from.slice(0, 8)}: ${msg.content?.slice(0, 80)}`);
    } else if (msg.type === "file_meta") {
      log("recv", `file meta from=${from.slice(0, 8)}: ${msg.filename}`);
    } else if (msg.type === "file_done") {
      log("recv", `file done from=${from.slice(0, 8)}: ${msg.filename}`);
    }
  });

  p2p.setFileDataHandler((msg, from) => {
    log("recv", `file chunk from=${from.slice(0, 8)} idx=${msg.chunk_idx}`);
  });

  // Auto-connect to relay when found
  let relayConnected = false;
  p2p.setRelayHandler(async (relayAddr) => {
    if (relayConnected) return;
    log("discovery", `found relay at ${relayAddr}, connecting...`);
    // TODO: implement relay client connection for TS
    relayConnected = true;
  });

  // Start P2P
  p2p.start().then(() => {
    log("init", `ready on port ${portArg}`);

    // Create MCP server and run on stdio
    const mcpServer = createMCPServer(agentID, p2p);
    mcpServer.connect({} as any); // StdioServerTransport handles stdio
  });
}

main();
