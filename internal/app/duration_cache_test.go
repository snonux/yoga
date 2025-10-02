package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDurationCacheRecordLifecycle(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	cache, err := loadDurationCache(cachePath)
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	video := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	info, err := os.Stat(video)
	if err != nil {
		t.Fatalf("stat video: %v", err)
	}
	duration := 90 * time.Second
	if err := cache.Record(video, info, duration); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := cache.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	cache2, err := loadDurationCache(cachePath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	dur, ok := cache2.Lookup(video, info)
	if !ok {
		t.Fatalf("expected cached entry")
	}
	if dur != duration {
		t.Fatalf("expected %v, got %v", duration, dur)
	}
}

func TestDurationCacheInvalidatesOnChange(t *testing.T) {
	dir := t.TempDir()
	cache := newDurationCache(filepath.Join(dir, "cache.json"))
	video := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, _ := os.Stat(video)
	_ = cache.Record(video, info, 30*time.Second)
	if err := os.WriteFile(video, []byte("xx"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	info, _ = os.Stat(video)
	if dur, ok := cache.Lookup(video, info); ok || dur != 0 {
		t.Fatalf("expected cache miss after change")
	}
}

func TestLoadDurationCacheInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	cache, err := loadDurationCache(path)
	if err == nil {
		t.Fatalf("expected error for invalid json")
	}
	if len(cache.entries) != 0 {
		t.Fatalf("expected cache to reset entries")
	}
}
