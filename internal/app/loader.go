package app

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func loadVideosCmd(root, cachePath string, progress *loadProgress) tea.Cmd {
	return func() tea.Msg {
		cache, cacheErr := loadDurationCache(cachePath)
		videos, pending, err := loadVideos(root, cache, progress)
		if progress != nil {
			progress.MarkDone()
		}
		return videosLoadedMsg{videos: videos, err: err, cacheErr: cacheErr, pending: pending, cache: cache}
	}
}

func progressTickerCmd(progress *loadProgress) tea.Cmd {
	if progress == nil {
		return nil
	}
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		processed, total, done := progress.Snapshot()
		return progressUpdateMsg{processed: processed, total: total, done: done}
	})
}

func loadVideos(root string, cache *durationCache, progress *loadProgress) ([]video, []string, error) {
	paths, err := collectVideoPaths(root)
	if err != nil {
		return nil, nil, err
	}
	if progress != nil {
		progress.SetTotal(len(paths))
	}
	videos := make([]video, 0, len(paths))
	pending := make([]string, 0)
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			videos = append(videos, video{Name: filepath.Base(path), Path: path, Err: statErr})
			increment(progress)
			continue
		}
		dur := cachedDuration(cache, path, info)
		if dur == 0 {
			pending = append(pending, path)
		}
		videos = append(videos, video{
			Name:     filepath.Base(path),
			Path:     path,
			Duration: dur,
			ModTime:  info.ModTime(),
			Size:     info.Size(),
		})
		increment(progress)
	}
	return videos, pending, nil
}

func increment(progress *loadProgress) {
	if progress != nil {
		progress.Increment()
	}
}

func cachedDuration(cache *durationCache, path string, info os.FileInfo) time.Duration {
	if cache == nil {
		return 0
	}
	dur, ok := cache.Lookup(path, info)
	if !ok {
		return 0
	}
	return dur
}

func collectVideoPaths(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if isVideo(root) {
			return []string{root}, nil
		}
		return nil, nil
	}
	visited := make(map[string]struct{})
	var paths []string
	if err := traverseVideoPaths(root, root, visited, &paths); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func traverseVideoPaths(displayPath, realPath string, visited map[string]struct{}, acc *[]string) error {
	resolved, err := filepath.EvalSymlinks(realPath)
	if err != nil {
		resolved = realPath
	}
	resolved = filepath.Clean(resolved)
	if _, seen := visited[resolved]; seen {
		return nil
	}
	visited[resolved] = struct{}{}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		displayChild := filepath.Join(displayPath, entry.Name())
		realChild := filepath.Join(resolved, entry.Name())
		mode := entry.Type()
		var info os.FileInfo
		if mode == fs.FileMode(0) {
			info, err = entry.Info()
			if err != nil {
				return err
			}
			mode = info.Mode()
		}
		if mode&os.ModeSymlink != 0 {
			if err := handleSymlink(displayChild, realChild, visited, acc); err != nil {
				return err
			}
			continue
		}
		if mode.IsDir() {
			if err := traverseVideoPaths(displayChild, realChild, visited, acc); err != nil {
				return err
			}
			continue
		}
		if isVideo(displayChild) {
			*acc = append(*acc, displayChild)
		}
	}
	return nil
}

func handleSymlink(displayChild, realChild string, visited map[string]struct{}, acc *[]string) error {
	targetPath, err := filepath.EvalSymlinks(realChild)
	if err != nil {
		return recordIfVideo(displayChild, acc)
	}
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return recordIfVideo(displayChild, acc)
	}
	if targetInfo.IsDir() {
		return traverseVideoPaths(displayChild, targetPath, visited, acc)
	}
	if isVideo(displayChild) || isVideo(targetPath) {
		*acc = append(*acc, displayChild)
	}
	return nil
}

func recordIfVideo(path string, acc *[]string) error {
	if isVideo(path) {
		*acc = append(*acc, path)
	}
	return nil
}

func probeDurationsCmd(path string, cache *durationCache) tea.Cmd {
	return func() tea.Msg {
		dur, err := probeDuration(path)
		if err == nil && cache != nil {
			if info, statErr := os.Stat(path); statErr == nil {
				_ = cache.Record(path, info, dur)
			}
		}
		return durationUpdateMsg{path: path, duration: dur, err: err}
	}
}

func probeDuration(path string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return 0, errors.New("empty duration")
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func playVideoCmd(path, crop string) tea.Cmd {
	return func() tea.Msg {
		args := buildVLCArgs(path, crop)
		cmd := exec.Command("vlc", args...)
		if err := cmd.Start(); err != nil {
			return playVideoMsg{path: path, err: err}
		}
		go func() { _ = cmd.Wait() }()
		return playVideoMsg{path: path}
	}
}

func buildVLCArgs(path, crop string) []string {
	args := []string{}
	if crop != "" {
		args = append(args, "--crop", crop)
	}
	return append(args, path)
}

func isVideo(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := videoExtensions[ext]
	return ok
}

// CollectVideoPathsForTest exposes collectVideoPaths for unit testing.
func CollectVideoPathsForTest(root string) ([]string, error) {
	return collectVideoPaths(root)
}
