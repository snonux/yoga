package thumbnail

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type entry struct {
	VideoPath string    `json:"video_path"`
	Thumbnail string    `json:"thumbnail"`
	ModTime   time.Time `json:"mod_time"`
	Timestamp time.Time `json:"timestamp"`
}

type Cache struct {
	entries map[string]entry
	mu      sync.RWMutex
	path    string
}

func NewCache(root string) *Cache {
	return newCache(root)
}

func newCache(root string) *Cache {
	path := filepath.Join(root, cacheFilename)
	return &Cache{
		entries: make(map[string]entry),
		path:    path,
	}
}

func (c *Cache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read cache file: %w", err)
	}

	var loaded []entry
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("unmarshal cache: %w", err)
	}

	c.entries = make(map[string]entry, len(loaded))
	for _, e := range loaded {
		c.entries[e.VideoPath] = e
	}

	return nil
}

func (c *Cache) Lookup(videoPath string, modTime time.Time) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.entries[videoPath]
	if !ok {
		return "", false
	}

	if !e.ModTime.Equal(modTime) {
		return "", false
	}

	if _, err := os.Stat(e.Thumbnail); err != nil {
		return "", false
	}

	return e.Thumbnail, true
}

func (c *Cache) Store(videoPath string, modTime time.Time, thumbnailPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[videoPath] = entry{
		VideoPath: videoPath,
		Thumbnail: thumbnailPath,
		ModTime:   modTime,
		Timestamp: time.Now(),
	}

	return c.flush()
}

func (c *Cache) Remove(videoPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, videoPath)
	return c.flush()
}

func (c *Cache) flush() error {
	loaded := make([]entry, 0, len(c.entries))
	for _, e := range c.entries {
		loaded = append(loaded, e)
	}

	data, err := json.MarshalIndent(loaded, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}
