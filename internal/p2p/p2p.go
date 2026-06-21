package p2p

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/grandcat/zeroconf"
	"github.com/fdwl/lan-a2a/internal/logger"
	"github.com/fdwl/lan-a2a/internal/protocol"
)

const (
	ServiceType      = "_lan-agent-bus._tcp"
	RelayServiceType = "_lan-agent-bus-relay._tcp"
	ServiceDomain    = "local."
)

type OnlinePeer struct {
	ID       string
	Addr     string
	LastSeen time.Time
	Profile  *protocol.ProfilePayload
}

type P2P struct {
	agentID string
	port    int
	server  *http.Server
	done    chan struct{}
	profile *protocol.ProfilePayload

	online   map[string]*OnlinePeer
	onlineMu sync.RWMutex

	OnMessage func(msg protocol.Message, from string)
	OnFileData func(msg protocol.Message, data io.Reader, from string)
}

func New(agentID string, port int) *P2P {
	return &P2P{
		agentID: agentID,
		port:    port,
		online:  make(map[string]*OnlinePeer),
		done:    make(chan struct{}),
	}
}

func (p *P2P) SetProfile(prof *protocol.ProfilePayload) {
	p.profile = prof
}

func (p *P2P) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.handleWS)

	p.server = &http.Server{Addr: fmt.Sprintf(":%d", p.port), Handler: mux}
	go func() {
		logger.Info("listening", "port", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()
	go p.cleanupStalePeers()
	return nil
}

func (p *P2P) cleanupStalePeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.onlineMu.Lock()
			for id, peer := range p.online {
				if time.Since(peer.LastSeen) > 60*time.Second {
					delete(p.online, id)
					logger.Info("removed stale peer", "peer_id", id)
				}
			}
			p.onlineMu.Unlock()
		}
	}
}

func (p *P2P) Stop() {
	close(p.done)
	p.server.Close()
}

func (p *P2P) Port() int { return p.port }

func (p *P2P) MarkOnline(peerID, addr string) {
	p.onlineMu.Lock()
	p.online[peerID] = &OnlinePeer{ID: peerID, Addr: addr, LastSeen: time.Now()}
	p.onlineMu.Unlock()
}

func (p *P2P) GetOnlinePeers() []string {
	p.onlineMu.RLock()
	defer p.onlineMu.RUnlock()
	ids := make([]string, 0, len(p.online))
	for id := range p.online {
		ids = append(ids, id)
	}
	return ids
}

func (p *P2P) IsOnline(peerID string) bool {
	p.onlineMu.RLock()
	defer p.onlineMu.RUnlock()
	_, ok := p.online[peerID]
	return ok
}

// OpenConn opens an on-demand WebSocket connection to a peer.
func (p *P2P) OpenConn(peerID string) (*protocol.Conn, error) {
	p.onlineMu.RLock()
	peer, ok := p.online[peerID]
	p.onlineMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("peer %s not online", peerID)
	}

	wsURL := fmt.Sprintf("ws://%s/ws", peer.Addr)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to %s failed: %w", peerID, err)
	}

	conn := protocol.NewConn(ws)

	// Handshake with profile exchange
	if err := conn.Send(protocol.Message{
		Type: protocol.MsgTypeRegister, From: p.agentID, ID: protocol.NewMsgID(),
		Profile: p.profile,
	}); err != nil {
		conn.Close()
		return nil, err
	}
	resp, err := conn.Read()
	if err != nil || resp.Type != protocol.MsgTypeRegisterOK {
		conn.Close()
		return nil, fmt.Errorf("handshake failed")
	}

	// Store remote profile
	if resp.Profile != nil {
		p.onlineMu.Lock()
		if peer, ok := p.online[peerID]; ok {
			peer.Profile = resp.Profile
		}
		p.onlineMu.Unlock()
	}

	return conn, nil
}

func (p *P2P) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := protocol.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("upgrade error", "error", err)
		return
	}
	conn := protocol.NewConn(ws)

	// Handshake: expect register with profile
	msg, err := conn.Read()
	if err != nil || msg.Type != protocol.MsgTypeRegister {
		conn.Close()
		return
	}
	if err := conn.Send(protocol.Message{
		Type: protocol.MsgTypeRegisterOK, From: p.agentID, ID: protocol.NewMsgID(),
		Profile: p.profile,
	}); err != nil {
		conn.Close()
		return
	}

	// Store peer with profile
	p.onlineMu.Lock()
	if existing, ok := p.online[msg.From]; ok {
		existing.LastSeen = time.Now()
		if msg.Profile != nil {
			existing.Profile = msg.Profile
		}
	} else {
		addr := r.RemoteAddr
		p.online[msg.From] = &OnlinePeer{
			ID: msg.From, Addr: addr, LastSeen: time.Now(),
			Profile: msg.Profile,
		}
	}
	p.onlineMu.Unlock()

	// Read messages until closed
	for {
		msg, err := conn.Read()
		if err != nil {
			return
		}
		switch msg.Type {
		case protocol.MsgTypeText, protocol.MsgTypeFileMeta, protocol.MsgTypeFileDone:
			if p.OnMessage != nil {
				p.OnMessage(msg, msg.From)
			}
		case protocol.MsgTypeFileData:
			if p.OnFileData != nil {
				p.OnFileData(msg, protocol.BytesReader(msg.Data), msg.From)
			}
		}
	}
}

// --- Discovery ---

func StartDiscovery(agentID string, port int, onFound func(peerID, addr string, port int), onRelay func(addr string)) (stop func(), err error) {
	srv, err := zeroconf.Register(agentID, ServiceType, ServiceDomain, port,
		[]string{"id=" + agentID, "version=0.1.0"}, nil)
	if err != nil {
		return nil, err
	}
	logger.Info("mDNS registered", "agent_id", agentID)

	resolver, _ := zeroconf.NewResolver(nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	browseAll := func() {
		browse(ctx, resolver, ServiceType, agentID, onFound)
		if onRelay != nil {
			browseRelay(ctx, resolver, onRelay)
		}
	}

	go func() {
		defer close(done)
		browseAll()
		intervals := []time.Duration{3 * time.Second, 3 * time.Second, 3 * time.Second, 5 * time.Second}
		for _, iv := range intervals {
			select {
			case <-ctx.Done():
				return
			case <-time.After(iv):
				browseAll()
			}
		}
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				browseAll()
			}
		}
	}()

	return func() {
		cancel()
		srv.Shutdown()
		<-done
	}, nil
}

func browse(ctx context.Context, resolver *zeroconf.Resolver, svcType string, selfID string, onFound func(string, string, int)) {
	browseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry)
	go func() { resolver.Browse(browseCtx, svcType, ServiceDomain, entries) }()
	for entry := range entries {
		if entry.Instance == selfID {
			continue
		}
		var peerID string
		for _, txt := range entry.Text {
			if len(txt) > 3 && txt[:3] == "id=" {
				peerID = txt[3:]
			}
		}
		if peerID == "" {
			peerID = entry.Instance
		}
		addr := ""
		for _, ip := range entry.AddrIPv4 {
			addr = ip.String()
			break
		}
		if addr != "" {
			onFound(peerID, addr, entry.Port)
		}
	}
}

func browseRelay(ctx context.Context, resolver *zeroconf.Resolver, onFound func(string)) {
	browseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry)
	go func() { resolver.Browse(browseCtx, RelayServiceType, ServiceDomain, entries) }()
	for entry := range entries {
		addr := ""
		for _, ip := range entry.AddrIPv4 {
			addr = ip.String()
			break
		}
		if addr != "" {
			onFound(fmt.Sprintf("%s:%d", addr, entry.Port))
		}
	}
}

func RegisterRelay(name string, port int) (func(), error) {
	srv, err := zeroconf.Register(name, RelayServiceType, ServiceDomain, port,
		[]string{"id=" + name, "version=0.1.0", "role=relay"}, nil)
	if err != nil {
		return nil, err
	}
	logger.Info("mDNS relay registered", "name", name, "port", port)
	return func() { srv.Shutdown() }, nil
}

func bytesReader(b []byte) io.Reader {
	if b == nil {
		return nil
	}
	return &sliceReader{data: b}
}

type sliceReader struct {
	data []byte
	off  int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
