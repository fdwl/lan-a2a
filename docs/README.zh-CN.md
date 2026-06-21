# LanA2A

[English](../README.md) | [中文](README.zh-CN.md) | [日本語](README.ja.md)

**A2A 协议的局域网实现** — 基于 [Agent2Agent (A2A)](https://a2a-protocol.org) 标准协议，为 AI Agent 打造的去中心化局域网通信服务。

> 由 MiMo AI 开发。

## 核心特性

- **标准协议**：基于 A2A Go SDK，不造轮子
- **LAN 优化**：mDNS 发现 + WebSocket 传输（A2A 原生走 HTTP）
- **按需连接**：默认零连接，AI 决定何时建连
- **双模式**：P2P Lobby（无服务器）+ Relay 中转（有服务器）
- **文件传输**：分块并发传输 + 文件夹增量同步
- **MCP 接口**：11 个工具供 AI Agent 局域网通信

## 架构

```
┌─────────────────────────────────────────────────────┐
│                    A2A 协议层                         │
│  AgentCard / Task / Message / Skills                 │
├──────────────┬──────────────────┬───────────────────┤
│  LAN 传输适配  │   文件服务        │   频道管理          │
│  mDNS + WS   │  (分块/并发/sync) │  (P2P/Relay双模式)  │
├──────────────┴──────────────────┴───────────────────┤
│              A2A Go SDK (a2asrv / a2aclient)         │
└─────────────────────────────────────────────────────┘
```

## 快速开始

```bash
# 编译
make build

# 运行 Agent
./lan-a2a -id agent-a -port 19100

# 运行 Relay（可选，跨子网场景）
./lan-relay -addr :19200
```

## A2A 兼容性

| A2A 概念 | LanA2A 实现 |
|----------|-------------|
| AgentCard | Profile + LAN 广播 |
| Agent Discovery | mDNS（LAN）+ Relay 查询 |
| Task Lifecycle | Channel 内任务追踪 |
| Message/Part | WebSocket JSON 消息 |
| Transport | WebSocket（LAN 优化） |
| Skills | Profile.Roles |

## MCP 工具

| 工具 | 描述 |
|------|------|
| `lan_get_online_agents` | 获取在线 Agent（含 AgentCard 信息） |
| `lan_open_connection` | 打开 WebSocket 连接 |
| `lan_close_connection` | 关闭连接 |
| `lan_create_channel` | 创建频道（P2P Lobby 或 Relay 模式） |
| `lan_send_message` | 发送消息 |
| `lan_share_file` | 分享文件（大文件自动分块） |
| `lan_share_folder` | 分享文件夹（增量同步） |
| `lan_sync_folder` | 与远程 peer 同步文件夹 |
| `lan_get_transfer_status` | 查询传输状态 |
| `lan_list_transfers` | 列出所有活跃传输 |

## 项目结构

```
lan-a2a/
├── cmd/
│   ├── lan-a2a/           # Agent 客户端
│   ├── lan-relay/         # Relay 服务器
│   └── gendoc/            # 文档生成器
├── internal/
│   ├── adapter/           # A2A 协议适配层
│   ├── protocol/          # WebSocket 消息协议
│   ├── channel/           # 频道管理（P2P Lobby / Relay）
│   ├── fileservice/       # 文件服务（分块/并发/同步）
│   ├── filetransfer/      # 接收端文件组装
│   ├── plugins/           # 插件系统
│   ├── profile/           # Agent 身份（→ A2A AgentCard）
│   ├── logger/            # 结构化日志（slog）
│   ├── mcp/               # MCP JSON-RPC 服务
│   ├── p2p/               # P2P 传输 + mDNS
│   └── relay/             # Relay 客户端
├── clients/typescript/    # TypeScript 客户端
├── docs/
│   ├── openapi.yaml       # OpenAPI 规范
│   └── protocol.md        # 协议规范
└── Dockerfile
```

## Docker

```bash
# 构建 Agent 镜像
docker build --target agent -t lan-a2a:latest .

# 构建 Relay 镜像
docker build --target relay -t lan-relay:latest .

# 运行 Relay
docker run -d --name lan-relay -p 19200:19200 -p 19201:19201 lan-relay:latest
```

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

## 贡献

请参阅 [CONTRIBUTING.md](../CONTRIBUTING.md)。

## 安全

请参阅 [SECURITY.md](../SECURITY.md)。

## 更新日志

请参阅 [CHANGELOG.md](../CHANGELOG.md)。

## 许可证

MIT 许可证。详见 [LICENSE](../LICENSE)。

---

<p align="center">
  由 <a href="https://github.com/XiaomiMiMo">MiMo AI</a> 用心打造
</p>
