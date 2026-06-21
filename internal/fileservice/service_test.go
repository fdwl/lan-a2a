package fileservice

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSplitIntoChunks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "split_test.bin")

	data := make([]byte, 150*1024) // 150KB
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(path, data, 0644)

	chunks, checksum, size, err := SplitIntoChunks(path, 64*1024)
	if err != nil {
		t.Fatal(err)
	}

	if size != 150*1024 {
		t.Errorf("expected size 153600, got %d", size)
	}
	if checksum == "" {
		t.Error("checksum should not be empty")
	}
	// 150KB / 64KB = 3 chunks
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(chunks))
	}

	for i, c := range chunks {
		if c.Index != i {
			t.Errorf("chunk %d: wrong index %d", i, c.Index)
		}
		if c.Hash == "" {
			t.Errorf("chunk %d: empty hash", i)
		}
	}
}

func TestReadChunk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "read_test.bin")
	data := []byte("0123456789ABCDEF")
	os.WriteFile(path, data, 0644)

	chunk := &Chunk{Offset: 4, Size: 6}
	readData, err := ReadChunk(path, chunk)
	if err != nil {
		t.Fatal(err)
	}
	if string(readData) != "456789" {
		t.Errorf("expected '456789', got '%s'", string(readData))
	}
}

func TestSendFileConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent_test.bin")
	data := make([]byte, 256*1024) // 256KB
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(path, data, 0644)

	var mu sync.Mutex
	received := make(map[int][]byte)

	fs := NewFileService(func(peerID string, chunkData []byte, transfer *Transfer, chunk *Chunk) error {
		mu.Lock()
		received[chunk.Index] = chunkData
		mu.Unlock()
		return nil
	}, 4)

	t.Log("Starting concurrent send...")
	trans, err := fs.SendFile("peer-1", path)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for completion
	for i := 0; i < 100; i++ {
		if trans.Status == StatusCompleted || trans.Status == StatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if trans.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", trans.Status)
	}

	mu.Lock()
	if len(received) != trans.TotalChunks {
		t.Errorf("expected %d chunks received, got %d", trans.TotalChunks, len(received))
	}
	mu.Unlock()
}

func TestWalkFolder(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("bbb"), 0644)

	entries, err := WalkFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 files, got %d", len(entries))
	}
	if _, ok := entries["a.txt"]; !ok {
		t.Error("a.txt not found")
	}
	if _, ok := entries["sub/b.txt"]; !ok {
		t.Error("sub/b.txt not found")
	}
}

func TestDiffFolders(t *testing.T) {
	old := map[string]FileEntry{
		"a.txt": {RelPath: "a.txt", Hash: "hash1", Size: 10},
		"b.txt": {RelPath: "b.txt", Hash: "hash2", Size: 20},
		"c.txt": {RelPath: "c.txt", Hash: "hash3", Size: 30},
	}
	new := map[string]FileEntry{
		"a.txt": {RelPath: "a.txt", Hash: "hash1", Size: 10},  // unchanged
		"b.txt": {RelPath: "b.txt", Hash: "hash_new", Size: 25}, // modified
		"d.txt": {RelPath: "d.txt", Hash: "hash4", Size: 40},  // new
	}
	// c.txt is deleted

	diff := DiffFolders(old, new)

	if len(diff.Adds) != 1 || diff.Adds[0].RelPath != "d.txt" {
		t.Errorf("expected 1 add (d.txt), got %d", len(diff.Adds))
	}
	if len(diff.Modifies) != 1 || diff.Modifies[0].RelPath != "b.txt" {
		t.Errorf("expected 1 modify (b.txt), got %d", len(diff.Modifies))
	}
	if len(diff.Deletes) != 1 || diff.Deletes[0] != "c.txt" {
		t.Errorf("expected 1 delete (c.txt), got %d", len(diff.Deletes))
	}
}
