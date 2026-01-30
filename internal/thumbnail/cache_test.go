package thumbnail

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if g.ffmpegPath != "ffmpeg" {
		t.Errorf("expected ffmpegPath to be 'ffmpeg', got '%s'", g.ffmpegPath)
	}
}

func TestGeneratorGetThumbnailPath(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		videoPath      string
		expectedSuffix string
	}{
		{
			videoPath:      "/home/user/video.mp4",
			expectedSuffix: filepath.Join(".thumbnails", "video.jpg"),
		},
		{
			videoPath:      "/home/user/yoga/morning.mp4",
			expectedSuffix: filepath.Join(".thumbnails", "morning.jpg"),
		},
		{
			videoPath:      "/home/user/yoga/evening.mkv",
			expectedSuffix: filepath.Join(".thumbnails", "evening.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.videoPath, func(t *testing.T) {
			result := g.getThumbnailPath(tt.videoPath)
			if !strings.HasSuffix(result, tt.expectedSuffix) {
				t.Errorf("expected path to end with '%s', got '%s'", tt.expectedSuffix, result)
			}
		})
	}
}

func TestCacheNewCache(t *testing.T) {
	c := newCache("/tmp/test")
	if c == nil {
		t.Fatal("newCache returned nil")
	}
	if c.path != filepath.Join("/tmp/test", cacheFilename) {
		t.Errorf("expected path to be '%s', got '%s'", filepath.Join("/tmp/test", cacheFilename), c.path)
	}
	if c.entries == nil {
		t.Fatal("expected entries to be initialized")
	}
}

func TestCacheLoadNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	c := newCache(tmpDir)

	err := c.Load()
	if err != nil {
		t.Errorf("expected no error for non-existent cache file, got %v", err)
	}
}

func TestCacheStoreAndLookup(t *testing.T) {
	tmpDir := t.TempDir()
	c := newCache(tmpDir)

	videoPath := "/test/video.mp4"
	modTime := time.Now()
	thumbnailPath := filepath.Join(tmpDir, ".thumbnails", "video.jpg")

	err := c.Store(videoPath, modTime, thumbnailPath)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(thumbnailPath), 0o755); err != nil {
		t.Fatalf("failed to create thumbnail directory: %v", err)
	}
	if err := os.WriteFile(thumbnailPath, []byte("fake thumbnail"), 0o644); err != nil {
		t.Fatalf("failed to create thumbnail file: %v", err)
	}

	retrieved, ok := c.Lookup(videoPath, modTime)
	if !ok {
		t.Fatal("Lookup returned false for stored entry")
	}
	if retrieved != thumbnailPath {
		t.Errorf("expected thumbnailPath '%s', got '%s'", thumbnailPath, retrieved)
	}
}

func TestCacheLookupDifferentModTime(t *testing.T) {
	tmpDir := t.TempDir()
	c := newCache(tmpDir)

	videoPath := "/test/video.mp4"
	modTime1 := time.Now()
	thumbnailPath := "/test/.thumbnails/video.jpg"

	err := c.Store(videoPath, modTime1, thumbnailPath)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	modTime2 := modTime1.Add(1 * time.Hour)
	_, ok := c.Lookup(videoPath, modTime2)
	if ok {
		t.Error("expected Lookup to return false for different mod time")
	}
}

func TestCacheRemove(t *testing.T) {
	tmpDir := t.TempDir()
	c := newCache(tmpDir)

	videoPath := "/test/video.mp4"
	modTime := time.Now()
	thumbnailPath := "/test/.thumbnails/video.jpg"

	err := c.Store(videoPath, modTime, thumbnailPath)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	err = c.Remove(videoPath)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, ok := c.Lookup(videoPath, modTime)
	if ok {
		t.Error("expected Lookup to return false after Remove")
	}
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, cacheFilename)

	videoPath := "/test/video.mp4"
	modTime := time.Now()
	thumbnailPath := filepath.Join(tmpDir, ".thumbnails", "video.jpg")

	if err := os.MkdirAll(filepath.Dir(thumbnailPath), 0o755); err != nil {
		t.Fatalf("failed to create thumbnail directory: %v", err)
	}
	if err := os.WriteFile(thumbnailPath, []byte("fake thumbnail"), 0o644); err != nil {
		t.Fatalf("failed to create thumbnail file: %v", err)
	}

	c1 := newCache(tmpDir)
	err := c1.Store(videoPath, modTime, thumbnailPath)
	if err != nil {
		t.Fatalf("first Store failed: %v", err)
	}

	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	c2 := newCache(tmpDir)
	err = c2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	retrieved, ok := c2.Lookup(videoPath, modTime)
	if !ok {
		t.Fatal("Lookup returned false for loaded entry")
	}
	if retrieved != thumbnailPath {
		t.Errorf("expected thumbnailPath '%s', got '%s'", thumbnailPath, retrieved)
	}
}

func TestGenerateWithMissingFFmpeg(t *testing.T) {
	g := &Generator{ffmpegPath: "nonexistent-ffmpeg-binary"}

	ctx := context.Background()
	_, err := g.Generate(ctx, "/test/video.mp4")
	if err == nil {
		t.Error("expected error for missing ffmpeg")
	}
}
