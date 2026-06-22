package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type MsgType string

const (
	MsgTypeRegister     MsgType = "register"
	MsgTypeRegisterOK   MsgType = "register_ok"
	MsgTypePing         MsgType = "ping"
	MsgTypePong         MsgType = "pong"
	MsgTypeText         MsgType = "text"
	MsgTypeFileMeta     MsgType = "file_meta"
	MsgTypeFileData     MsgType = "file_data"
	MsgTypeFileDone     MsgType = "file_done"
	MsgTypeQueryOnline  MsgType = "query_online"
	MsgTypeOnlineList   MsgType = "online_list"
	MsgTypeGoodbye      MsgType = "goodbye"
)

type Message struct {
	Type        MsgType `json:"type"`
	ID          string  `json:"id"`
	From        string  `json:"from"`
	ChannelID   string  `json:"channel_id,omitempty"`
	Content     string  `json:"content,omitempty"`
	Timestamp   int64   `json:"ts"`
	Filename    string  `json:"filename,omitempty"`
	FileSize    int64   `json:"file_size,omitempty"`
	Checksum    string  `json:"checksum,omitempty"`
	ChunkIdx    int     `json:"chunk_idx,omitempty"`
	TotalChunks int     `json:"total_chunks,omitempty"`
	Data        []byte  `json:"data,omitempty"`
	Profile     *ProfilePayload `json:"profile,omitempty"`
}

type ProfilePayload struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Avatar  string   `json:"avatar,omitempty"`
	Roles   []string `json:"roles,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func NewMsgID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

// Conn wraps a WebSocket connection with read/write methods.
type Conn struct {
	WS   *websocket.Conn
	Mu   sync.Mutex
}

func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{WS: ws}
}

func (c *Conn) Send(msg Message) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.WS.WriteJSON(msg)
}

func (c *Conn) Read() (Message, error) {
	var msg Message
	err := c.WS.ReadJSON(&msg)
	return msg, err
}

func (c *Conn) Close() {
	c.WS.Close()
}

// UpgradeHTTP upgrades an HTTP request to WebSocket.
var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func BytesReader(b []byte) io.Reader {
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
