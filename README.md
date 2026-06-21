# LanA2A

[English](README.md) | [中文](docs/README.zh-CN.md) | [日本語](docs/README.ja.md)

**A2A Protocol for LAN** — A decentralized LAN communication service for AI Agents based on the [Agent2Agent (A2A)](https://a2a-protocol.org) standard protocol.

> Developed by MiMo AI. Powered by MiMo AI.

## Key Features

- **Standard Protocol**: Built on the A2A Go SDK, not reinventing the wheel
- **LAN Optimized**: mDNS discovery + WebSocket transport (A2A uses HTTP natively)
- **On-Demand Connections**: Zero connections by default, AI decides when to connect
- **Dual Mode**: P2P Lobby (no server) + Relay (with server)
- **File Transfer**: Chunked concurrent transfers with folder sync support
- **MCP Interface**: 11 tools for AI agents to communicate over LAN

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                 │
│  AgentCard / Task / Message / Skills                 │
├──────────────┬──────────────────┬───────────────────┤
│  LAN Transport │   File Service   │   Channel Mgmt    │
│  mDNS + WS    │  (chunk/conc/sync)│  (P2P/Relay)     │
├──────────────┴──────────────────┴───────────────────┤
│              A2A Go SDK (a2asrv / a2aclient)         │
└─────────────────────────────────────────────────────┘
```

## Quick Start

```bash
# Build
make build

# Run Agent
./lan-a2a -id agent-a -port 19100

# Run Relay (optional, for cross-subnet scenarios)
./lan-relay -addr :19200
```

## A2A Compatibility

| A2A Concept | LanA2A Implementation |
|-------------|----------------------|
| AgentCard | Profile + LAN broadcast |
| Agent Discovery | mDNS (LAN) + Relay query |
| Task Lifecycle | Channel-based task tracking |
| Message/Part | WebSocket JSON messages |
| Transport | WebSocket (LAN optimized) |
| Skills | Profile.Roles |

## MCP Tools

| Tool | Description |
|------|-------------|
| `lan_get_online_agents` | Get online agents (with AgentCard info) |
| `lan_open_connection` | Open WebSocket connection |
| `lan_close_connection` | Close connection |
| `lan_create_channel` | Create channel (P2P Lobby or Relay mode) |
| `lan_send_message` | Send message |
| `lan_share_file` | Share file (auto-chunked for large files) |
| `lan_share_folder` | Share folder (incremental sync) |
| `lan_sync_folder` | Sync folder with remote peer |
| `lan_get_transfer_status` | Get transfer status |
| `lan_list_transfers` | List active transfers |

## Project Structure

```
lan-a2a/
├── cmd/
│   ├── lan-a2a/           # Agent client
│   ├── lan-relay/         # Relay server
│   └── gendoc/            # Documentation generator
├── internal/
│   ├── adapter/           # A2A protocol adapter
│   ├── protocol/          # WebSocket message protocol
│   ├── channel/           # Channel management (P2P Lobby / Relay)
│   ├── fileservice/       # File service (chunk/concurrent/sync)
│   ├── filetransfer/      # Incoming file reassembly
│   ├── plugins/           # Plugin system
│   ├── profile/           # Agent identity (→ A2A AgentCard)
│   ├── logger/            # Structured logging (slog)
│   ├── mcp/               # MCP JSON-RPC server
│   ├── p2p/               # P2P transport + mDNS
│   └── relay/             # Relay client
├── clients/typescript/    # TypeScript client
├── docs/
│   ├── openapi.yaml       # OpenAPI specification
│   └── protocol.md        # Wire protocol specification
└── Dockerfile
```

## Docker

```bash
# Build agent image
docker build --target agent -t lan-a2a:latest .

# Build relay image
docker build --target relay -t lan-relay:latest .

# Run relay
docker run -d --name lan-relay -p 19200:19200 -p 19201:19201 lan-relay:latest
```

## Configuration

### Agent

| Flag | Default | Description |
|------|---------|-------------|
| `-id` | `<hostname>-<pid>` | Agent ID |
| `-port` | `19100 + PID%1000` | WebSocket listen port |

### Relay

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:19200` | WebSocket listen address |
| `-http` | `:19201` | HTTP status address |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

MIT License. See [LICENSE](LICENSE).

---

<p align="center">
  Built with ❤️ by <a href="https://github.com/XiaomiMiMo">MiMo AI</a>
</p>
