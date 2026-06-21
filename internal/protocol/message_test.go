package protocol

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewMsgID(t *testing.T) {
	id1 := NewMsgID()
	id2 := NewMsgID()
	if id1 == id2 {
		t.Error("NewMsgID should generate unique IDs")
	}
	if len(id1) < 10 {
		t.Error("NewMsgID too short")
	}
}

func TestConnSendReceive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()

		conn := NewConn(ws)
		msg, err := conn.Read()
		if err != nil {
			t.Fatal(err)
		}
		if msg.Type != MsgTypeText {
			t.Errorf("expected text, got %s", msg.Type)
		}
		if msg.Content != "hello" {
			t.Errorf("expected hello, got %s", msg.Content)
		}

		conn.Send(Message{
			Type:    MsgTypeText,
			ID:      NewMsgID(),
			From:    "server",
			Content: "world",
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	conn := NewConn(ws)

	err = conn.Send(Message{
		Type:    MsgTypeText,
		ID:      NewMsgID(),
		From:    "client",
		Content: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := conn.Read()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "world" {
		t.Errorf("expected world, got %s", resp.Content)
	}
}

func TestHandshake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		ws, _ := upgrader.Upgrade(w, r, nil)
		defer ws.Close()
		conn := NewConn(ws)

		msg, err := conn.Read()
		if err != nil || msg.Type != MsgTypeRegister {
			return
		}
		conn.Send(Message{Type: MsgTypeRegisterOK, From: "server", ID: NewMsgID()})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer ws.Close()
	conn := NewConn(ws)

	ok, err := handshakeAsClient(conn, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("handshake should succeed")
	}
}

func handshakeAsClient(conn *Conn, agentID string) (bool, error) {
	err := conn.Send(Message{Type: MsgTypeRegister, From: agentID, ID: NewMsgID()})
	if err != nil {
		return false, err
	}
	resp, err := conn.Read()
	if err != nil {
		return false, err
	}
	return resp.Type == MsgTypeRegisterOK, nil
}
