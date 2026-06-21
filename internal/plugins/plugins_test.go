package plugins

import (
	"sync"
	"testing"
)

type testPlugin struct {
	name   string
	events []*Event
	mu     sync.Mutex
}

func (p *testPlugin) Name() string { return p.name }
func (p *testPlugin) OnEvent(e *Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *testPlugin) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func TestRegisterAndEmit(t *testing.T) {
	m := NewManager()
	tp := &testPlugin{name: "test"}
	m.Register(tp)

	m.Emit(&Event{Type: "test.event"})
	m.Emit(&Event{Type: "other.event"})

	if tp.count() != 2 {
		t.Errorf("expected 2 events, got %d", tp.count())
	}
}

func TestHookFiltering(t *testing.T) {
	m := NewManager()
	var received []string

	m.On("transfer.start", func(e *Event) {
		received = append(received, "start")
	})
	m.On("transfer.done", func(e *Event) {
		received = append(received, "done")
	})

	m.Emit(&Event{Type: "transfer.start"})
	m.Emit(&Event{Type: "transfer.done"})
	m.Emit(&Event{Type: "other.event"})

	if len(received) != 2 {
		t.Errorf("expected 2 hook calls, got %d", len(received))
	}
}

func TestFilterPlugin(t *testing.T) {
	m := NewManager()
	inner := &testPlugin{name: "inner"}
	filter := &FilterPlugin{
		Match: func(e *Event) bool { return e.Type == "important" },
		Inner: inner,
	}
	m.Register(filter)

	m.Emit(&Event{Type: "important"})
	m.Emit(&Event{Type: "noise"})
	m.Emit(&Event{Type: "important"})

	if inner.count() != 2 {
		t.Errorf("expected 2 events for inner, got %d", inner.count())
	}
}

func TestTransformPlugin(t *testing.T) {
	m := NewManager()
	inner := &testPlugin{name: "inner"}
	transform := &TransformPlugin{
		Transform: func(e *Event) *Event {
			return &Event{Type: "transformed:" + e.Type}
		},
		Inner: inner,
	}
	m.Register(transform)

	m.Emit(&Event{Type: "original"})

	if inner.count() != 1 {
		t.Fatal("expected 1 event")
	}
	if inner.events[0].Type != "transformed:original" {
		t.Errorf("expected transformed:original, got %s", inner.events[0].Type)
	}
}
