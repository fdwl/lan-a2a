package fileservice

import (
	"fmt"
	"sync"
	"time"

	"github.com/fdwl/lan-a2a/internal/logger"
	"github.com/fdwl/lan-a2a/internal/plugins"
)

type SendFunc func(peerID string, chunkData []byte, transfer *Transfer, chunk *Chunk) error

type FileService struct {
	transfers map[string]*Transfer
	mu        sync.RWMutex
	plugins   *plugins.Manager
	sendFunc  SendFunc
	maxWorkers int
	done      chan struct{}
}

func NewFileService(sendFunc SendFunc, maxWorkers int) *FileService {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	return &FileService{
		transfers:  make(map[string]*Transfer),
		plugins:    plugins.NewManager(),
		sendFunc:   sendFunc,
		maxWorkers: maxWorkers,
		done:       make(chan struct{}),
	}
}

func (fs *FileService) Stop() {
	close(fs.done)
}

func (fs *FileService) Plugins() *plugins.Manager {
	return fs.plugins
}

// SendFile splits a file into chunks and sends them concurrently to a peer.
func (fs *FileService) SendFile(peerID, filePath string) (*Transfer, error) {
	chunks, checksum, fileSize, err := SplitIntoChunks(filePath, DefaultChunkSize)
	if err != nil {
		return nil, err
	}

	t := &Transfer{
		ID:          fmt.Sprintf("tr-%d", time.Now().UnixNano()),
		FilePath:    filePath,
		FileSize:    fileSize,
		FileHash:    checksum,
		ChunkSize:   DefaultChunkSize,
		TotalChunks: len(chunks),
		Chunks:      chunks,
		Status:      StatusRunning,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	fs.mu.Lock()
	fs.transfers[t.ID] = t
	fs.mu.Unlock()

	fs.plugins.Emit(&plugins.Event{
		Type:     plugins.EventTransferStart,
		Transfer: t,
		PeerID:   peerID,
	})

	go fs.sendConcurrent(peerID, t)

	return t, nil
}

func (fs *FileService) sendConcurrent(peerID string, t *Transfer) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, fs.maxWorkers)
	var failedMu sync.Mutex
	var failed bool

	for _, chunk := range t.Chunks {
		select {
		case <-fs.done:
			return
		default:
		}

		if chunk.Status == ChunkDone {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(c *Chunk) {
			defer wg.Done()
			defer func() { <-sem }()

			for c.Retries < 3 {
				select {
				case <-fs.done:
					return
				default:
				}

				data, err := ReadChunk(t.FilePath, c)
				if err != nil {
					c.Retries++
					continue
				}

				c.Status = ChunkSending
				err = fs.sendFunc(peerID, data, t, c)
				if err != nil {
					c.Retries++
					c.Status = ChunkFailed
					continue
				}

				c.Status = ChunkDone
				fs.updateProgress(t)

				fs.plugins.Emit(&plugins.Event{
					Type:     plugins.EventChunkDone,
					Transfer: t,
					Chunk:    c,
					PeerID:   peerID,
				})
				return
			}

			failedMu.Lock()
			failed = true
			failedMu.Unlock()
		}(chunk)
	}

	wg.Wait()

	if failed {
		t.Status = StatusFailed
	} else {
		t.Status = StatusCompleted
	}
	t.UpdatedAt = time.Now()

	fs.plugins.Emit(&plugins.Event{
		Type:     plugins.EventTransferDone,
		Transfer: t,
		PeerID:   peerID,
	})
}

func (fs *FileService) updateProgress(t *Transfer) {
	done := 0
	for _, c := range t.Chunks {
		if c.Status == ChunkDone {
			done++
		}
	}
	t.Progress = float64(done) / float64(t.TotalChunks) * 100
	t.UpdatedAt = time.Now()
}

func (fs *FileService) GetTransfer(id string) (*Transfer, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	t, ok := fs.transfers[id]
	return t, ok
}

func (fs *FileService) PauseTransfer(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	t, ok := fs.transfers[id]
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.Status = StatusPaused
	return nil
}

func (fs *FileService) ResumeTransfer(id string, peerID string) error {
	fs.mu.RLock()
	t, ok := fs.transfers[id]
	fs.mu.RUnlock()
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.Status = StatusRunning
	go fs.sendConcurrent(peerID, t)
	return nil
}

func (fs *FileService) CancelTransfer(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	t, ok := fs.transfers[id]
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.Status = StatusFailed
	return nil
}

func (fs *FileService) ListTransfers() []*Transfer {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	result := make([]*Transfer, 0, len(fs.transfers))
	for _, t := range fs.transfers {
		result = append(result, t)
	}
	return result
}

func (fs *FileService) LogStatus() {
	for _, t := range fs.ListTransfers() {
		logger.Info("transfer status", "transfer_id", t.ID, "status", t.Status, "progress", t.Progress, "done_chunks", fs.doneCount(t), "total_chunks", t.TotalChunks)
	}
}

func (fs *FileService) doneCount(t *Transfer) int {
	n := 0
	for _, c := range t.Chunks {
		if c.Status == ChunkDone {
			n++
		}
	}
	return n
}
