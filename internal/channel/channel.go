package channel

import (
	"fmt"
	"sync"
	"time"
)

type ChannelMode string

const (
	ModeP2P   ChannelMode = "p2p"   // P2P 广播，第一个成员为 Host
	ModeRelay ChannelMode = "relay" // 通过 Relay 服务器中转
)

type Channel struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Mode      ChannelMode `json:"mode"`
	Host      string      `json:"host"`       // P2P 模式下的 Host agent ID
	Members   []string    `json:"members"`
	CreatedAt time.Time   `json:"created_at"`
	Creator   string      `json:"creator"`
}

type Manager struct {
	channels map[string]*Channel
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{channels: make(map[string]*Channel)}
}

func (m *Manager) Create(id, name, creator string, members []string, mode ChannelMode) *Channel {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := &Channel{
		ID:        id,
		Name:      name,
		Mode:      mode,
		Host:      creator,
		Members:   members,
		CreatedAt: time.Now(),
		Creator:   creator,
	}
	m.channels[id] = ch
	return ch
}

func (m *Manager) Get(id string) (*Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[id]
	return ch, ok
}

func (m *Manager) Leave(id, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.channels[id]
	if !ok {
		return fmt.Errorf("channel %s not found", id)
	}
	for i, mid := range ch.Members {
		if mid == agentID {
			ch.Members = append(ch.Members[:i], ch.Members[i+1:]...)
			break
		}
	}
	// 如果 Host 离开了，切换 Host
	if ch.Host == agentID && len(ch.Members) > 0 {
		ch.Host = ch.Members[0]
	}
	if len(ch.Members) == 0 {
		delete(m.channels, id)
	}
	return nil
}

func (m *Manager) IsMember(id, agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[id]
	if !ok {
		return false
	}
	for _, mid := range ch.Members {
		if mid == agentID {
			return true
		}
	}
	return false
}

func (m *Manager) GetHost(id string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[id]
	if !ok {
		return "", false
	}
	return ch.Host, true
}

func (m *Manager) GetMode(id string) ChannelMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.channels[id]
	if !ok {
		return ModeP2P
	}
	return ch.Mode
}

func (m *Manager) ListByPeer(agentID string) []*Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Channel
	for _, ch := range m.channels {
		for _, mid := range ch.Members {
			if mid == agentID {
				result = append(result, ch)
				break
			}
		}
	}
	return result
}
