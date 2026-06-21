package filetransfer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fdwl/lan-a2a/internal/logger"
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
	Received    map[int]bool
	file        *os.File
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
		Received: make(map[int]bool),
	}
	fi.LocalPath = filepath.Join(m.DownloadDir(channelID), safeFilename)
	f, err := os.Create(fi.LocalPath)
	if err != nil {
		logger.Error("failed to create file", "path", fi.LocalPath, "error", err)
	} else {
		fi.file = f
	}
	m.incoming[msgID] = fi
	return fi.LocalPath
}

func (m *Manager) AddChunk(msgID string, chunkIdx int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fi, ok := m.incoming[msgID]
	if !ok {
		return fmt.Errorf("unknown transfer: %s", msgID)
	}
	if fi.file != nil {
		offset := int64(chunkIdx) * ChunkSize
		if _, err := fi.file.WriteAt(data, offset); err != nil {
			return fmt.Errorf("write chunk %d: %w", chunkIdx, err)
		}
	}
	fi.Received[chunkIdx] = true
	if len(fi.Received) == fi.TotalChunks {
		return m.assemble(fi)
	}
	return nil
}

func (m *Manager) assemble(fi *IncomingFile) error {
	if fi.file != nil {
		fi.file.Close()
		fi.file = nil
	}
	f, err := os.Open(fi.LocalPath)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	logger.Info("file saved", "filename", fi.Filename, "path", fi.LocalPath, "checksum", hex.EncodeToString(hash.Sum(nil)))
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
