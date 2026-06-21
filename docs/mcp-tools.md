# LAN Agent Bus - MCP 工具接口文档

## 工具列表

| 工具名 | 描述 |
|--------|------|
| `lan_get_online_agents` | 获取局域网内所有在线的 AI 节点 |
| `lan_open_connection` | 打开到指定 peer 的 WebSocket 连接。发送消息或文件前必须先调用此工具。 |
| `lan_close_connection` | 关闭到指定 peer 的 WebSocket 连接 |
| `lan_create_channel` | 创建通信频道并邀请 peer 加入 |
| `lan_leave_channel` | 退出通信频道 |
| `lan_send_message` | 向频道发送消息。频道内所有成员的连接必须已打开。 |
| `lan_share_file` | 分享本地文件到频道。支持大文件（自动分块并发传输）。 |
| `lan_share_folder` | 分享整个文件夹到频道。自动检测变化，只传输修改过的文件。 |
| `lan_sync_folder` | 与远程 peer 同步本地文件夹。扫描变化，只发送 diff。 |
| `lan_get_transfer_status` | 查询文件传输状态 |
| `lan_list_transfers` | 列出所有活跃的文件传输 |

## 详细接口

### `lan_get_online_agents`

**描述：** 获取局域网内所有在线的 AI 节点

**输入参数：**
```json
{
  "properties": {},
  "type": "object"
}
```

---

### `lan_open_connection`

**描述：** 打开到指定 peer 的 WebSocket 连接。发送消息或文件前必须先调用此工具。

**输入参数：**
```json
{
  "properties": {
    "peer_id": {
      "description": "要连接的 Agent ID",
      "type": "string"
    }
  },
  "required": [
    "peer_id"
  ],
  "type": "object"
}
```

---

### `lan_close_connection`

**描述：** 关闭到指定 peer 的 WebSocket 连接

**输入参数：**
```json
{
  "properties": {
    "peer_id": {
      "description": "要断开的 Agent ID",
      "type": "string"
    }
  },
  "required": [
    "peer_id"
  ],
  "type": "object"
}
```

---

### `lan_create_channel`

**描述：** 创建通信频道并邀请 peer 加入

**输入参数：**
```json
{
  "properties": {
    "channel_name": {
      "description": "频道名称",
      "type": "string"
    },
    "peer_ids": {
      "description": "要邀请的 Agent ID 列表",
      "items": {
        "type": "string"
      },
      "type": "array"
    }
  },
  "required": [
    "channel_name",
    "peer_ids"
  ],
  "type": "object"
}
```

---

### `lan_leave_channel`

**描述：** 退出通信频道

**输入参数：**
```json
{
  "properties": {
    "channel_id": {
      "description": "频道 ID",
      "type": "string"
    }
  },
  "required": [
    "channel_id"
  ],
  "type": "object"
}
```

---

### `lan_send_message`

**描述：** 向频道发送消息。频道内所有成员的连接必须已打开。

**输入参数：**
```json
{
  "properties": {
    "channel_id": {
      "description": "频道 ID",
      "type": "string"
    },
    "message_body": {
      "description": "消息内容",
      "type": "string"
    }
  },
  "required": [
    "channel_id",
    "message_body"
  ],
  "type": "object"
}
```

---

### `lan_share_file`

**描述：** 分享本地文件到频道。支持大文件（自动分块并发传输）。

**输入参数：**
```json
{
  "properties": {
    "channel_id": {
      "description": "频道 ID",
      "type": "string"
    },
    "file_path": {
      "description": "本地文件路径",
      "type": "string"
    }
  },
  "required": [
    "channel_id",
    "file_path"
  ],
  "type": "object"
}
```

---

### `lan_share_folder`

**描述：** 分享整个文件夹到频道。自动检测变化，只传输修改过的文件。

**输入参数：**
```json
{
  "properties": {
    "channel_id": {
      "description": "频道 ID",
      "type": "string"
    },
    "folder_path": {
      "description": "本地文件夹路径",
      "type": "string"
    }
  },
  "required": [
    "channel_id",
    "folder_path"
  ],
  "type": "object"
}
```

---

### `lan_sync_folder`

**描述：** 与远程 peer 同步本地文件夹。扫描变化，只发送 diff。

**输入参数：**
```json
{
  "properties": {
    "folder_path": {
      "description": "本地文件夹路径",
      "type": "string"
    },
    "peer_id": {
      "description": "远程 Agent ID",
      "type": "string"
    }
  },
  "required": [
    "folder_path",
    "peer_id"
  ],
  "type": "object"
}
```

---

### `lan_get_transfer_status`

**描述：** 查询文件传输状态

**输入参数：**
```json
{
  "properties": {
    "transfer_id": {
      "description": "传输 ID",
      "type": "string"
    }
  },
  "required": [
    "transfer_id"
  ],
  "type": "object"
}
```

---

### `lan_list_transfers`

**描述：** 列出所有活跃的文件传输

**输入参数：**
```json
{
  "properties": {},
  "type": "object"
}
```

---

