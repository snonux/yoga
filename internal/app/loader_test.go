package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"yoga/internal/tags"
)

func TestCollectVideoPathsDetectsMP4(t *testing.T) {
	dir := t.TempDir()
	lower := filepath.Join(dir, "video.mp4")
	upper := filepath.Join(dir, "UPPER.MP4")
	for _, path := range []string{lower, upper} {
		if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	paths, err := CollectVideoPathsForTest(dir)
	if err != nil {
		t.Fatalf("collect paths: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	want := map[string]struct{}{lower: {}, upper: {}}
	for _, got := range paths {
		if _, ok := want[got]; !ok {
			t.Fatalf("unexpected path %s", got)
		}
	}
}

func TestCollectVideoPathsFollowsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := t.TempDir()
	storage := t.TempDir()
	video := filepath.Join(storage, "movie.mp4")
	if err := os.WriteFile(video, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	link := filepath.Join(root, "videos")
	if err := os.Symlink(storage, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	paths, err := CollectVideoPathsForTest(root)
	if err != nil {
		t.Fatalf("collect paths: %v", err)
	}
	expected := filepath.Join(link, "movie.mp4")
	if len(paths) != 1 || paths[0] != expected {
		t.Fatalf("expected %s, got %v", expected, paths)
	}
}

func TestLoadVideosWithCache(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(video, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cache := newDurationCache(filepath.Join(dir, "cache.json"))
	info, err := os.Stat(video)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	_ = cache.Record(video, info, time.Minute)
	progress := &loadProgress{}
	progress.Reset()
	videos, pending, tagErr, err := loadVideos(dir, cache, progress)
	if err != nil {
		t.Fatalf("loadVideos: %v", err)
	}
	if tagErr != nil {
		t.Fatalf("unexpected tag error: %v", tagErr)
	}
	if len(videos) != 1 || len(pending) != 0 {
		t.Fatalf("expected cached video without pending: videos=%d pending=%d", len(videos), len(pending))
	}
	if videos[0].Duration != time.Minute {
		t.Fatalf("expected cached duration")
	}
}

func TestLoadVideosReadsTags(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "session.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	metaPath := tags.PathFor(videoPath)
	if err := os.WriteFile(metaPath, []byte("[\"calm\", \"focus\"]"), 0o644); err != nil {
		t.Fatalf("write tags: %v", err)
	}
	videos, _, tagErr, err := loadVideos(dir, nil, nil)
	if err != nil {
		t.Fatalf("loadVideos: %v", err)
	}
	if tagErr != nil {
		t.Fatalf("unexpected tag error: %v", tagErr)
	}
	if len(videos) != 1 || len(videos[0].Tags) != 2 {
		t.Fatalf("expected tags loaded, got %#v", videos)
	}
}

func TestProbeDurationsCmdHandlesMissingBinary(t *testing.T) {
	cmd := probeDurationsCmd("/no/such/file.mp4", nil)
	msg := cmd()
	update := msg.(durationUpdateMsg)
	if update.err == nil {
		t.Fatalf("expected error from ffprobe")
	}
}

func TestProbeDurationSuccess(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ffprobe")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 5\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+oldPath)
	dur, err := probeDuration("dummy.mp4")
	if err != nil {
		t.Fatalf("probeDuration: %v", err)
	}
	if dur != 5*time.Second {
		t.Fatalf("expected 5s duration, got %v", dur)
	}
}

func TestPlayVideoCmdMissingBinary(t *testing.T) {
	cmd := playVideoCmd("/no/such/file.mp4", "")
	msg := cmd()
	result := msg.(playVideoMsg)
	if result.path != "/no/such/file.mp4" {
		t.Fatalf("unexpected path %s", result.path)
	}
}

func TestRecordIfVideo(t *testing.T) {
	var acc []string
	if err := recordIfVideo("test.mp4", &acc); err != nil {
		t.Fatalf("recordIfVideo: %v", err)
	}
	if len(acc) != 1 {
		t.Fatalf("expected video recorded")
	}
}

func TestHandleSymlinkBrokenVideo(t *testing.T) {
	dir := t.TempDir()
	symlink := filepath.Join(dir, "clip.mp4")
	target := filepath.Join(dir, "missing.mp4")
	if err := os.Symlink(target, symlink); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var acc []string
	if err := handleSymlink(symlink, symlink, map[string]struct{}{}, &acc); err != nil {
		t.Fatalf("handleSymlink: %v", err)
	}
	if len(acc) != 1 {
		t.Fatalf("expected symlink video recorded")
	}
}

func TestLoadVideosHandlesStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.mp4")
	if err := os.Symlink(filepath.Join(dir, "missing.mp4"), broken); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	videos, _, tagErr, err := loadVideos(dir, nil, nil)
	if err != nil {
		t.Fatalf("loadVideos: %v", err)
	}
	if tagErr != nil {
		t.Fatalf("unexpected tag error: %v", tagErr)
	}
	if len(videos) != 1 || videos[0].Err == nil {
		t.Fatalf("expected stat error recorded, got %+v", videos)
	}
}
