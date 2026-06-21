package channel

import (
	"testing"
)

func TestCreateChannel(t *testing.T) {
	m := NewManager()
	ch := m.Create("ch-1", "test-channel", "agent-a", []string{"agent-a", "agent-b"}, ModeP2P)
	if ch.ID != "ch-1" {
		t.Errorf("expected ch-1, got %s", ch.ID)
	}
	if ch.Mode != ModeP2P {
		t.Errorf("expected P2P mode, got %s", ch.Mode)
	}
	if ch.Host != "agent-a" {
		t.Errorf("expected host agent-a, got %s", ch.Host)
	}
}

func TestGetChannel(t *testing.T) {
	m := NewManager()
	m.Create("ch-1", "test", "a", []string{"a"}, ModeP2P)
	ch, ok := m.Get("ch-1")
	if !ok || ch == nil {
		t.Error("channel should exist")
	}
	_, ok = m.Get("ch-999")
	if ok {
		t.Error("channel should not exist")
	}
}

func TestLeaveChannel(t *testing.T) {
	m := NewManager()
	m.Create("ch-1", "test", "a", []string{"a", "b", "c"}, ModeP2P)
	m.Leave("ch-1", "b")
	ch, _ := m.Get("ch-1")
	if len(ch.Members) != 2 {
		t.Errorf("expected 2 members after leave, got %d", len(ch.Members))
	}
	for _, id := range ch.Members {
		if id == "b" {
			t.Error("b should have left")
		}
	}
}

func TestLeaveHost(t *testing.T) {
	m := NewManager()
	m.Create("ch-1", "test", "a", []string{"a", "b"}, ModeP2P)
	m.Leave("ch-1", "a")
	ch, _ := m.Get("ch-1")
	if ch.Host != "b" {
		t.Errorf("host should switch to b, got %s", ch.Host)
	}
}

func TestLeaveLastMember(t *testing.T) {
	m := NewManager()
	m.Create("ch-1", "test", "a", []string{"a"}, ModeP2P)
	m.Leave("ch-1", "a")
	_, ok := m.Get("ch-1")
	if ok {
		t.Error("channel should be deleted when empty")
	}
}

func TestIsMember(t *testing.T) {
	m := NewManager()
	m.Create("ch-1", "test", "a", []string{"a", "b"}, ModeP2P)
	if !m.IsMember("ch-1", "a") {
		t.Error("a should be member")
	}
	if m.IsMember("ch-1", "c") {
		t.Error("c should not be member")
	}
	if m.IsMember("ch-999", "a") {
		t.Error("non-existent channel")
	}
}

func TestRelayMode(t *testing.T) {
	m := NewManager()
	ch := m.Create("ch-1", "relay-test", "a", []string{"a", "b"}, ModeRelay)
	if ch.Mode != ModeRelay {
		t.Errorf("expected relay mode, got %s", ch.Mode)
	}
	if m.GetMode("ch-1") != ModeRelay {
		t.Error("GetMode should return relay")
	}
}
