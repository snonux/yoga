package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadVideosDetectsMP4(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(videoPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create test video: %v", err)
	}
	upperPath := filepath.Join(dir, "UPPER.MP4")
	if err := os.WriteFile(upperPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create upper test video: %v", err)
	}

	vids, pending, err := loadVideos(dir, nil, nil)
	if err != nil {
		t.Fatalf("loadVideos returned error: %v", err)
	}
	if len(vids) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(vids))
	}
	paths := map[string]bool{videoPath: false, upperPath: false}
	for _, v := range vids {
		if _, ok := paths[v.Path]; ok {
			paths[v.Path] = true
		}
	}
	for p, seen := range paths {
		if !seen {
			t.Fatalf("missing video %s", p)
		}
	}
	if len(pending) != 2 {
		t.Fatalf("expected pending durations for both videos, got %d", len(pending))
	}
}

func TestLoadVideosFollowSymlinkDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}

	root := t.TempDir()
	storage := t.TempDir()

	if err := os.WriteFile(filepath.Join(storage, "movie.mp4"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create storage video: %v", err)
	}

	linkPath := filepath.Join(root, "videos")
	if err := os.Symlink(storage, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	vids, _, err := loadVideos(root, nil, nil)
	if err != nil {
		t.Fatalf("loadVideos returned error: %v", err)
	}

	expected := filepath.Join(linkPath, "movie.mp4")
	found := false
	for _, v := range vids {
		if v.Path == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find video at %s, paths=%v", expected, vids)
	}
}

func TestResolveRootPathDefaultCreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got, err := resolveRootPath("")
	if err != nil {
		t.Fatalf("resolveRootPath returned error: %v", err)
	}
	want := filepath.Join(tmp, "Yoga")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat expected dir failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", want)
	}
}

func TestResolveRootPathRequiresExistingDirectory(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing")
	if _, err := resolveRootPath(missing); err == nil {
		t.Fatalf("expected error for missing path %s", missing)
	}
}

func TestExpandPathHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	got, err := expandPath("~/custom")
	if err != nil {
		t.Fatalf("expandPath error: %v", err)
	}
	want := filepath.Join(tmp, "custom")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
