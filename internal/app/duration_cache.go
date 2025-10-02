package app

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"
)

type cacheEntry struct {
	DurationSeconds float64 `json:"duration_seconds"`
	ModTimeUnix     int64   `json:"mod_time_unix"`
	Size            int64   `json:"size"`
}

type durationCache struct {
	path    string
	entries map[string]cacheEntry
	mu      sync.Mutex
	dirty   bool
}

func newDurationCache(path string) *durationCache {
	return &durationCache{path: path, entries: make(map[string]cacheEntry)}
}

func loadDurationCache(path string) (*durationCache, error) {
	cache := newDurationCache(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache.entries); err != nil {
		cache.entries = make(map[string]cacheEntry)
		return cache, err
	}
	return cache, nil
}

func (c *durationCache) Lookup(path string, info os.FileInfo) (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[path]
	if !ok {
		return 0, false
	}
	if entry.ModTimeUnix != info.ModTime().Unix() || entry.Size != info.Size() {
		delete(c.entries, path)
		c.dirty = true
		return 0, false
	}
	if entry.DurationSeconds <= 0 {
		return 0, false
	}
	return time.Duration(entry.DurationSeconds * float64(time.Second)), true
}

func (c *durationCache) Record(path string, info os.FileInfo, dur time.Duration) error {
	if c == nil || dur <= 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]cacheEntry)
	}
	c.entries[path] = cacheEntry{
		DurationSeconds: dur.Seconds(),
		ModTimeUnix:     info.ModTime().Unix(),
		Size:            info.Size(),
	}
	c.dirty = true
	return nil
}

func (c *durationCache) Flush() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if !c.dirty {
		c.mu.Unlock()
		return nil
	}
	snapshot := make(map[string]cacheEntry, len(c.entries))
	for k, v := range c.entries {
		snapshot[k] = v
	}
	c.dirty = false
	c.mu.Unlock()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}
