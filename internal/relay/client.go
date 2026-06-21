package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/fdwl/lan-a2a/internal/protocol"
)

type Client struct {
	agentID string
	server  string
	conn    *protocol.Conn
	mu      sync.Mutex
	done    chan struct{}

	OnMessage   func(msg protocol.Message, from string)
	OnFileData   func(msg protocol.Message, data io.Reader, from string)
	OnOnlineList func(ids []string)
}

func NewClient(agentID, serverAddr string) *Client {
	return &Client{agentID: agentID, server: serverAddr, done: make(chan struct{})}
}

func (c *Client) Connect() error {
	wsURL := fmt.Sprintf("ws://%s/ws", c.server)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect to relay %s: %w", c.server, err)
	}

	conn := protocol.NewConn(ws)

	if err := conn.Send(protocol.Message{
		Type: protocol.MsgTypeRegister, From: c.agentID, ID: protocol.NewMsgID(),
	}); err != nil {
		conn.Close()
		return err
	}
	resp, err := conn.Read()
	if err != nil || resp.Type != protocol.MsgTypeRegisterOK {
		conn.Close()
		return fmt.Errorf("relay register failed")
	}

	c.conn = conn
	log.Printf("[relay] connected to %s", c.server)
	go c.readLoop()
	go c.pingLoop()
	go c.queryLoop()
	return nil
}

func (c *Client) Stop() {
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) Send(msg protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Send(msg)
}

func (c *Client) QueryOnline() error {
	return c.Send(protocol.Message{
		Type: protocol.MsgTypeQueryOnline, From: c.agentID, ID: protocol.NewMsgID(),
	})
}

func (c *Client) readLoop() {
	defer func() {
		log.Printf("[relay] disconnected from %s", c.server)
	}()
	for {
		select {
		case <-c.done:
			return
		default:
		}
		msg, err := c.conn.Read()
		if err != nil {
			return
		}
		switch msg.Type {
		case protocol.MsgTypePing:
			c.conn.Send(protocol.Message{Type: protocol.MsgTypePong, From: c.agentID, ID: protocol.NewMsgID()})
		case protocol.MsgTypePong:
		case protocol.MsgTypeOnlineList:
			var ids []string
			if json.Unmarshal([]byte(msg.Content), &ids) == nil && c.OnOnlineList != nil {
				c.OnOnlineList(ids)
			}
		case protocol.MsgTypeText, protocol.MsgTypeFileMeta, protocol.MsgTypeFileDone:
			if c.OnMessage != nil {
				c.OnMessage(msg, msg.From)
			}
		case protocol.MsgTypeFileData:
			if c.OnFileData != nil {
				c.OnFileData(msg, protocol.BytesReader(msg.Data), msg.From)
			}
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.Send(protocol.Message{Type: protocol.MsgTypePing, From: c.agentID, ID: protocol.NewMsgID()})
		}
	}
}

func (c *Client) queryLoop() {
	c.QueryOnline()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.QueryOnline()
		}
	}
}
