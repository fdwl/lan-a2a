package filetransfer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")

	data := make([]byte, 200*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(path, data, 0644)

	chunks, checksum, size, err := SplitFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if size != int64(len(data)) {
		t.Errorf("expected size %d, got %d", len(data), size)
	}

	expectedHash := sha256.Sum256(data)
	expectedChecksum := hex.EncodeToString(expectedHash[:])
	if checksum != expectedChecksum {
		t.Error("checksum mismatch")
	}

	if len(chunks) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(chunks))
	}

	var reassembled []byte
	totalSize := 0
	for _, c := range chunks {
		reassembled = append(reassembled, c...)
		totalSize += len(c)
	}
	if totalSize != len(data) {
		t.Errorf("reassembled size mismatch: %d vs %d", totalSize, len(data))
	}
}

func TestHashBytes(t *testing.T) {
	h := sha256.Sum256([]byte("hello"))
	hash := hex.EncodeToString(h[:])
	if len(hash) != 64 {
		t.Errorf("expected 64 char hex, got %d", len(hash))
	}
}
