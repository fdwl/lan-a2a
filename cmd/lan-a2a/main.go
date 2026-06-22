package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fdwl/lan-a2a/internal/channel"
	"github.com/fdwl/lan-a2a/internal/cli"
	"github.com/fdwl/lan-a2a/internal/fileservice"
	"github.com/fdwl/lan-a2a/internal/filetransfer"
	"github.com/fdwl/lan-a2a/internal/logger"
	"github.com/fdwl/lan-a2a/internal/mcp"
	"github.com/fdwl/lan-a2a/internal/p2p"
	"github.com/fdwl/lan-a2a/internal/profile"
	"github.com/fdwl/lan-a2a/internal/protocol"
	"github.com/fdwl/lan-a2a/internal/relay"
)

type agent struct {
	id      string
	p2p     *p2p.P2P
	relay   *relay.Client
	chMgr   *channel.Manager
	fileMgr *filetransfer.Manager
	fileSvc *fileservice.FileService
	folderSync *fileservice.FolderSync

	conns   map[string]*protocol.Conn
	connsMu sync.Mutex

	lanOnline   map[string]bool
	relayOnline map[string]bool
	onlineMu    sync.RWMutex
}

func (a *agent) markLANOnline(peerID string) {
	a.onlineMu.Lock()
	a.lanOnline[peerID] = true
	a.onlineMu.Unlock()
}

func (a *agent) markRelayOnline(ids []string) {
	a.onlineMu.Lock()
	a.relayOnline = make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != a.id {
			a.relayOnline[id] = true
		}
	}
	a.onlineMu.Unlock()
}

func (a *agent) GetOnlinePeers() []string {
	a.onlineMu.RLock()
	defer a.onlineMu.RUnlock()
	seen := make(map[string]bool)
	var result []string
	for id := range a.lanOnline {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	for id := range a.relayOnline {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func (a *agent) OpenConnection(peerID string) error {
	a.onlineMu.RLock()
	isLAN := a.lanOnline[peerID]
	isRelay := a.relayOnline[peerID]
	a.onlineMu.RUnlock()

	if !isLAN && !isRelay {
		return fmt.Errorf("peer %s is not online", peerID)
	}

	a.connsMu.Lock()
	if _, ok := a.conns[peerID]; ok {
		a.connsMu.Unlock()
		return nil // already connected
	}
	a.connsMu.Unlock()

	if isLAN {
		// Direct TCP connection to LAN peer
		// Need to know the address — stored in p2p online map
		conn, err := a.p2p.OpenConn(peerID)
		if err != nil {
			return err
		}
		a.connsMu.Lock()
		a.conns[peerID] = conn
		a.connsMu.Unlock()
		logger.Info("connection opened", "peer", peerID, "type", "p2p")
		return nil
	}

	if isRelay {
		// Can't directly connect to relay peer — mark as "relay-connected"
		// Messages will be forwarded through the relay
		a.connsMu.Lock()
		a.conns[peerID] = nil // nil = relay-forwarded
		a.connsMu.Unlock()
		logger.Info("connection opened", "peer", peerID, "type", "relay")
		return nil
	}

	return fmt.Errorf("peer %s is not online", peerID)
}

func (a *agent) CloseConnection(peerID string) error {
	a.connsMu.Lock()
	conn, ok := a.conns[peerID]
	if ok {
		delete(a.conns, peerID)
	}
	a.connsMu.Unlock()
	if ok {
		conn.Close()
		logger.Info("connection closed", "peer", peerID)
	}
	return nil
}

func (a *agent) getConn(peerID string) (*protocol.Conn, error) {
	a.connsMu.Lock()
	conn, ok := a.conns[peerID]
	a.connsMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no connection to %s, call lan_open_connection first", peerID)
	}
	return conn, nil
}

func (a *agent) CreateChannel(name string, peerIDs []string) (string, []string, error) {
	for _, pid := range peerIDs {
		if !a.p2p.IsOnline(pid) {
			return "", nil, fmt.Errorf("peer %s not online", pid)
		}
		if _, err := a.getConn(pid); err != nil {
			return "", nil, fmt.Errorf("peer %s: %w", pid, err)
		}
	}
	chID := fmt.Sprintf("ch-%d-%s", time.Now().UnixNano(), a.id)
	members := append([]string{a.id}, peerIDs...)

	mode := channel.ModeP2P
	if a.relay != nil {
		mode = channel.ModeRelay
	}
	a.chMgr.Create(chID, name, a.id, members, mode)
	if mode == channel.ModeP2P {
		logger.Info("channel created as P2P lobby", "channel", chID[:12], "host", a.id)
	}

	msg := protocol.Message{
		Type: protocol.MsgTypeText, ID: protocol.NewMsgID(), From: a.id,
		ChannelID: chID, Content: fmt.Sprintf(`{"event":"channel_created","channel_id":"%s","channel_name":"%s","mode":"%s","host":"%s","members":%s}`, chID, name, mode, a.id, jsonArr(members)),
		Timestamp: time.Now().Unix(),
	}

	if mode == channel.ModeRelay {
		a.relay.Send(msg)
	} else {
		a.broadcastToMembers(peerIDs, msg)
	}
	return chID, members, nil
}

func (a *agent) LeaveChannel(chID string) error {
	ch, ok := a.chMgr.Get(chID)
	if !ok {
		return fmt.Errorf("channel %s not found", chID)
	}

	msg := protocol.Message{
		Type: protocol.MsgTypeText, ID: protocol.NewMsgID(), From: a.id,
		ChannelID: chID, Content: fmt.Sprintf(`{"event":"peer_left","channel_id":"%s","peer_id":"%s"}`, chID, a.id),
		Timestamp: time.Now().Unix(),
	}

	if ch.Mode == channel.ModeRelay {
		a.chMgr.Leave(chID, a.id)
		if a.relay != nil {
			a.relay.Send(msg)
		}
		return nil
	}

	var others []string
	for _, m := range ch.Members {
		if m != a.id {
			others = append(others, m)
		}
	}
	a.chMgr.Leave(chID, a.id)
	a.broadcastToMembers(others, msg)
	return nil
}

func (a *agent) SendMessage(chID, body string) error {
	ch, ok := a.chMgr.Get(chID)
	if !ok {
		return fmt.Errorf("channel %s not found", chID)
	}

	msg := protocol.Message{
		Type: protocol.MsgTypeText, ID: protocol.NewMsgID(), From: a.id,
		ChannelID: chID, Content: body, Timestamp: time.Now().Unix(),
	}

	if ch.Mode == channel.ModeRelay {
		if a.relay == nil {
			return fmt.Errorf("no relay connection")
		}
		return a.relay.Send(msg)
	}

	var others []string
	for _, m := range ch.Members {
		if m != a.id {
			others = append(others, m)
		}
	}
	if len(others) == 0 {
		return fmt.Errorf("no other members in channel")
	}
	a.broadcastToMembers(others, msg)
	return nil
}

func (a *agent) broadcastToMembers(ids []string, msg protocol.Message) {
	for _, id := range ids {
		conn, err := a.getConn(id)
		if err != nil {
			logger.Info("skipping send", "peer", id, "error", err)
			continue
		}
		conn.Send(msg)
	}
}

func (a *agent) ShareFile(chID, filePath string) error {
	ch, ok := a.chMgr.Get(chID)
	if !ok {
		return fmt.Errorf("channel %s not found", chID)
	}

	var others []string
	for _, m := range ch.Members {
		if m != a.id {
			others = append(others, m)
		}
	}

	if ch.Mode == channel.ModeRelay {
		if a.relay == nil {
			return fmt.Errorf("no relay connection")
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return err
		}
		filename := filepath.Base(absPath)
		chunks, checksum, fileSize, err := filetransfer.SplitFile(absPath)
		if err != nil {
			return err
		}
		msgID := protocol.NewMsgID()
		metaMsg := protocol.Message{
			Type: protocol.MsgTypeFileMeta, ID: msgID, From: a.id, ChannelID: chID,
			Filename: filename, FileSize: fileSize, Checksum: checksum,
			TotalChunks: len(chunks), Timestamp: time.Now().Unix(),
		}
		a.relay.Send(metaMsg)
		for i, chunk := range chunks {
			a.relay.Send(protocol.Message{
				Type: protocol.MsgTypeFileData, ID: protocol.NewMsgID(), From: a.id,
				ChannelID: chID, ChunkIdx: i, Data: chunk, Timestamp: time.Now().Unix(),
			})
		}
		a.relay.Send(protocol.Message{
			Type: protocol.MsgTypeFileDone, ID: msgID, From: a.id, ChannelID: chID,
			Filename: filename, Checksum: checksum, Timestamp: time.Now().Unix(),
		})
		return nil
	}

	for _, peerID := range others {
		_, err := a.fileSvc.SendFile(peerID, filePath)
		if err != nil {
			logger.Error("file send failed", "peer", peerID, "error", err)
		}
	}
	return nil
}

func (a *agent) ShareFolder(chID, folderPath string) error {
	ch, ok := a.chMgr.Get(chID)
	if !ok {
		return fmt.Errorf("channel %s not found", chID)
	}

	var others []string
	for _, m := range ch.Members {
		if m != a.id {
			others = append(others, m)
		}
	}

	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return err
	}

	for _, peerID := range others {
		result, err := a.folderSync.SyncFolder(absPath, peerID)
		if err != nil {
			logger.Error("folder sync failed", "peer", peerID, "error", err)
			continue
		}
		logger.Info("folder sync summary", "peer", peerID, "files", len(result.Diff.Adds)+len(result.Diff.Modifies))
	}
	return nil
}

func (a *agent) SyncFolder(folderPath, peerID string) error {
	if !a.p2p.IsOnline(peerID) {
		return fmt.Errorf("peer %s not online", peerID)
	}
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return err
	}
	result, err := a.folderSync.SyncFolder(absPath, peerID)
	if err != nil {
		return err
	}
	logger.Info("folder sync details", "peer", peerID, "adds", len(result.Diff.Adds), "modifies", len(result.Diff.Modifies), "deletes", len(result.Diff.Deletes))
	return nil
}

func (a *agent) GetTransferStatus(id string) (interface{}, error) {
	t, ok := a.fileSvc.GetTransfer(id)
	if !ok {
		return nil, fmt.Errorf("transfer %s not found", id)
	}
	return t, nil
}

func (a *agent) ListTransfers() ([]interface{}, error) {
	transfers := a.fileSvc.ListTransfers()
	result := make([]interface{}, len(transfers))
	for i, t := range transfers {
		result[i] = t
	}
	return result, nil
}

func (a *agent) GetAgentInfo(peerID string) (interface{}, error) {
	if !a.p2p.IsOnline(peerID) {
		return nil, fmt.Errorf("peer %s is not online", peerID)
	}
	profile := a.p2p.GetPeerProfile(peerID)
	if profile == nil {
		return map[string]interface{}{
			"id":      peerID,
			"status":  "online",
			"source":  "lan",
		}, nil
	}
	source := "lan"
	if a.relayOnline[peerID] {
		source = "relay"
	}
	return map[string]interface{}{
		"id":       profile.ID,
		"name":     profile.Name,
		"avatar":   profile.Avatar,
		"roles":    profile.Roles,
		"tags":     profile.Tags,
		"metadata": profile.Metadata,
		"status":   "online",
		"source":   source,
	}, nil
}

func (a *agent) ListChannels() ([]interface{}, error) {
	channels := a.chMgr.ListByPeer(a.id)
	result := make([]interface{}, len(channels))
	for i, ch := range channels {
		result[i] = map[string]interface{}{
			"channel_id": ch.ID,
			"name":       ch.Name,
			"mode":       ch.Mode,
			"host":       ch.Host,
			"members":    ch.Members,
			"created_at": ch.CreatedAt,
			"creator":    ch.Creator,
		}
	}
	return result, nil
}

func (a *agent) SetProfile(name, description string, skills []string) error {
	prof := &protocol.ProfilePayload{
		ID:   a.id,
		Name: name,
		Roles: skills,
		Metadata: map[string]string{
			"description": description,
		},
	}
	a.p2p.SetProfile(prof)
	logger.Info("profile updated", "name", name, "description", description, "skills", skills)
	return nil
}

func main() {
	id := flag.String("id", "", "Agent ID (auto-generated if empty)")
	port := flag.Int("port", 0, "TCP port (auto if 0)")
	profileDir := flag.String("profile-dir", "", "Profile directory (default: cwd, then ~/.lan-a2a)")
	profilePath := flag.String("profile", "", "Exact profile file path")
	flag.Parse()

	logger.Init("info", nil)

	// Load or create profile
	prof := loadProfile(*id, *profileDir, *profilePath)
	if *id == "" {
		*id = prof.ID
	}
	if *port == 0 {
		*port = 19100 + os.Getpid()%1000
	}

	a := &agent{
		id:          *id,
		chMgr:       channel.NewManager(),
		fileMgr:     filetransfer.NewManager(*id),
		conns:       make(map[string]*protocol.Conn),
		lanOnline:   make(map[string]bool),
		relayOnline: make(map[string]bool),
	}

	// File service — concurrent chunked transfer with plugin support
	a.fileSvc = fileservice.NewFileService(func(peerID string, chunkData []byte, t *fileservice.Transfer, c *fileservice.Chunk) error {
		conn, err := a.getConn(peerID)
		if err != nil {
			return err
		}
		return conn.Send(protocol.Message{
			Type:        protocol.MsgTypeFileData,
			ID:          protocol.NewMsgID(),
			From:        a.id,
			ChannelID:   t.ID,
			ChunkIdx:    c.Index,
			Data:        chunkData,
			Timestamp:   time.Now().Unix(),
		})
	}, 4)
	a.folderSync = fileservice.NewFolderSync(a.fileSvc)

	// P2P — no auto connections, mDNS only tracks online status
	a.p2p = p2p.New(*id, *port)
	a.p2p.OnMessage = func(msg protocol.Message, from string) {
		switch msg.Type {
		case protocol.MsgTypeText:
			if msg.ChannelID != "" && a.chMgr.IsMember(msg.ChannelID, from) {
				logger.Info("message received", "channel", msg.ChannelID[:12], "from", from[:8], "content", truncate(msg.Content, 80))
			}
		case protocol.MsgTypeFileMeta:
			logger.Info("file meta received", "from", from[:8], "filename", msg.Filename, "size", msg.FileSize)
			a.fileMgr.PrepareIncoming(msg.ChannelID, msg.ID, from, msg.Filename, msg.FileSize, msg.TotalChunks)
		case protocol.MsgTypeFileDone:
			logger.Info("file transfer complete", "from", from[:8], "filename", msg.Filename)
		}
	}
	a.p2p.OnFileData = func(msg protocol.Message, reader io.Reader, from string) {
		data, _ := io.ReadAll(reader)
		a.fileMgr.AddChunk(msg.ID, msg.ChunkIdx, data)
	}
	a.fileMgr.OnComplete = func(f *filetransfer.IncomingFile) {
		logger.Info("file saved", "filename", f.Filename, "path", f.LocalPath)
	}

	if err := a.p2p.Start(); err != nil {
		logger.Error("p2p start failed", "error", err)
		os.Exit(1)
	}

	// mDNS discovery — only tracks who's online, zero connections
	relayConnected := false
	stopDisc, err := p2p.StartDiscovery(*id, *port,
		func(peerID, addr string, peerPort int) {
			a.markLANOnline(peerID)
			a.p2p.MarkOnline(peerID, fmt.Sprintf("%s:%d", addr, peerPort))
			logger.Info("peer online", "peer", peerID, "type", "lan")
		},
		func(relayAddr string) {
			if relayConnected || a.relay != nil {
				return
			}
			a.relay = relay.NewClient(*id, relayAddr)
			a.relay.OnMessage = a.p2p.OnMessage
			a.relay.OnFileData = a.p2p.OnFileData
			a.relay.OnOnlineList = func(ids []string) {
				a.markRelayOnline(ids)
			}
			if err := a.relay.Connect(); err != nil {
				logger.Error("relay connect failed", "error", err)
				a.relay = nil
				return
			}
			relayConnected = true
			logger.Info("relay connected", "address", relayAddr)
		},
	)
	if err != nil {
		logger.Warn("mDNS discovery unavailable", "error", err)
	}

	logger.Info("agent ready", "id", *id, "port", *port)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// MCP stdio
	srv := mcp.NewServer(a)

	// Detect interactive mode: stdin is a terminal
	interactive := isTerminal()

	if interactive {
		// Interactive CLI mode
		c := cli.New(a, *id, profString(prof))
		go func() {
			<-c.Quit()
			sigCh <- syscall.SIGTERM
		}()
		go func() {
			fmt.Println("\033[32mReady. Type 'help' for commands.\033[0m")
			c.Run()
		}()
	} else {
		// SDK/MCP mode
		go srv.Run()
	}

	home, _ := os.UserHomeDir()
	os.MkdirAll(filepath.Join(home, filetransfer.DownloadBaseDir), 0755)

	<-sigCh
	logger.Info("shutting down")

	// Send goodbye to all connected peers
	a.connsMu.Lock()
	for peerID, conn := range a.conns {
		if conn != nil {
			conn.Send(protocol.Message{
				Type: protocol.MsgTypeGoodbye, From: a.id, ID: protocol.NewMsgID(),
			})
		}
		logger.Info("goodbye sent", "peer", peerID)
		delete(a.conns, peerID)
	}
	a.connsMu.Unlock()

	// Send goodbye through relay
	if a.relay != nil {
		a.relay.SendGoodbye()
		a.relay.Stop()
	}

	if stopDisc != nil {
		stopDisc()
	}
	a.p2p.Stop()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func jsonArr(items []string) string {
	b, _ := json.Marshal(items)
	return string(b)
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func loadProfile(id, profileDir, profilePath string) *profile.Profile {
	// 1. Exact path specified
	if profilePath != "" {
		p, err := profile.Load(filepath.Dir(profilePath))
		if err == nil {
			if id != "" {
				p.ID = id
			}
			logger.Info("profile loaded", "path", profilePath, "name", p.Name)
			return p
		}
		logger.Warn("profile not found at specified path", "path", profilePath, "error", err)
	}

	// 2. Directory specified
	if profileDir != "" {
		p, err := profile.Load(profileDir)
		if err == nil {
			if id != "" {
				p.ID = id
			}
			logger.Info("profile loaded", "dir", profileDir, "name", p.Name)
			return p
		}
		logger.Warn("profile not found in specified directory", "dir", profileDir, "error", err)
	}

	// 3. Current working directory
	if p, err := profile.Load("."); err == nil {
		if id != "" {
			p.ID = id
		}
		logger.Info("profile loaded from cwd", "name", p.Name)
		return p
	}

	// 4. Home directory (~/.lan-a2a/)
	home, _ := os.UserHomeDir()
	if home != "" {
		homeDir := filepath.Join(home, ".lan-a2a")
		os.MkdirAll(homeDir, 0755)
		if p, err := profile.Load(homeDir); err == nil {
			if id != "" {
				p.ID = id
			}
			logger.Info("profile loaded from home", "name", p.Name)
			return p
		}
		// Create default profile in home dir
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "agent"
		}
		agentID := id
		if agentID == "" {
			agentID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
		}
		p := &profile.Profile{
			ID:   agentID,
			Name: hostname,
			Roles: []string{"general"},
			Metadata: map[string]string{
				"description": "LanA2A agent",
			},
		}
		p.SetTimestamps()
		profile.Save(p, homeDir)
		logger.Info("default profile created", "path", homeDir, "id", agentID)
		return p
	}

	// 5. Fallback: create in-memory profile
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "agent"
	}
	agentID := id
	if agentID == "" {
		agentID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}
	return &profile.Profile{
		ID:   agentID,
		Name: hostname,
		Roles: []string{"general"},
	}
}

func profString(p *profile.Profile) string {
	return fmt.Sprintf("%s (%s)", p.Name, p.ID)
}
