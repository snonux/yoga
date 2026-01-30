package thumbnail

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Generator struct {
	ffmpegPath string
}

func NewGenerator() *Generator {
	return &Generator{
		ffmpegPath: "ffmpeg",
	}
}

func (g *Generator) Generate(ctx context.Context, videoPath string) (string, error) {
	duration, err := g.probeVideoDuration(ctx, videoPath)
	if err != nil {
		return "", fmt.Errorf("probe video duration: %w", err)
	}

	timestamp := duration * time.Duration(thumbnailPercent) / 100
	thumbnailPath := g.getThumbnailPath(videoPath)

	if err := g.extractFrame(ctx, videoPath, thumbnailPath, timestamp); err != nil {
		return "", fmt.Errorf("extract frame: %w", err)
	}

	return thumbnailPath, nil
}

func (g *Generator) probeVideoDuration(ctx context.Context, videoPath string) (time.Duration, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

func (g *Generator) extractFrame(ctx context.Context, videoPath, thumbnailPath string, timestamp time.Duration) error {
	thumbnailDir := filepath.Dir(thumbnailPath)
	if err := ensureDir(thumbnailDir); err != nil {
		return err
	}

	timestampSec := timestamp.Seconds()
	timestampStr := fmt.Sprintf("%.3f", timestampSec)

	args := []string{
		"-ss", timestampStr,
		"-i", videoPath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d", thumbnailWidth, thumbnailHeight),
		"-y",
		thumbnailPath,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) getThumbnailPath(videoPath string) string {
	dir := filepath.Dir(videoPath)
	thumbnailDir := filepath.Join(dir, thumbnailDir)

	filename := filepath.Base(videoPath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	return filepath.Join(thumbnailDir, name+"."+thumbnailFormat)
}

func ensureDir(path string) error {
	if _, err := exec.LookPath("mkdir"); err == nil {
		cmd := exec.Command("mkdir", "-p", path)
		return cmd.Run()
	}

	return fmt.Errorf("mkdir not found")
}
