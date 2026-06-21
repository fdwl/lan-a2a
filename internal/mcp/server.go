package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type AgentAPI interface {
	GetOnlinePeers() []string
	OpenConnection(peerID string) error
	CloseConnection(peerID string) error
	CreateChannel(name string, peerIDs []string) (string, []string, error)
	LeaveChannel(channelID string) error
	SendMessage(channelID, body string) error
	ShareFile(channelID, filePath string) error
	ShareFolder(channelID, folderPath string) error
	SyncFolder(folderPath, peerID string) error
	GetTransferStatus(transferID string) (interface{}, error)
	ListTransfers() ([]interface{}, error)
}

type Server struct {
	api    AgentAPI
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewServer(api AgentAPI) *Server {
	return &Server{
		api:    api,
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

func (s *Server) Run() {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      interface{}     `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		s.handle(req)
	}
}

func (s *Server) handle(req struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}) {
	switch req.Method {
	case "initialize":
		s.reply(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":   map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":     map[string]interface{}{"name": "lan-agent-bus", "version": "0.1.0"},
		})
	case "notifications/initialized":
	case "tools/list":
		s.reply(req.ID, map[string]interface{}{"tools": s.tools()})
	case "tools/call":
		s.handleToolCall(req.ID, req.Params)
	case "ping":
		s.reply(req.ID, map[string]interface{}{})
	default:
		s.replyError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (s *Server) tools() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "lan_get_online_agents", "description": "Get all online AI agents in the local network",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "lan_open_connection", "description": "Open a TCP connection to a peer agent. Must be called before sending messages or files.",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"peer_id": map[string]interface{}{"type": "string", "description": "Agent ID to connect to"},
			}, "required": []string{"peer_id"}}},
		{"name": "lan_close_connection", "description": "Close the TCP connection to a peer agent",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"peer_id": map[string]interface{}{"type": "string", "description": "Agent ID to disconnect from"},
			}, "required": []string{"peer_id"}}},
		{"name": "lan_create_channel", "description": "Create a communication channel and invite peer agents",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_name": map[string]interface{}{"type": "string"},
				"peer_ids":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			}, "required": []string{"channel_name", "peer_ids"}}},
		{"name": "lan_leave_channel", "description": "Leave a communication channel",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id": map[string]interface{}{"type": "string"},
			}, "required": []string{"channel_id"}}},
		{"name": "lan_send_message", "description": "Send a message to a channel. Connections to all channel members must be open first.",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id":   map[string]interface{}{"type": "string"},
				"message_body": map[string]interface{}{"type": "string"},
			}, "required": []string{"channel_id", "message_body"}}},
		{"name": "lan_share_file", "description": "Share a local file to a channel. Supports large files (auto-chunked). Connections to all channel members must be open first.",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id": map[string]interface{}{"type": "string"},
				"file_path":  map[string]interface{}{"type": "string"},
			}, "required": []string{"channel_id", "file_path"}}},
		{"name": "lan_share_folder", "description": "Share an entire folder to a peer. Auto-detects changes and transfers only modified files. Connections must be open first.",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"channel_id":  map[string]interface{}{"type": "string"},
				"folder_path": map[string]interface{}{"type": "string"},
			}, "required": []string{"channel_id", "folder_path"}}},
		{"name": "lan_sync_folder", "description": "Sync a local folder with a remote peer. Scans for changes, sends only diffs. Connections must be open first.",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"folder_path": map[string]interface{}{"type": "string"},
				"peer_id":     map[string]interface{}{"type": "string"},
			}, "required": []string{"folder_path", "peer_id"}}},
		{"name": "lan_get_transfer_status", "description": "Get status of a file transfer",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"transfer_id": map[string]interface{}{"type": "string"},
			}, "required": []string{"transfer_id"}}},
		{"name": "lan_list_transfers", "description": "List all active file transfers",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
	}
}

func (s *Server) handleToolCall(id interface{}, params json.RawMessage) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.replyError(id, -32602, "Invalid params")
		return
	}

	var result interface{}
	var toolErr string

	switch p.Name {
	case "lan_get_online_agents":
		peers := s.api.GetOnlinePeers()
		result = map[string]interface{}{"agents": peers, "count": len(peers)}

	case "lan_open_connection":
		var a struct{ PeerID string `json:"peer_id"` }
		if json.Unmarshal(p.Arguments, &a) != nil || a.PeerID == "" {
			toolErr = "Invalid arguments: peer_id required"
			break
		}
		if err := s.api.OpenConnection(a.PeerID); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "connected", "peer_id": a.PeerID}

	case "lan_close_connection":
		var a struct{ PeerID string `json:"peer_id"` }
		if json.Unmarshal(p.Arguments, &a) != nil || a.PeerID == "" {
			toolErr = "Invalid arguments: peer_id required"
			break
		}
		if err := s.api.CloseConnection(a.PeerID); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "disconnected", "peer_id": a.PeerID}

	case "lan_create_channel":
		var a struct {
			ChannelName string   `json:"channel_name"`
			PeerIDs     []string `json:"peer_ids"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil || len(a.PeerIDs) == 0 {
			toolErr = "Invalid arguments"
			break
		}
		chID, members, err := s.api.CreateChannel(a.ChannelName, a.PeerIDs)
		if err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"channel_id": chID, "members": members}

	case "lan_leave_channel":
		var a struct{ ChannelID string `json:"channel_id"` }
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		if err := s.api.LeaveChannel(a.ChannelID); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "left", "channel_id": a.ChannelID}

	case "lan_send_message":
		var a struct {
			ChannelID   string `json:"channel_id"`
			MessageBody string `json:"message_body"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		if err := s.api.SendMessage(a.ChannelID, a.MessageBody); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "sent", "channel_id": a.ChannelID}

	case "lan_share_file":
		var a struct {
			ChannelID string `json:"channel_id"`
			FilePath  string `json:"file_path"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		if err := s.api.ShareFile(a.ChannelID, a.FilePath); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "shared", "channel_id": a.ChannelID}

	case "lan_share_folder":
		var a struct {
			ChannelID  string `json:"channel_id"`
			FolderPath string `json:"folder_path"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		if err := s.api.ShareFolder(a.ChannelID, a.FolderPath); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "sharing", "channel_id": a.ChannelID, "folder": a.FolderPath}

	case "lan_sync_folder":
		var a struct {
			FolderPath string `json:"folder_path"`
			PeerID     string `json:"peer_id"`
		}
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		if err := s.api.SyncFolder(a.FolderPath, a.PeerID); err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"status": "syncing", "folder": a.FolderPath, "peer_id": a.PeerID}

	case "lan_get_transfer_status":
		var a struct{ TransferID string `json:"transfer_id"` }
		if json.Unmarshal(p.Arguments, &a) != nil {
			toolErr = "Invalid arguments"
			break
		}
		status, err := s.api.GetTransferStatus(a.TransferID)
		if err != nil {
			toolErr = err.Error()
			break
		}
		result = status

	case "lan_list_transfers":
		transfers, err := s.api.ListTransfers()
		if err != nil {
			toolErr = err.Error()
			break
		}
		result = map[string]interface{}{"transfers": transfers, "count": len(transfers)}

	default:
		s.replyError(id, -32601, "Unknown tool: "+p.Name)
		return
	}

	if toolErr != "" {
		s.reply(id, map[string]interface{}{
			"isError": true,
			"content": []map[string]interface{}{{"type": "text", "text": toolErr}},
		})
		return
	}

	text := fmt.Sprintf("%v", result)
	s.reply(id, map[string]interface{}{
		"content": []map[string]interface{}{{"type": "text", "text": text}},
	})
}

func (s *Server) reply(id interface{}, result interface{}) {
	data, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": id, "result": result})
	s.writer.Write(data)
	s.writer.WriteByte('\n')
	s.writer.Flush()
}

func (s *Server) replyError(id interface{}, code int, msg string) {
	data, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]interface{}{"code": code, "message": msg},
	})
	s.writer.Write(data)
	s.writer.WriteByte('\n')
	s.writer.Flush()
}
