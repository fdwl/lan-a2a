package fileservice

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"
)

type TransferStatus string

const (
	StatusPending    TransferStatus = "pending"
	StatusRunning    TransferStatus = "running"
	StatusCompleted  TransferStatus = "completed"
	StatusFailed     TransferStatus = "failed"
	StatusPaused     TransferStatus = "paused"
)

type ChunkStatus string

const (
	ChunkPending   ChunkStatus = "pending"
	ChunkSending   ChunkStatus = "sending"
	ChunkDone      ChunkStatus = "done"
	ChunkFailed    ChunkStatus = "failed"
)

type Chunk struct {
	Index    int        `json:"index"`
	Offset   int64      `json:"offset"`
	Size     int        `json:"size"`
	Hash     string     `json:"hash"`
	Status   ChunkStatus `json:"status"`
	Retries  int        `json:"retries"`
}

type Transfer struct {
	ID          string         `json:"id"`
	FilePath    string         `json:"file_path"`
	FileName    string         `json:"file_name"`
	FileSize    int64          `json:"file_size"`
	FileHash    string         `json:"file_hash"`
	ChunkSize   int            `json:"chunk_size"`
	TotalChunks int            `json:"total_chunks"`
	Chunks      []*Chunk       `json:"chunks"`
	Status      TransferStatus `json:"status"`
	Progress    float64        `json:"progress"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type FolderManifest struct {
	Root      string              `json:"root"`
	Files     map[string]FileEntry `json:"files"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type FileEntry struct {
	Path    string `json:"path"`
	RelPath string `json:"rel_path"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
	ModTime int64  `json:"mod_time"`
}

type SyncAction string

const (
	ActionAdd    SyncAction = "add"
	ActionModify SyncAction = "modify"
	ActionDelete SyncAction = "delete"
)

type SyncDiff struct {
	Root  string      `json:"root"`
	Adds  []FileEntry `json:"adds"`
	Modifies []FileEntry `json:"modifies"`
	Deletes []string    `json:"deletes"`
}

type SyncResult struct {
	Diff      *SyncDiff `json:"diff"`
	Transfers []string  `json:"transfer_ids"`
	StartedAt time.Time `json:"started_at"`
}

const DefaultChunkSize = 64 * 1024 // 64KB

func HashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func SplitIntoChunks(filePath string, chunkSize int) ([]*Chunk, string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", 0, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, "", 0, err
	}
	fileSize := stat.Size()

	fileHash := sha256.New()
	io.Copy(fileHash, f)
	checksum := hex.EncodeToString(fileHash.Sum(nil))

	f.Seek(0, 0)
	var chunks []*Chunk
	offset := int64(0)
	idx := 0
	buf := make([]byte, chunkSize)

	for {
		n, err := f.Read(buf)
		if n > 0 {
			chunks = append(chunks, &Chunk{
				Index:  idx,
				Offset: offset,
				Size:   n,
				Hash:   HashBytes(buf[:n]),
				Status: ChunkPending,
			})
			offset += int64(n)
			idx++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", 0, err
		}
	}

	return chunks, checksum, fileSize, nil
}

func ReadChunk(filePath string, chunk *Chunk) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	f.Seek(chunk.Offset, 0)
	buf := make([]byte, chunk.Size)
	n, err := f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func WalkFolder(root string) (map[string]FileEntry, error) {
	entries := make(map[string]FileEntry)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		hash, size, _ := HashFile(path)
		entries[rel] = FileEntry{
			Path:    path,
			RelPath: rel,
			Size:    size,
			Hash:    hash,
			ModTime: info.ModTime().Unix(),
		}
		return nil
	})
	return entries, err
}

func DiffFolders(old, new map[string]FileEntry) *SyncDiff {
	diff := &SyncDiff{}

	for relPath, newEntry := range new {
		oldEntry, exists := old[relPath]
		if !exists {
			diff.Adds = append(diff.Adds, newEntry)
		} else if oldEntry.Hash != newEntry.Hash {
			diff.Modifies = append(diff.Modifies, newEntry)
		}
	}

	for relPath := range old {
		if _, exists := new[relPath]; !exists {
			diff.Deletes = append(diff.Deletes, relPath)
		}
	}

	return diff
}
