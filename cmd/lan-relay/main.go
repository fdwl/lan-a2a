package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fdwl/lan-a2a/internal/logger"
	"github.com/fdwl/lan-a2a/internal/p2p"
	"github.com/fdwl/lan-a2a/internal/protocol"
)

type AgentConn struct {
	ID       string
	Conn     *protocol.Conn
	mu       sync.Mutex
	done     chan struct{}
	channels map[string]bool // channels this agent has joined
}

type Server struct {
	agents map[string]*AgentConn
	mu     sync.RWMutex
	done   chan struct{}

	// Channel index: channelID → set of agentID
	channelIndex   map[string]map[string]bool
	channelIndexMu sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		agents:       make(map[string]*AgentConn),
		done:         make(chan struct{}),
		channelIndex: make(map[string]map[string]bool),
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := protocol.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("websocket upgrade error", "error", err)
		return
	}
	conn := protocol.NewConn(ws)

	msg, err := conn.Read()
	if err != nil || msg.Type != protocol.MsgTypeRegister {
		conn.Close()
		return
	}
	agentID := msg.From
	if err := conn.Send(protocol.Message{Type: protocol.MsgTypeRegisterOK, From: "relay", ID: protocol.NewMsgID()}); err != nil {
		conn.Close()
		return
	}

	ac := &AgentConn{
		ID:       agentID,
		Conn:     conn,
		done:     make(chan struct{}),
		channels: make(map[string]bool),
	}
	s.mu.Lock()
	if old, ok := s.agents[agentID]; ok {
		close(old.done)
	}
	s.agents[agentID] = ac
	s.mu.Unlock()
	logger.Info("agent registered", "agent", agentID, "total", s.AgentCount())

	defer func() {
		close(ac.done)
		s.mu.Lock()
		if cur, ok := s.agents[agentID]; ok && cur == ac {
			delete(s.agents, agentID)
		}
		s.mu.Unlock()
		// Clean up channel index
		s.channelIndexMu.Lock()
		for chID := range ac.channels {
			if members, ok := s.channelIndex[chID]; ok {
				delete(members, agentID)
				if len(members) == 0 {
					delete(s.channelIndex, chID)
				}
			}
		}
		s.channelIndexMu.Unlock()
		conn.Close()
		logger.Info("agent disconnected", "agent", agentID, "total", s.AgentCount())
	}()

	s.handleChannelEvents(ac)

	for {
		msg, err := conn.Read()
		if err != nil {
			return
		}
		switch msg.Type {
		case protocol.MsgTypePing:
			if err := conn.Send(protocol.Message{Type: protocol.MsgTypePong, From: "relay", ID: protocol.NewMsgID()}); err != nil {
				return
			}
		case protocol.MsgTypeQueryOnline:
			s.handleQueryOnline(conn, msg)
		case protocol.MsgTypeText:
			s.handleChannelJoinLeave(msg, ac)
			s.relayToChannel(msg, agentID)
		case protocol.MsgTypeFileMeta, protocol.MsgTypeFileData, protocol.MsgTypeFileDone:
			s.relayToChannel(msg, agentID)
		}
	}
}

func (s *Server) handleChannelEvents(_ *AgentConn) {}

func (s *Server) handleChannelJoinLeave(msg protocol.Message, ac *AgentConn) {
	if msg.ChannelID == "" || msg.Content == "" {
		return
	}

	var event struct {
		Event     string   `json:"event"`
		ChannelID string   `json:"channel_id"`
		Members   []string `json:"members"`
	}
	if json.Unmarshal([]byte(msg.Content), &event) != nil {
		return
	}

	switch event.Event {
	case "channel_created":
		s.channelIndexMu.Lock()
		if s.channelIndex[event.ChannelID] == nil {
			s.channelIndex[event.ChannelID] = make(map[string]bool)
		}
		for _, memberID := range event.Members {
			s.channelIndex[event.ChannelID][memberID] = true
			s.mu.RLock()
			if member, ok := s.agents[memberID]; ok {
				member.mu.Lock()
				member.channels[event.ChannelID] = true
				member.mu.Unlock()
			}
			s.mu.RUnlock()
		}
		s.channelIndexMu.Unlock()
		logger.Info("channel created", "channel", event.ChannelID[:12], "members", len(event.Members))

	case "peer_left":
		s.channelIndexMu.Lock()
		if members, ok := s.channelIndex[event.ChannelID]; ok {
			delete(members, ac.ID)
			if len(members) == 0 {
				delete(s.channelIndex, event.ChannelID)
			}
		}
		s.channelIndexMu.Unlock()
		ac.mu.Lock()
		delete(ac.channels, event.ChannelID)
		ac.mu.Unlock()
	}
}

// relayToChannel forwards messages only to channel members, not all agents.
func (s *Server) relayToChannel(msg protocol.Message, fromAgent string) {
	if msg.ChannelID == "" {
		return
	}

	s.channelIndexMu.RLock()
	members, ok := s.channelIndex[msg.ChannelID]
	if !ok {
		s.channelIndexMu.RUnlock()
		return
	}
	targets := make([]string, 0, len(members))
	for id := range members {
		if id != fromAgent {
			targets = append(targets, id)
		}
	}
	s.channelIndexMu.RUnlock()

	s.mu.RLock()
	for _, id := range targets {
		if ac, ok := s.agents[id]; ok {
			ac.mu.Lock()
			go func(ac *AgentConn) {
				defer ac.mu.Unlock()
				if err := ac.Conn.Send(msg); err != nil {
					logger.Error("relay send failed", "agent", ac.ID, "error", err)
				}
			}(ac)
		}
	}
	s.mu.RUnlock()
}

func (s *Server) handleQueryOnline(conn *protocol.Conn, reqMsg protocol.Message) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.agents))
	for id := range s.agents {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	data, _ := json.Marshal(ids)
	if err := conn.Send(protocol.Message{
		Type: protocol.MsgTypeOnlineList, From: "relay",
		ID: reqMsg.ID, Content: string(data),
	}); err != nil {
		logger.Error("failed to send online list", "error", err)
	}
}

func (s *Server) AgentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.agents)
}

func (s *Server) ChannelCount() int {
	s.channelIndexMu.RLock()
	defer s.channelIndexMu.RUnlock()
	return len(s.channelIndex)
}

func main() {
	addr := flag.String("addr", ":19200", "Relay listen address")
	httpAddr := flag.String("http", ":19201", "HTTP status address")
	flag.Parse()

	logger.Init("info", nil)

	srv := NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", srv.handleWS)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		srv.mu.RLock()
		ids := make([]string, 0, len(srv.agents))
		for id := range srv.agents {
			ids = append(ids, id)
		}
		srv.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agents":   ids,
			"count":    len(ids),
			"channels": srv.ChannelCount(),
		})
	})

	stopMDNS, err := p2p.RegisterRelay("lan-relay", parsePort(*addr))
	if err != nil {
		logger.Error("mDNS registration failed", "error", err)
	}

	logger.Info("relay listening", "address", *addr)
	relayServer := &http.Server{Addr: *addr, Handler: mux}
	go func() {
		if err := relayServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("relay listen failed", "error", err)
		}
	}()
	logger.Info("HTTP status server started", "address", *httpAddr)
	statusServer := &http.Server{Addr: *httpAddr, Handler: nil}
	go func() {
		if err := statusServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("status server failed", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutting down")
	relayServer.Close()
	statusServer.Close()
	if stopMDNS != nil {
		stopMDNS()
	}
}

func parsePort(addr string) int {
	_, p, _ := net.SplitHostPort(addr)
	port := 0
	fmt.Sscanf(p, "%d", &port)
	return port
}
