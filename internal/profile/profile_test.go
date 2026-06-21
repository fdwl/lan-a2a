package profile

import (
	"testing"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()
	if id1 == "" || id2 == "" {
		t.Error("IDs should not be empty")
	}
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
	if len(id1) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(id1))
	}
}

func TestValidate(t *testing.T) {
	p := &Profile{Name: "test"}
	if err := p.Validate(); err == nil {
		t.Error("should fail without ID")
	}

	p = &Profile{ID: "test-id"}
	if err := p.Validate(); err == nil {
		t.Error("should fail without name")
	}

	p = &Profile{ID: "test-id", Name: "test"}
	if err := p.Validate(); err != nil {
		t.Errorf("should pass: %v", err)
	}
}

func TestSetTimestamps(t *testing.T) {
	p := &Profile{ID: "test", Name: "test"}
	p.SetTimestamps()
	if p.Created == 0 {
		t.Error("created should be set")
	}
	if p.Updated == 0 {
		t.Error("updated should be set")
	}
	if p.Version != 1 {
		t.Error("version should default to 1")
	}

	origCreated := p.Created
	p.SetTimestamps()
	if p.Created != origCreated {
		t.Error("created should not change")
	}
}

func TestManagerCRUD(t *testing.T) {
	m := NewManager()
	p1 := &Profile{ID: "a", Name: "Agent A"}
	p2 := &Profile{ID: "b", Name: "Agent B"}

	m.Set(p1)
	m.Set(p2)
	if m.Count() != 2 {
		t.Errorf("expected 2, got %d", m.Count())
	}

	got, ok := m.Get("a")
	if !ok || got.Name != "Agent A" {
		t.Error("Get failed")
	}

	m.Remove("a")
	if m.Count() != 1 {
		t.Error("Remove failed")
	}

	list := m.List()
	if len(list) != 1 {
		t.Error("List failed")
	}
}

func TestProfileRoles(t *testing.T) {
	p := &Profile{
		ID:   "test",
		Name: "Agent",
		Roles: []string{"frontend", "react"},
		Avatar: "https://example.com/avatar.png",
		Tags:   []string{"senior"},
		Metadata: map[string]string{"team": "platform"},
	}

	if !p.HasRole("frontend") {
		t.Error("should have frontend role")
	}
	if p.HasRole("backend") {
		t.Error("should not have backend role")
	}

	p.AddRole("backend")
	if !p.HasRole("backend") {
		t.Error("should have backend role after add")
	}
	if len(p.Roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(p.Roles))
	}

	// Duplicate add
	p.AddRole("frontend")
	if len(p.Roles) != 3 {
		t.Error("should not add duplicate role")
	}

	if p.Avatar != "https://example.com/avatar.png" {
		t.Error("avatar mismatch")
	}
	if p.Metadata["team"] != "platform" {
		t.Error("metadata mismatch")
	}
}
