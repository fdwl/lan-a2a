package fileservice

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fdwl/lan-a2a/internal/plugins"
)

type FolderSync struct {
	manifests map[string]*FolderManifest
	mu        sync.RWMutex
	fs        *FileService
	plugins   *plugins.Manager
}

func NewFolderSync(fs *FileService) *FolderSync {
	return &FolderSync{
		manifests: make(map[string]*FolderManifest),
		fs:        fs,
		plugins:   fs.Plugins(),
	}
}

// ScanFolder scans a local folder and returns its manifest.
func (fsync *FolderSync) ScanFolder(root string) (*FolderManifest, error) {
	entries, err := WalkFolder(root)
	if err != nil {
		return nil, err
	}

	manifest := &FolderManifest{
		Root:      root,
		Files:     entries,
		UpdatedAt: time.Now(),
	}

	fsync.mu.Lock()
	fsync.manifests[root] = manifest
	fsync.mu.Unlock()

	return manifest, nil
}

// SaveManifest persists a manifest to disk for later comparison.
func (fsync *FolderSync) SaveManifest(root string) error {
	fsync.mu.RLock()
	m, ok := fsync.manifests[root]
	fsync.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no manifest for %s", root)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(root, ".lan-sync-manifest.json")
	return os.WriteFile(manifestPath, data, 0644)
}

// LoadManifest loads a previously saved manifest from disk.
func (fsync *FolderSync) LoadManifest(root string) (*FolderManifest, error) {
	manifestPath := filepath.Join(root, ".lan-sync-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var m FolderManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	fsync.mu.Lock()
	fsync.manifests[root] = &m
	fsync.mu.Unlock()
	return &m, nil
}

// DiffWith compares current folder state against the last known manifest.
// Returns what needs to be synced.
func (fsync *FolderSync) DiffWith(root string) (*SyncDiff, error) {
	current, err := fsync.ScanFolder(root)
	if err != nil {
		return nil, err
	}

	// Try loading previous manifest
	var old *FolderManifest
	old, err = fsync.LoadManifest(root)
	if err != nil {
		// No previous manifest → everything is new
		old = &FolderManifest{Files: make(map[string]FileEntry)}
	}

	diff := DiffFolders(old.Files, current.Files)
	diff.Root = root
	return diff, nil
}

// SyncFolder scans, diffs, and schedules transfers for changed files.
func (fsync *FolderSync) SyncFolder(root, peerID string) (*SyncResult, error) {
	diff, err := fsync.DiffWith(root)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{
		Diff:      diff,
		StartedAt: time.Now(),
	}

	fsync.plugins.Emit(&plugins.Event{
		Type:    plugins.EventFolderSyncStart,
		Folder:  root,
		PeerID:  peerID,
		Diff:    diff,
	})

	// Send added and modified files
	allEntries := append(diff.Adds, diff.Modifies...)
	for _, entry := range allEntries {
		t, err := fsync.fs.SendFile(peerID, entry.Path)
		if err != nil {
			log.Printf("[sync] send %s failed: %v", entry.RelPath, err)
			continue
		}
		result.Transfers = append(result.Transfers, t.ID)
	}

	// Save current manifest after sync
	fsync.SaveManifest(root)

	fsync.plugins.Emit(&plugins.Event{
		Type:    plugins.EventFolderSyncDone,
		Folder:  root,
		PeerID:  peerID,
		Result:  result,
	})

	return result, nil
}

// GetStatus returns sync status for a folder.
func (fsync *FolderSync) GetStatus(root string) map[string]interface{} {
	fsync.mu.RLock()
	m, ok := fsync.manifests[root]
	fsync.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"status": "unknown", "root": root}
	}

	var totalSize int64
	for _, e := range m.Files {
		totalSize += e.Size
	}

	return map[string]interface{}{
		"status":    "tracked",
		"root":      root,
		"files":     len(m.Files),
		"total_size": totalSize,
		"last_scan": m.UpdatedAt,
	}
}
