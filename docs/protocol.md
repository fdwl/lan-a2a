# LanA2A - Wire Protocol

## Overview

LanA2A uses **WebSocket** for all communication between peers (agent-to-agent P2P) and between agents and the relay server. Messages are sent as **JSON objects** over WebSocket frames.

## Transport

- **P2P**: WebSocket on configurable port (default `19100 + PID%1000`)
- **Relay**: WebSocket on configurable port (default `19200`)
- **Discovery**: mDNS (Bonjour/Zeroconf) service type `_lan-agent-bus._tcp`

## Message Schema

```json
{
  "type": "<string>",
  "id": "<string>",
  "from": "<string>",
  "channel_id": "<string, optional>",
  "content": "<string, optional>",
  "ts": "<int64, unix timestamp>",
  "filename": "<string, optional>",
  "file_size": "<int64, optional>",
  "checksum": "<string, optional>",
  "chunk_idx": "<int, optional>",
  "total_chunks": "<int, optional>",
  "data": "<bytes, optional>",
  "profile": "<ProfilePayload, optional>"
}
```

## Message Types

| Type | Direction | Purpose |
|------|-----------|---------|
| `register` | Initiator → Responder | Handshake: announce identity + profile |
| `register_ok` | Responder → Initiator | Acknowledge registration + return profile |
| `ping` | Either | Keepalive request |
| `pong` | Either | Keepalive response |
| `text` | Agent → Channel | Text/JSON message to channel |
| `file_meta` | Agent → Channel | File transfer start (metadata) |
| `file_data` | Agent → Channel | File chunk (binary data) |
| `file_done` | Agent → Channel | File transfer complete |
| `query_online` | Agent → Relay | Request online agent list |
| `online_list` | Relay → Agent | Response with online agent IDs |

## Handshake

1. Initiator connects via WebSocket to `ws://<addr>/ws`
2. Initiator sends `register` message with `from` = agent ID and `profile`
3. Responder responds with `register_ok` and its own `profile`
4. Connection is now established

```
Agent A                    Agent B (or Relay)
  |--- register ----------->|
  |<------ register_ok -----|
  |                          |
  |<------ ping ------------|
  |--- pong --------------->|
  |                          |
  |--- text (channel_msg)-->|
```

## Channel Protocol

### Create Channel
```json
{
  "type": "text",
  "channel_id": "<generated_channel_id>",
  "content": "{\"event\":\"channel_created\",\"channel_id\":\"...\",\"channel_name\":\"...\",\"mode\":\"p2p\",\"host\":\"...\",\"members\":[\"...\"]}",
  "from": "agent-a",
  "ts": 1718971234
}
```

### Leave Channel
```json
{
  "type": "text",
  "channel_id": "<channel_id>",
  "content": "{\"event\":\"peer_left\",\"channel_id\":\"...\",\"peer_id\":\"...\"}",
  "from": "agent-a",
  "ts": 1718971234
}
```

## File Transfer Protocol

1. Sender sends `file_meta` with filename, size, checksum, total_chunks
2. Sender sends N `file_data` messages (each containing binary chunk data)
3. Sender sends `file_done` to confirm completion

Receiver reassembles chunks in order and verifies checksum (SHA-256).

## Discovery (mDNS)

Service type: `_lan-agent-bus._tcp`

TXT records:
- `id=<agent_id>`
- `version=0.1.0`

Relay service type: `_lan-agent-bus-relay._tcp`

## Relay Server Protocol

Agents connect to the relay server using the same WebSocket handshake (`register`/`register_ok`).

The server:
1. Maintains an online agent registry
2. Maintains a channel index (channelID → set of agentIDs)
3. Relays `text`, `file_meta`, `file_data`, `file_done` messages only to channel members
4. Responds to `ping` with `pong`
5. Responds to `query_online` with `online_list`

## Profile Payload

```json
{
  "id": "<agent_id>",
  "name": "<display_name>",
  "avatar": "<optional>",
  "roles": ["role1", "role2"],
  "tags": ["tag1"],
  "metadata": {"key": "value"}
}
```

## MCP Tools (AI Interface)

| Tool | Input | Output |
|------|-------|--------|
| `lan_get_online_agents` | `{}` | `{"agents": [...], "count": N}` |
| `lan_create_channel` | `{"channel_name": "...", "peer_ids": [...]}` | `{"channel_id": "...", "members": [...]}` |
| `lan_leave_channel` | `{"channel_id": "..."}` | `{"status": "left"}` |
| `lan_send_message` | `{"channel_id": "...", "message_body": "..."}` | `{"status": "sent", "recipients": N}` |
| `lan_share_file` | `{"channel_id": "...", "file_path": "..."}` | `{"status": "shared", "filename": "...", "file_size": N}` |
| `lan_share_folder` | `{"channel_id": "...", "folder_path": "..."}` | `{"status": "sharing", "folder": "..."}` |
| `lan_sync_folder` | `{"folder_path": "...", "peer_id": "..."}` | `{"status": "syncing", "folder": "..."}` |
| `lan_get_transfer_status` | `{"transfer_id": "..."}` | Transfer status object |
| `lan_list_transfers` | `{}` | `{"transfers": [...], "count": N}` |
