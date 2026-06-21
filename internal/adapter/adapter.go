package adapter

import (
	"github.com/fdwl/lan-a2a/internal/profile"
)

// AgentCard represents an A2A-compatible agent card for LAN discovery.
// Maps to A2A AgentCard: https://a2a-protocol.org/latest/specification/#agentcard
type AgentCard struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url"`
	Version     string            `json:"version"`
	Capabilities AgentCapabilities `json:"capabilities"`
	Skills      []Skill           `json:"skills,omitempty"`
	DefaultInputModes  []string   `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string   `json:"defaultOutputModes,omitempty"`
}

type AgentCapabilities struct {
	Streaming     bool `json:"streaming,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// ProfileToCard converts a Profile to an A2A AgentCard.
func ProfileToCard(p *profile.Profile, addr string) *AgentCard {
	return &AgentCard{
		Name:        p.Name,
		Description: p.Metadata["description"],
		URL:         addr,
		Version:     "0.1.0",
		Capabilities: AgentCapabilities{
			Streaming: true,
		},
		Skills: profileToSkills(p),
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
	}
}

func profileToSkills(p *profile.Profile) []Skill {
	var skills []Skill
	for _, role := range p.Roles {
		skills = append(skills, Skill{
			ID:   role,
			Name: role,
		})
	}
	return skills
}

// CardToProfile converts an A2A AgentCard to a Profile.
func CardToProfile(card *AgentCard) *profile.Profile {
	p := &profile.Profile{
		Name:  card.Name,
		Roles: cardToRoles(card),
		Metadata: map[string]string{
			"description": card.Description,
			"url":         card.URL,
			"version":     card.Version,
		},
	}
	return p
}

func cardToRoles(card *AgentCard) []string {
	var roles []string
	for _, skill := range card.Skills {
		roles = append(roles, skill.ID)
	}
	return roles
}

// TaskState maps A2A task states.
type TaskState string

const (
	TaskStateSubmitted    TaskState = "submitted"
	TaskStateWorking      TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted    TaskState = "completed"
	TaskStateFailed       TaskState = "failed"
	TaskStateCanceled     TaskState = "canceled"
)

// Task represents an A2A-compatible task.
type Task struct {
	ID       string      `json:"id"`
	State    TaskState   `json:"state"`
	Messages []Message   `json:"messages,omitempty"`
}

// Message represents an A2A-compatible message.
type Message struct {
	Role  string      `json:"role"`
	Parts []Part      `json:"parts"`
}

type Part struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     []byte `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func NewTextMessage(role, text string) Message {
	return Message{
		Role:  role,
		Parts: []Part{{Type: "text", Text: text}},
	}
}
