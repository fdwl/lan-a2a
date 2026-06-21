package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ToolDoc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func main() {
	tools := []ToolDoc{
		{Name: "lan_get_online_agents", Description: "获取局域网内所有在线的 AI 节点",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{Name: "lan_open_connection", Description: "打开到指定 peer 的 WebSocket 连接。发送消息或文件前必须先调用此工具。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"peer_id": map[string]interface{}{"type": "string", "description": "要连接的 Agent ID"},
			}, "required": []string{"peer_id"}}},
		{Name: "lan_close_connection", Description: "关闭到指定 peer 的 WebSocket 连接",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"peer_id": map[string]interface{}{"type": "string", "description": "要断开的 Agent ID"},
			}, "required": []string{"peer_id"}}},
		{Name: "lan_create_channel", Description: "创建通信频道并邀请 peer 加入",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_name": map[string]interface{}{"type": "string", "description": "频道名称"},
				"peer_ids":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "要邀请的 Agent ID 列表"},
			}, "required": []string{"channel_name", "peer_ids"}}},
		{Name: "lan_leave_channel", Description: "退出通信频道",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id": map[string]interface{}{"type": "string", "description": "频道 ID"},
			}, "required": []string{"channel_id"}}},
		{Name: "lan_send_message", Description: "向频道发送消息。频道内所有成员的连接必须已打开。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id":   map[string]interface{}{"type": "string", "description": "频道 ID"},
				"message_body": map[string]interface{}{"type": "string", "description": "消息内容"},
			}, "required": []string{"channel_id", "message_body"}}},
		{Name: "lan_share_file", Description: "分享本地文件到频道。支持大文件（自动分块并发传输）。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id": map[string]interface{}{"type": "string", "description": "频道 ID"},
				"file_path":  map[string]interface{}{"type": "string", "description": "本地文件路径"},
			}, "required": []string{"channel_id", "file_path"}}},
		{Name: "lan_share_folder", Description: "分享整个文件夹到频道。自动检测变化，只传输修改过的文件。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id":  map[string]interface{}{"type": "string", "description": "频道 ID"},
				"folder_path": map[string]interface{}{"type": "string", "description": "本地文件夹路径"},
			}, "required": []string{"channel_id", "folder_path"}}},
		{Name: "lan_sync_folder", Description: "与远程 peer 同步本地文件夹。扫描变化，只发送 diff。",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"folder_path": map[string]interface{}{"type": "string", "description": "本地文件夹路径"},
				"peer_id":     map[string]interface{}{"type": "string", "description": "远程 Agent ID"},
			}, "required": []string{"folder_path", "peer_id"}}},
		{Name: "lan_get_transfer_status", Description: "查询文件传输状态",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"transfer_id": map[string]interface{}{"type": "string", "description": "传输 ID"},
			}, "required": []string{"transfer_id"}}},
		{Name: "lan_list_transfers", Description: "列出所有活跃的文件传输",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
	}

	var sb strings.Builder
	sb.WriteString("# LAN Agent Bus - MCP 工具接口文档\n\n")
	sb.WriteString("## 工具列表\n\n")
	sb.WriteString("| 工具名 | 描述 |\n")
	sb.WriteString("|--------|------|\n")
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("| `%s` | %s |\n", t.Name, t.Description))
	}
	sb.WriteString("\n## 详细接口\n\n")
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("### `%s`\n\n", t.Name))
		sb.WriteString(fmt.Sprintf("**描述：** %s\n\n", t.Description))
		schema, _ := json.MarshalIndent(t.InputSchema, "", "  ")
		sb.WriteString(fmt.Sprintf("**输入参数：**\n```json\n%s\n```\n\n", schema))
		sb.WriteString("---\n\n")
	}

	os.WriteFile("docs/mcp-tools.md", []byte(sb.String()), 0644)

	jsonData, _ := json.MarshalIndent(tools, "", "  ")
	os.WriteFile("docs/mcp-tools.json", jsonData, 0644)
	fmt.Println("API 文档已生成:")
	fmt.Println("  docs/mcp-tools.md")
	fmt.Println("  docs/mcp-tools.json")
}
