# Architecture

[English](architecture.md) | [中文](architecture.zh-CN.md)

## System Overview

```
┌──────────────────────────────────────────────────────────┐
│                     AI Decision Layer                     │
│  "I need to collaborate with agent-b, let me check who   │
│   is online first"                                       │
├──────────────────────────────────────────────────────────┤
│                  MCP Tool Layer (stdio)                   │
│  16 tools: get_online_agents, get_agent_info, ...          │
├───────────────┬────────────────────┬─────────────────────┤
│   P2P Module  │    File Service     │   Channel Mgmt      │
│  (WebSocket)  │ (chunk/concurrent/ │  (in-memory/stateless)│
│  + mDNS       │  streaming-to-disk)│  (P2P/Relay dual)   │
├───────────────┴────────────────────┴─────────────────────┤
│         Plugin System (Auth / Events / Hooks)             │
├──────────────────────────────────────────────────────────┤
│     Structured Logging (slog) + Error Propagation         │
├──────────────────────────────────────────────────────────┤
│          mDNS Discovery + Relay (optional)                 │
│          + Automatic Reconnection (backoff)               │
└──────────────────────────────────────────────────────────┘
```

## Connection Model

```
Default: Zero connections
  Agent A ──── mDNS ──── Agent B    (aware of online status only)

AI decides to interact:
  Agent A ──WebSocket──> Agent B    (on-demand, close when done)

Cross-subnet scenario:
  Agent A ──WS──> Relay ──WS──> Agent B

Reconnection (on disconnect):
  Agent A ──backoff──> Relay        (1s → 2s → 4s → ... → 30s max)
```

## Module Reference

| Module | Package | Responsibility | Key Features |
|--------|---------|---------------|--------------|
| `protocol` | `internal/protocol` | WebSocket message protocol | JSON messages, connection wrapper, `BytesReader` utility |
| `p2p` | `internal/p2p` | P2P transport + mDNS discovery | On-demand connections, online list, WebSocket server, **stale peer cleanup (60s)** |
| `channel` | `internal/channel` | Channel management | In-memory, stateless, host promotion on leave, P2P Lobby / Relay dual mode |
| `fileservice` | `internal/fileservice` | File service | Chunked transfer, concurrent workers (4), retry, folder sync, plugin events |
| `filetransfer` | `internal/filetransfer` | Incoming file reassembly | **Streaming to disk (WriteAt)**, SHA-256 verification, path traversal protection |
| `plugins` | `internal/plugins` | Plugin system | Event-driven hooks, filter/transform decorators, **optional AuthPlugin interface** |
| `profile` | `internal/profile` | Agent identity | Profile management, persistence, A2A AgentCard conversion |
| `logger` | `internal/logger` | Structured logging | slog JSON output, component tags, level filtering |
| `mcp` | `internal/mcp` | MCP JSON-RPC server | **16 tools**, stdio interface, JSON-RPC 2.0, **push notifications** |
| `adapter` | `internal/adapter` | A2A protocol adapter | AgentCard/Task/Message mapping, Profile ↔ Card conversion |
| `relay` | `internal/relay` | Relay client | Cross-subnet relay, **automatic reconnection with exponential backoff** |

## Data Flow

### Message Sending

```
AI calls lan_send_message
  → MCP Server parses arguments
  → Agent looks up channel members
  → For each member:
      Has connection → direct WebSocket send
      No connection → error "call lan_open_connection first"
      Relay mode → forward through relay server
```

### File Transfer (Sending)

```
AI calls lan_share_file
  → FileService.SplitIntoChunks() splits file (64KB chunks)
  → Creates Transfer object with Chunk metadata
  → 4 goroutines concurrently send chunks
  → Each chunk: ReadChunk → WebSocket send
  → Plugin: EventChunkDone per chunk
  → All done → Status=completed
  → Plugin: EventTransferDone
```

### File Reception (Streaming to Disk)

```
Incoming file_meta message
  → PrepareIncoming(): create destination file, initialize Received map
  → Each file_data message:
      AddChunk(): WriteAt(chunk_data, offset) directly to disk
      Mark Received[chunkIdx] = true
  → file_done message:
      assemble(): verify SHA-256 checksum against written data
      Close file handle
      Trigger OnComplete callback
```

### Folder Sync

```
AI calls lan_sync_folder
  → FolderSync.ScanFolder(): scan current filesystem state
  → FolderSync.LoadManifest(): load previous state from .lan-sync-manifest.json
  → DiffFolders(): compute diff (adds / modifies / deletes)
  → For adds + modifies: FileService.SendFile()
  → Save new manifest to .lan-sync-manifest.json
  → Plugin: EventFolderSyncDone
```

### Relay Reconnection

```
Relay connection lost
  → readLoop() detects error
  → reconnect() triggered (unless stopped)
  → Exponential backoff: 1s → 2s → 4s → 8s → 16s → 30s (max)
  → On success: reset retry counter, re-register, restore callbacks
  → On Stop(): skip reconnection
```

### Stale Peer Cleanup

```
Every 30 seconds:
  → Scan p.online map
  → Remove peers where LastSeen > 60 seconds ago
  → Log removal for debugging
```

## MCP Tools (16 total)

| Tool | Description |
|------|-------------|
| `lan_get_online_agents` | Get all online agent IDs |
| `lan_get_agent_info` | Get detailed AgentCard info for a peer |
| `lan_open_connection` | Open WebSocket connection to a peer |
| `lan_close_connection` | Close connection to a peer |
| `lan_create_channel` | Create channel and invite peers |
| `lan_leave_channel` | Leave a channel |
| `lan_list_channels` | List channels the agent has joined |
| `lan_send_message` | Send message to a channel |
| `lan_share_file` | Share file (auto-chunked) |
| `lan_share_folder` | Share folder (incremental sync) |
| `lan_sync_folder` | Sync folder with remote peer |
| `lan_get_transfer_status` | Get transfer status |
| `lan_list_transfers` | List active transfers |
| `lan_set_profile` | Update agent name/description/skills |
| `lan_subscribe` | Subscribe to event notifications |
| `lan_unsubscribe` | Unsubscribe from events |

### Push Notifications

MCP server supports push notifications via stdout JSON-RPC notifications:

```json
{"jsonrpc":"2.0","method":"agent.online","params":{"peer_id":"agent-b"}}
```

Subscribe with `lan_subscribe`, unsubscribe with `lan_unsubscribe`.

## Plugin System

### Event Types

| Event | Trigger |
|-------|---------|
| `transfer.start` | File transfer begins |
| `chunk.done` | Individual chunk sent successfully |
| `transfer.done` | File transfer completed |
| `folder_sync.start` | Folder sync begins |
| `folder_sync.done` | Folder sync completed |
| `file.received` | File received from peer |

### Auth Plugin (Optional)

Implement the `AuthPlugin` interface to add custom authentication:

```go
type AuthPlugin interface {
    Authenticate(agentID string, msg protocol.Message) error
}
```

Set via `manager.SetAuth(plugin)`. If not set, all connections are allowed (default LAN behavior).

### Built-in Plugins

- **LogPlugin** — Logs all events
- **ProgressPlugin** — Reports progress via callback
- **FilterPlugin** — Filters events by predicate
- **TransformPlugin** — Transforms event data before passing to inner plugin

## Error Handling

- All `conn.Send()` calls check and log errors
- Relay reconnection handles transient failures gracefully
- File reception streams to disk (no in-memory buffering)
- Path traversal protection via `sanitizeFilename()`
- Structured logging via `slog` for all operations

## Security Considerations

- **LAN Only**: Designed for trusted local networks
- **No Authentication**: By default, any device can connect
- **Optional Auth**: Plugin interface available for custom auth
- **No Encryption**: WebSocket connections are unencrypted (ws://)
- **Path Traversal**: Filenames sanitized before writing to disk

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

## Technology Stack

- **Language**: Go 1.25+
- **Transport**: WebSocket (gorilla/websocket)
- **Discovery**: mDNS (grandcat/zeroconf)
- **Logging**: slog (structured, JSON)
- **Protocol**: MCP JSON-RPC 2.0
- **Build**: Make, Docker (multi-stage)
- **CI**: GitHub Actions (lint, test, build, docker)
