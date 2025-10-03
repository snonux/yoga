package tags

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathForReplacesExtension(t *testing.T) {
	path := "/tmp/video.MP4"
	tagsPath := PathFor(path)
	if tagsPath != "/tmp/video.json" {
		t.Fatalf("expected json path, got %s", tagsPath)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	tags := []string{" calm ", "focus", "focus"}
	if err := Save(videoPath, tags); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(videoPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected sanitized tags, got %v", loaded)
	}
	if loaded[0] != "calm" || loaded[1] != "focus" {
		t.Fatalf("unexpected ordering: %v", loaded)
	}
}

func TestLoadMissingFile(t *testing.T) {
	tags, err := Load("/tmp/missing.mp4")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if tags != nil {
		t.Fatalf("expected nil tags for missing file, got %v", tags)
	}
}
