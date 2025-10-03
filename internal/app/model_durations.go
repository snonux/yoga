package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleDurationUpdate(msg durationUpdateMsg) (tea.Model, tea.Cmd) {
	if msg.path != "" {
		m.updateVideoDuration(msg.path, msg.duration, msg.err)
		m.durationDone++
		m.updateStatusForDuration(msg)
	}
	if m.durationInFlight > 0 {
		m.durationInFlight--
	}
	selectedPath := m.currentSelectionPath()
	m.applyFiltersAndSort()
	m.restoreSelection(selectedPath)
	if m.allDurationsResolved() {
		m.onDurationsComplete()
		return m, nil
	}
	cmd := m.dequeueDurationCmd()
	return m, cmd
}

func (m *model) updateStatusForDuration(msg durationUpdateMsg) {
	if msg.err != nil {
		m.statusMessage = fmt.Sprintf("Duration error for %s: %v", filepath.Base(msg.path), msg.err)
		return
	}
	if m.durationTotal > 0 {
		m.statusMessage = fmt.Sprintf("Probing durations %d/%d...", m.durationDone, m.durationTotal)
	}
}

func (m model) currentSelectionPath() string {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.filtered) {
		return ""
	}
	return m.filtered[idx].Path
}

func (m *model) restoreSelection(path string) {
	if path == "" {
		return
	}
	for i, video := range m.filtered {
		if video.Path == path {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m *model) updateVideoDuration(path string, dur time.Duration, err error) {
	for i := range m.videos {
		if m.videos[i].Path != path {
			continue
		}
		m.videos[i].Duration = dur
		m.videos[i].Err = err
		return
	}
}

func (m model) allDurationsResolved() bool {
	return m.durationDone >= m.durationTotal && m.durationInFlight == 0
}

func (m *model) onDurationsComplete() {
	if m.cache != nil {
		if err := m.cache.Flush(); err != nil {
			m.statusMessage = fmt.Sprintf("Duration cache flush error: %v", err)
		} else {
			m.statusMessage = fmt.Sprintf("Durations ready (%d videos)", len(m.filtered))
		}
		m.resetDurationState()
		return
	}
	m.statusMessage = fmt.Sprintf("Durations ready (%d videos)", len(m.filtered))
	m.resetDurationState()
}

func (m *model) resetDurationState() {
	m.pendingDurations = nil
	m.durationTotal = 0
	m.durationDone = 0
	m.durationInFlight = 0
}

func (m *model) dequeueDurationCmd() tea.Cmd {
	if len(m.pendingDurations) == 0 {
		return nil
	}
	path := m.pendingDurations[0]
	m.pendingDurations = m.pendingDurations[1:]
	m.durationInFlight++
	return probeDurationsCmd(path, m.cache)
}

func (m *model) startDurationWorkers() tea.Cmd {
	if len(m.pendingDurations) == 0 {
		return nil
	}
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > 6 {
		workers = 6
	}
	if workers > len(m.pendingDurations) {
		workers = len(m.pendingDurations)
	}
	cmds := make([]tea.Cmd, 0, workers)
	for i := 0; i < workers; i++ {
		cmd := m.dequeueDurationCmd()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m model) activeCrop() string {
	if m.cropEnabled && m.cropValue != "" {
		return m.cropValue
	}
	return ""
}

func (m model) handleVideosLoaded(msg videosLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		m.statusMessage = fmt.Sprintf("error: %v", msg.err)
	}
	m.videos = msg.videos
	m.cache = msg.cache
	m.pendingDurations = msg.pending
	m.durationTotal = len(msg.pending)
	m.durationDone = 0
	m.applyFiltersAndSort()
	m.updateStatusAfterLoad(msg)
	m.durationInFlight = 0
	if len(msg.pending) == 0 {
		return m, nil
	}
	cmd := m.startDurationWorkers()
	return m, cmd
}

func (m *model) updateStatusAfterLoad(msg videosLoadedMsg) {
	if len(m.filtered) == 0 {
		m.baseStatus = "No videos found"
		m.statusMessage = m.baseStatus
		return
	}
	status := ""
	if len(msg.pending) > 0 {
		status = fmt.Sprintf("Loaded %d videos, probing durations...", len(m.filtered))
		if msg.cacheErr != nil {
			status = fmt.Sprintf("Loaded %d videos (cache warning: %v), probing durations...", len(m.filtered), msg.cacheErr)
		}
	} else {
		status = fmt.Sprintf("Loaded %d videos", len(m.filtered))
		if msg.cacheErr != nil {
			status = fmt.Sprintf("Loaded %d videos (cache warning: %v)", len(m.filtered), msg.cacheErr)
		}
	}
	if msg.tagErr != nil {
		status = fmt.Sprintf("%s (tag warning: %v)", status, msg.tagErr)
	}
	m.baseStatus = status
	m.statusMessage = status
}
