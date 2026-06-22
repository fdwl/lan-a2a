package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/fdwl/lan-a2a/internal/logger"
	"github.com/fdwl/lan-a2a/internal/protocol"
)

type Client struct {
	agentID string
	server  string
	conn    *protocol.Conn
	mu      sync.Mutex
	done    chan struct{}
	stopped bool

	maxRetries int

	OnMessage    func(msg protocol.Message, from string)
	OnFileData   func(msg protocol.Message, data io.Reader, from string)
	OnOnlineList func(ids []string)
	OnGoodbye    func(from string)
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
	c.maxRetries = 0
	logger.Info("connected", "server", c.server)
	go c.readLoop()
	go c.pingLoop()
	go c.queryLoop()
	return nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	c.stopped = true
	c.mu.Unlock()
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil && !c.stopped
}

func (c *Client) reconnect() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	for {
		backoff := c.nextBackoff()
		logger.Info("reconnecting", "server", c.server, "backoff", backoff, "attempt", c.maxRetries+1)
		time.Sleep(backoff)

		c.mu.Lock()
		if c.stopped {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()

		if err := c.Connect(); err != nil {
			logger.Error("reconnect failed", "error", err)
			continue
		}
		logger.Info("reconnected", "server", c.server)
		return
	}
}

func (c *Client) nextBackoff() time.Duration {
	c.maxRetries++
	backoff := time.Second * time.Duration(1<<uint(c.maxRetries-1))
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	return backoff
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

func (c *Client) SendGoodbye() error {
	return c.Send(protocol.Message{
		Type: protocol.MsgTypeGoodbye, From: c.agentID, ID: protocol.NewMsgID(),
	})
}

func (c *Client) readLoop() {
	defer func() {
		logger.Info("disconnected", "server", c.server)
	}()
	for {
		select {
		case <-c.done:
			return
		default:
		}
		msg, err := c.conn.Read()
		if err != nil {
			c.mu.Lock()
			stopped := c.stopped
			c.mu.Unlock()
			if stopped {
				return
			}
			go c.reconnect()
			return
		}
		switch msg.Type {
		case protocol.MsgTypePing:
			if err := c.conn.Send(protocol.Message{Type: protocol.MsgTypePong, From: c.agentID, ID: protocol.NewMsgID()}); err != nil {
				c.mu.Lock()
				stopped := c.stopped
				c.mu.Unlock()
				if stopped {
					return
				}
				go c.reconnect()
				return
			}
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
		case protocol.MsgTypeGoodbye:
			if c.OnGoodbye != nil {
				c.OnGoodbye(msg.From)
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
