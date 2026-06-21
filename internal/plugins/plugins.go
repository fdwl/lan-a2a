package plugins

import (
	"sync"
)

const (
	EventTransferStart  = "transfer.start"
	EventChunkDone      = "chunk.done"
	EventTransferDone   = "transfer.done"
	EventFolderSyncStart = "folder_sync.start"
	EventFolderSyncDone  = "folder_sync.done"
	EventFileReceived   = "file.received"
)

type Event struct {
	Type     string
	Transfer interface{}
	Chunk    interface{}
	PeerID   string
	Folder   string
	Diff     interface{}
	Result   interface{}
	Data     map[string]interface{}
}

type Hook func(event *Event)

type Plugin interface {
	Name() string
	OnEvent(event *Event)
}

type Manager struct {
	plugins []Plugin
	hooks   map[string][]Hook
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		hooks: make(map[string][]Hook),
	}
}

func (m *Manager) Register(p Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins = append(m.plugins, p)
}

func (m *Manager) On(eventType string, hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks[eventType] = append(m.hooks[eventType], hook)
}

func (m *Manager) Emit(event *Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Plugin handlers
	for _, p := range m.plugins {
		p.OnEvent(event)
	}

	// Direct hook handlers
	for _, hook := range m.hooks[event.Type] {
		hook(event)
	}
}

// --- Built-in plugins ---

type LogPlugin struct{}

func (p *LogPlugin) Name() string { return "log" }
func (p *LogPlugin) OnEvent(e *Event) {
	switch e.Type {
	case EventTransferStart:
		t := e.Transfer.(interface{ GetID() string })
		_ = t
	case EventChunkDone:
	case EventTransferDone:
	case EventFolderSyncStart:
	case EventFolderSyncDone:
	}
}

type ProgressPlugin struct {
	OnProgress func(event *Event)
}

func (p *ProgressPlugin) Name() string { return "progress" }
func (p *ProgressPlugin) OnEvent(e *Event) {
	if p.OnProgress != nil {
		p.OnProgress(e)
	}
}

// FilterPlugin skips events that don't match a predicate.
type FilterPlugin struct {
	Match func(event *Event) bool
	Inner Plugin
}

func (p *FilterPlugin) Name() string { return "filter:" + p.Inner.Name() }
func (p *FilterPlugin) OnEvent(e *Event) {
	if p.Match(e) {
		p.Inner.OnEvent(e)
	}
}

// TransformPlugin modifies event data before passing to inner plugin.
type TransformPlugin struct {
	Transform func(event *Event) *Event
	Inner     Plugin
}

func (p *TransformPlugin) Name() string { return "transform:" + p.Inner.Name() }
func (p *TransformPlugin) OnEvent(e *Event) {
	transformed := p.Transform(e)
	if transformed != nil {
		p.Inner.OnEvent(transformed)
	}
}
