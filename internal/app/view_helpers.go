package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
)

func videoRow(v video) table.Row {
	duration := "(unknown)"
	if v.Duration > 0 {
		duration = formatDuration(v.Duration)
	}
	age := humanizeAge(v.ModTime)
	path := trimPath(v.Path)
	if v.Err != nil {
		duration = "!" + v.Err.Error()
	}
	return table.Row{v.Name, duration, age, path}
}

func renderProgressBar(done, total, width int) string {
	if width <= 0 || total <= 0 {
		return ""
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	filled := int(float64(done) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	return fmt.Sprintf("[%s]", bar)
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "--"
	}
	totalSeconds := int(d.Seconds() + 0.5)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func humanizeAge(t time.Time) string {
	if t.IsZero() {
		return "--"
	}
	dur := time.Since(t)
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm ago", int(dur.Minutes()))
	}
	if dur < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(dur.Hours()))
	}
	return t.Format("2006-01-02")
}

func trimPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
