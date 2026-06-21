# 架构设计

[English](architecture.md) | [中文](architecture.zh-CN.md)

## 系统架构

```
┌──────────────────────────────────────────────────────────┐
│                     AI 决策层                              │
│  "我需要和 agent-b 协作，先看看谁在线"                       │
├──────────────────────────────────────────────────────────┤
│                   MCP 工具层 (stdio)                       │
│  16 个工具: get_online_agents, get_agent_info, ...         │
├───────────────┬────────────────────┬─────────────────────┤
│   P2P 模块     │    文件服务          │   频道管理           │
│  (WebSocket)  │ (分块/并发/流式写盘)  │  (纯内存/无状态)     │
│  + mDNS       │                    │  (P2P/Relay 双模式)  │
├───────────────┴────────────────────┴─────────────────────┤
│              插件系统 (认证/事件/Hook)                       │
├──────────────────────────────────────────────────────────┤
│            结构化日志 (slog) + 错误传播                      │
├──────────────────────────────────────────────────────────┤
│          mDNS 发现 + Relay 中转 (可选)                      │
│          + 自动重连 (指数退避)                               │
└──────────────────────────────────────────────────────────┘
```

## 连接模型

```
默认状态：零连接
  Agent A ──── mDNS ──── Agent B    (只感知在线，不建连)

AI 决定交互：
  Agent A ──WebSocket──> Agent B    (按需建连，用完可关)

跨子网场景：
  Agent A ──WS──> Relay ──WS──> Agent B

断线重连：
  Agent A ──退避──> Relay          (1s → 2s → 4s → ... → 30s 上限)
```

## 模块参考

| 模块 | 包 | 职责 | 关键特性 |
|------|-----|------|---------|
| `protocol` | `internal/protocol` | WebSocket 消息协议 | JSON 消息、连接封装、`BytesReader` 工具 |
| `p2p` | `internal/p2p` | P2P 传输 + mDNS 发现 | 按需连接、在线列表、WebSocket 服务端、**过期节点清理 (60s)** |
| `channel` | `internal/channel` | 频道管理 | 纯内存、无状态、Host 自动切换、P2P Lobby / Relay 双模式 |
| `fileservice` | `internal/fileservice` | 文件服务 | 分块传输、并发 Worker (4)、重试、文件夹同步、插件事件 |
| `filetransfer` | `internal/filetransfer` | 接收端文件组装 | **流式写盘 (WriteAt)**、SHA-256 校验、路径穿越防护 |
| `plugins` | `internal/plugins` | 插件系统 | 事件驱动 Hook、过滤/转换装饰器、**可选 AuthPlugin 接口** |
| `profile` | `internal/profile` | Agent 身份 | Profile 管理、持久化、A2A AgentCard 转换 |
| `logger` | `internal/logger` | 结构化日志 | slog JSON 输出、组件标签、级别过滤 |
| `mcp` | `internal/mcp` | MCP JSON-RPC 服务 | **16 个工具**、stdio 接口、JSON-RPC 2.0、**推送通知** |
| `adapter` | `internal/adapter` | A2A 协议适配 | AgentCard/Task/Message 映射、Profile ↔ Card 转换 |
| `relay` | `internal/relay` | Relay 客户端 | 跨子网中转、**自动重连 (指数退避)** |

## 数据流

### 消息发送

```
AI 调用 lan_send_message
  → MCP Server 解析参数
  → Agent 查找频道成员
  → 对每个成员：
      有连接 → 直接 WebSocket 发送
      无连接 → 返回错误 "call lan_open_connection first"
      Relay 模式 → 通过 Relay 服务器转发
```

### 文件传输 (发送端)

```
AI 调用 lan_share_file
  → FileService.SplitIntoChunks() 分块 (64KB)
  → 创建 Transfer 对象，记录 Chunk 元数据
  → 4 个 goroutine 并发发送
  → 每个 chunk: ReadChunk → WebSocket 发送
  → 插件触发 EventChunkDone
  → 全部完成 → Status=completed
  → 插件触发 EventTransferDone
```

### 文件接收 (流式写盘)

```
收到 file_meta 消息
  → PrepareIncoming(): 创建目标文件，初始化 Received map
  → 每个 file_data 消息：
      AddChunk(): WriteAt(chunk_data, offset) 直接写入磁盘
      标记 Received[chunkIdx] = true
  → file_done 消息：
      assemble(): 校验 SHA-256 哈希
      关闭文件句柄
      触发 OnComplete 回调
```

### 文件夹同步

```
AI 调用 lan_sync_folder
  → FolderSync.ScanFolder(): 扫描当前文件系统状态
  → FolderSync.LoadManifest(): 加载上次状态 (.lan-sync-manifest.json)
  → DiffFolders(): 计算 diff (adds / modifies / deletes)
  → 对 adds + modifies: FileService.SendFile()
  → 保存新 manifest 到 .lan-sync-manifest.json
  → 插件触发 EventFolderSyncDone
```

### Relay 自动重连

```
Relay 连接断开
  → readLoop() 检测到错误
  → 触发 reconnect() (除非已 Stop)
  → 指数退避: 1s → 2s → 4s → 8s → 16s → 30s (上限)
  → 成功: 重置重试计数器，重新注册，恢复回调
  → Stop(): 跳过重连
```

### 过期节点清理

```
每 30 秒：
  → 扫描 p.online map
  → 移除 LastSeen > 60 秒的节点
  → 记录日志
```

## MCP 工具 (共 16 个)

| 工具 | 描述 |
|------|------|
| `lan_get_online_agents` | 获取所有在线 Agent ID |
| `lan_get_agent_info` | 获取 peer 的详细 AgentCard 信息 |
| `lan_open_connection` | 打开到 peer 的 WebSocket 连接 |
| `lan_close_connection` | 关闭到 peer 的连接 |
| `lan_create_channel` | 创建频道并邀请 peer |
| `lan_leave_channel` | 离开频道 |
| `lan_list_channels` | 列出已加入的频道 |
| `lan_send_message` | 向频道发送消息 |
| `lan_share_file` | 分享文件（自动分块） |
| `lan_share_folder` | 分享文件夹（增量同步） |
| `lan_sync_folder` | 与远程 peer 同步文件夹 |
| `lan_get_transfer_status` | 获取传输状态 |
| `lan_list_transfers` | 列出活跃传输 |
| `lan_set_profile` | 更新 agent 名称/描述/技能 |
| `lan_subscribe` | 订阅事件通知 |
| `lan_unsubscribe` | 取消订阅事件 |

### 推送通知

MCP server 支持通过 stdout JSON-RPC notification 推送事件：

```json
{"jsonrpc":"2.0","method":"agent.online","params":{"peer_id":"agent-b"}}
```

使用 `lan_subscribe` 订阅，`lan_unsubscribe` 取消订阅。

## 插件系统

### 事件类型

| 事件 | 触发时机 |
|------|---------|
| `transfer.start` | 文件传输开始 |
| `chunk.done` | 单个 chunk 发送成功 |
| `transfer.done` | 文件传输完成 |
| `folder_sync.start` | 文件夹同步开始 |
| `folder_sync.done` | 文件夹同步完成 |
| `file.received` | 从 peer 接收到文件 |

### 认证插件 (可选)

实现 `AuthPlugin` 接口添加自定义认证：

```go
type AuthPlugin interface {
    Authenticate(agentID string, msg protocol.Message) error
}
```

通过 `manager.SetAuth(plugin)` 设置。未设置时所有连接均允许（默认 LAN 行为）。

### 内置插件

- **LogPlugin** — 记录所有事件
- **ProgressPlugin** — 通过回调报告进度
- **FilterPlugin** — 按谓词过滤事件
- **TransformPlugin** — 在传递给内部插件前转换事件数据

## 错误处理

- 所有 `conn.Send()` 调用检查并记录错误
- Relay 重连优雅处理瞬态故障
- 文件接收流式写盘（无内存缓冲）
- 路径穿越防护 (sanitizeFilename)
- 所有操作使用结构化日志 (slog)

## 安全考量

- **仅限 LAN**: 设计用于可信局域网
- **无认证**: 默认任何设备可连接
- **可选认证**: 提供插件接口用于自定义认证
- **无加密**: WebSocket 连接未加密 (ws://)
- **路径穿越防护**: 文件名写入前经过清理

## 配置

### Agent

| 参数 | 默认值 | 描述 |
|------|--------|------|
| `-id` | `<hostname>-<pid>` | Agent ID |
| `-port` | `19100 + PID%1000` | WebSocket 监听端口 |

### Relay

| 参数 | 默认值 | 描述 |
|------|--------|------|
| `-addr` | `:19200` | WebSocket 监听地址 |
| `-http` | `:19201` | HTTP 状态页地址 |

## 技术栈

- **语言**: Go 1.25+
- **传输**: WebSocket (gorilla/websocket)
- **发现**: mDNS (grandcat/zeroconf)
- **日志**: slog (结构化 JSON)
- **协议**: MCP JSON-RPC 2.0
- **构建**: Make, Docker (多阶段)
- **CI**: GitHub Actions (lint, test, build, docker)
