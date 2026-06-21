# LanA2A - A2A 协议局域网实现

## 概述
LanA2A 是 Agent2Agent (A2A) 协议的局域网实现，基于 A2A Go SDK，
为 AI Agent 提供去中心化通信能力。

## 架构
- **A2A 协议层**：AgentCard、Task、Message、Skills
- **LAN 传输**：mDNS 发现 + WebSocket 连接
- **双模式**：P2P Lobby（无服务器）+ Relay 中转（有服务器）
- **按需连接**：默认零连接，AI 决定何时建连

## MCP 工具

### 连接管理
- `lan_get_online_agents` — 获取在线 Agent（含 AgentCard 信息）
- `lan_open_connection(peer_id)` — 打开连接（必须先调用）
- `lan_close_connection(peer_id)` — 关闭连接

### 频道通信
- `lan_create_channel(channel_name, peer_ids[])` — 创建频道
- `lan_leave_channel(channel_id)` — 退出频道
- `lan_send_message(channel_id, message_body)` — 发送消息

### 文件服务
- `lan_share_file(channel_id, file_path)` — 分享文件
- `lan_share_folder(channel_id, folder_path)` — 分享文件夹
- `lan_sync_folder(folder_path, peer_id)` — 同步文件夹

### 传输管理
- `lan_get_transfer_status(transfer_id)` — 查询传输状态
- `lan_list_transfers()` — 列出所有传输

## A2A 兼容
- Profile → AgentCard（身份和能力声明）
- Channel → Task（协作任务）
- Message → A2A Message（标准消息格式）
- Skills → Profile.Roles（Agent 能力标签）

## 典型工作流
```
1. lan_get_online_agents → 获取在线 Agent 列表
2. lan_open_connection(peer_id="agent-b") → 建立连接
3. lan_create_channel(channel_name="前端协作", peer_ids=["agent-b"]) → 创建频道
4. lan_send_message(channel_id="ch-xxx", message_body="Hello!") → 发送消息
```
