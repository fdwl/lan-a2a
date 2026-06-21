package filetransfer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ChunkSize       = 64 * 1024
	DownloadBaseDir = ".lan-agent-bus/downloads"
)

type IncomingFile struct {
	ChannelID   string
	Filename    string
	FileSize    int64
	From        string
	ReceivedAt  time.Time
	TotalChunks int
	Chunks      map[int][]byte
	LocalPath   string
}

type Manager struct {
	agentID    string
	incoming   map[string]*IncomingFile
	mu         sync.RWMutex
	OnComplete func(file *IncomingFile)
}

func NewManager(agentID string) *Manager {
	return &Manager{agentID: agentID, incoming: make(map[string]*IncomingFile)}
}

func (m *Manager) DownloadDir(channelID string) string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, DownloadBaseDir, channelID)
	os.MkdirAll(dir, 0755)
	return dir
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	if name == "." || name == ".." || name == "/" {
		return "unnamed"
	}
	return name
}

func (m *Manager) PrepareIncoming(channelID, msgID, from, filename string, fileSize int64, totalChunks int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	safeFilename := sanitizeFilename(filename)
	fi := &IncomingFile{
		ChannelID: channelID, Filename: safeFilename, FileSize: fileSize,
		From: from, ReceivedAt: time.Now(), TotalChunks: totalChunks,
		Chunks: make(map[int][]byte),
	}
	m.incoming[msgID] = fi
	fi.LocalPath = filepath.Join(m.DownloadDir(channelID), safeFilename)
	return fi.LocalPath
}

func (m *Manager) AddChunk(msgID string, chunkIdx int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fi, ok := m.incoming[msgID]
	if !ok {
		return fmt.Errorf("unknown transfer: %s", msgID)
	}
	fi.Chunks[chunkIdx] = data
	if len(fi.Chunks) == fi.TotalChunks {
		return m.assemble(fi)
	}
	return nil
}

func (m *Manager) assemble(fi *IncomingFile) error {
	f, err := os.Create(fi.LocalPath)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	for i := 0; i < fi.TotalChunks; i++ {
		chunk, ok := fi.Chunks[i]
		if !ok {
			return fmt.Errorf("missing chunk %d", i)
		}
		f.Write(chunk)
		hash.Write(chunk)
	}
	log.Printf("[file] saved: %s -> %s", fi.Filename, fi.LocalPath)
	if m.OnComplete != nil {
		m.OnComplete(fi)
	}
	return nil
}

func SplitFile(filePath string) (chunks [][]byte, checksum string, size int64, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return
	}
	size = stat.Size()
	hash := sha256.New()
	buf := make([]byte, ChunkSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			c := make([]byte, n)
			copy(c, buf[:n])
			chunks = append(chunks, c)
			hash.Write(c)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = readErr
			return
		}
	}
	checksum = hex.EncodeToString(hash.Sum(nil))
	return
}
