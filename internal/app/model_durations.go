package app

import (
	"fmt"
	"path/filepath"
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



