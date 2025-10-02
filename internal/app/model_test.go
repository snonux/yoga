package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelHandleVideosLoadedAndSort(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	videos := []video{
		{Name: "B.mp4", Path: filepath.Join(root, "B.mp4"), Duration: time.Minute, ModTime: time.Now()},
		{Name: "A.mp4", Path: filepath.Join(root, "A.mp4"), Duration: 2 * time.Minute, ModTime: time.Now().Add(-time.Hour)},
	}
	msg := videosLoadedMsg{videos: videos, pending: nil, cache: newDurationCache(filepath.Join(root, "cache.json"))}
	modelAny, cmd := m.handleVideosLoaded(msg)
	if cmd != nil {
		t.Fatalf("expected no duration command")
	}
	m = modelAny.(model)
	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "A.mp4" {
		t.Fatalf("expected sorted by name ascending")
	}
	modelAny, _ = m.handleKeyMsg(keyMsg("l"))
	m = modelAny.(model)
	if m.filtered[0].Name != "B.mp4" {
		t.Fatalf("expected shortest duration first")
	}
}

func TestModelHandleDurationUpdateCompletes(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	pendingPath := filepath.Join(root, "pending.mp4")
	videos := []video{{Name: "pending.mp4", Path: pendingPath}}
	msg := videosLoadedMsg{videos: videos, pending: []string{pendingPath}, cache: newDurationCache(filepath.Join(root, "cache.json"))}
	modelAny, cmd := m.handleVideosLoaded(msg)
	if cmd == nil {
		t.Fatalf("expected duration command")
	}
	m = modelAny.(model)
	durMsg := durationUpdateMsg{path: pendingPath, duration: time.Minute}
	modelAny, next := m.handleDurationUpdate(durMsg)
	m = modelAny.(model)
	if next != nil {
		t.Fatalf("expected no further command after completion")
	}
	if m.durationDone != 0 || m.pendingDurations != nil {
		t.Fatalf("expected duration queue cleared")
	}
}

func TestModelFiltersWorkflow(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	videos := []video{{Name: "morning flow.mp4", Path: filepath.Join(root, "morning.mp4"), Duration: 10 * time.Minute}}
	modelAny, _ := m.handleVideosLoaded(videosLoadedMsg{videos: videos, cache: newDurationCache(filepath.Join(root, "cache.json"))})
	m = modelAny.(model)
	modelAny, _ = m.handleKeyMsg(keyMsg("/"))
	m = modelAny.(model)
	if !m.showFilters {
		t.Fatalf("expected filters to open")
	}
	m.inputs.fields[0].SetValue("morning")
	inputs, _ := m.updateFilterInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m.inputs = inputs
	m.inputs.fields[0].SetValue("morning")
	if modal := m.renderFilterModal(); !strings.Contains(modal, "Filter videos") {
		t.Fatalf("expected filter modal content")
	}
	modelAny, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = modelAny.(model)
	if len(m.filtered) != 1 {
		t.Fatalf("expected filtered result")
	}
}

func TestRenderProgressBar(t *testing.T) {
	bar := renderProgressBar(5, 10, 10)
	if bar != "[#####-----]" {
		t.Fatalf("unexpected bar %s", bar)
	}
	if renderProgressBar(0, 0, 10) != "" {
		t.Fatalf("expected empty bar for zero total")
	}
	if renderProgressBar(-1, 10, 5) == "" {
		t.Fatalf("expected bar even when done negative")
	}
}

func TestModelViewAndProgress(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	m.statusMessage = "Ready"
	m.filtered = []video{}
	view := m.View()
	if view == "" {
		t.Fatalf("expected non-empty view")
	}
	m.durationTotal = 10
	m.durationDone = 5
	if line := m.renderProgressLine(); !strings.Contains(line, "5/10") {
		t.Fatalf("unexpected progress line %s", line)
	}
}

func TestModelInitAndUpdate(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}
	m.loading = false
	modelAny, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = modelAny.(model)
	if m.filters.name != "" || !strings.Contains(m.statusMessage, "Filters cleared") {
		t.Fatalf("expected filters reset via update path")
	}
	m.loading = true
	modelAny, cmd = m.Update(progressUpdateMsg{processed: 1, total: 2, done: false})
	m = modelAny.(model)
	if cmd == nil || !strings.Contains(m.statusMessage, "Loading videos") {
		t.Fatalf("unexpected status %s", m.statusMessage)
	}
}

func TestHandlePlayVideoStatuses(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m = m.handlePlayVideo(playVideoMsg{path: "video.mp4", err: errors.New("fail")})
	if !strings.Contains(m.statusMessage, "Failed") {
		t.Fatalf("expected failure message")
	}
	m = m.handlePlayVideo(playVideoMsg{path: "video.mp4"})
	if !strings.Contains(m.statusMessage, "Playing") {
		t.Fatalf("expected playing message")
	}
}

func TestDescribeFilters(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.filters = filterState{name: "flow", minEnabled: true, minMinutes: 5, maxEnabled: true, maxMinutes: 20}
	desc := m.describeFilters()
	if !strings.Contains(desc, "flow") || !strings.Contains(desc, ">=5") {
		t.Fatalf("unexpected description %s", desc)
	}
}

func TestPlaySelectionCommand(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	m.filtered = []video{{Name: "clip", Path: "clip.mp4"}}
	cmdModel, cmd := m.playSelection()
	if cmd == nil {
		t.Fatalf("expected command to play video")
	}
	if cmdModel.(model).statusMessage == "" {
		t.Fatalf("expected status message set")
	}
}

func TestUpdateTableFallback(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
}

func TestProgressTickerNil(t *testing.T) {
	if progressTickerCmd(nil) != nil {
		t.Fatalf("expected nil command for nil progress")
	}
}

func TestHandleFilterKeyTabs(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.showFilters = true
	m.inputs = buildFilterInputs()
	m.inputs.fields[0].Focus()
	modelAny, _ := m.handleFilterKey(tea.KeyMsg{Type: tea.KeyTab})
	m = modelAny.(model)
	if !m.inputs.fields[1].Focused() {
		t.Fatalf("expected focus to move forward")
	}
	modelAny, _ = m.handleFilterKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = modelAny.(model)
	if !m.inputs.fields[0].Focused() {
		t.Fatalf("expected focus to move back")
	}
}

func TestUpdateStatusAfterLoadBranches(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	videos := []video{{Name: "a", Path: "a.mp4"}}
	msg := videosLoadedMsg{videos: videos, pending: []string{"a.mp4"}, cacheErr: errors.New("cache"), cache: newDurationCache("cache.json")}
	modelAny, cmd := m.handleVideosLoaded(msg)
	m = modelAny.(model)
	if cmd == nil {
		t.Fatalf("expected pending duration command")
	}
	if !strings.Contains(m.statusMessage, "cache warning") {
		t.Fatalf("expected cache warning status")
	}
}

func TestModelUpdateWithVideosLoaded(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	videos := []video{{Name: "a", Path: "a.mp4"}}
	msg := videosLoadedMsg{videos: videos}
	modelAny, _ := m.Update(msg)
	m = modelAny.(model)
	if len(m.videos) != 1 {
		t.Fatalf("expected videos loaded")
	}
}

func TestModelUpdatePlayVideoMsg(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m = m.handlePlayVideo(playVideoMsg{path: "a.mp4"})
	modelAny, _ := m.Update(playVideoMsg{path: "a.mp4"})
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "Playing") {
		t.Fatalf("expected playing status, got %s", m.statusMessage)
	}
}

func TestModelUpdateDurationMsg(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.videos = []video{{Name: "a", Path: "a.mp4"}}
	m.filtered = m.videos
	m.pendingDurations = []string{"a.mp4"}
	m.durationTotal = 1
	m.cache = newDurationCache("cache.json")
	update := durationUpdateMsg{path: "a.mp4", duration: time.Second}
	modelAny, _ := m.Update(update)
	m = modelAny.(model)
	if m.durationTotal != 0 {
		t.Fatalf("expected duration queue cleared")
	}
}

func TestUpdateStatusForDurationError(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.videos = []video{{Name: "a", Path: "a.mp4"}, {Name: "b", Path: "b.mp4"}}
	m.filtered = m.videos
	m.pendingDurations = []string{"a.mp4", "b.mp4"}
	m.durationTotal = 2
	m.durationInFlight = 1
	m.cache = newDurationCache("cache.json")
	msg := durationUpdateMsg{path: "a.mp4", err: errors.New("ffprobe")}
	modelAny, _ := m.Update(msg)
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "Duration error") {
		t.Fatalf("expected error status, got %s", m.statusMessage)
	}
}

func TestHandleProgressUpdateDone(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = true
	modelAny, _ := m.handleProgressUpdate(progressUpdateMsg{total: 2, done: true})
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "Loaded") {
		t.Fatalf("expected loaded status, got %s", m.statusMessage)
	}
}

func TestUpdateStatusAfterLoadCacheWarning(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	msg := videosLoadedMsg{videos: []video{{Name: "a", Path: "a.mp4"}}, cacheErr: errors.New("oops"), cache: newDurationCache("cache.json")}
	modelAny, _ := m.Update(msg)
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "cache warning") {
		t.Fatalf("expected cache warning, got %s", m.statusMessage)
	}
}

func TestPassesFiltersBounds(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.filters = filterState{minEnabled: true, minMinutes: 5, maxEnabled: true, maxMinutes: 15}
	video := video{Name: "clip", Duration: 10 * time.Minute}
	if !m.passesFilters(video) {
		t.Fatalf("expected video within bounds")
	}
	m.filters.maxMinutes = 5
	if m.passesFilters(video) {
		t.Fatalf("expected video to fail with tighter max")
	}
	m.filters = filterState{name: "yoga"}
	if m.passesFilters(video) {
		t.Fatalf("expected name filter to exclude video")
	}
}

func TestProgressTickerCmdTick(t *testing.T) {
	progress := &loadProgress{}
	progress.SetTotal(2)
	cmd := progressTickerCmd(progress)
	if cmd == nil {
		t.Fatalf("expected command")
	}
	msg := cmd().(progressUpdateMsg)
	if msg.total != 2 {
		t.Fatalf("unexpected ticker message %#v", msg)
	}
}

func TestProgressUpdateMessages(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	modelAny, cmd := m.handleProgressUpdate(progressUpdateMsg{processed: 1, total: 3, done: false})
	if cmd == nil {
		t.Fatalf("expected ticker command")
	}
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "Loading videos") {
		t.Fatalf("unexpected status %s", m.statusMessage)
	}
	modelAny, cmd = m.handleProgressUpdate(progressUpdateMsg{total: 0, done: true})
	m = modelAny.(model)
	if cmd != nil || m.statusMessage != "No videos found" {
		t.Fatalf("expected no videos message")
	}
}

func TestToggleCrop(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir(), Crop: "5:4"})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	modelAny, _ := m.handleKeyMsg(keyMsg("c"))
	m = modelAny.(model)
	if m.statusMessage != "Crop disabled" {
		t.Fatalf("expected crop disabled, got %s", m.statusMessage)
	}
	modelAny, _ = m.handleKeyMsg(keyMsg("c"))
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "Crop enabled") {
		t.Fatalf("expected crop enabled, got %s", m.statusMessage)
	}
	if m.activeCrop() == "" {
		t.Fatalf("expected active crop")
	}
}

func TestToggleSort(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.toggleSort(sortByDuration)
	if m.sortField != sortByDuration || !m.sortAscending {
		t.Fatalf("expected sort by duration ascending")
	}
	m.toggleSort(sortByDuration)
	if m.sortAscending {
		t.Fatalf("expected sort order to flip")
	}
}

func TestResetFilters(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	m.filters = filterState{name: "x"}
	modelAny, _ := m.handleKeyMsg(keyMsg("r"))
	m = modelAny.(model)
	if m.filters.name != "" {
		t.Fatalf("expected filters cleared")
	}
}

func TestSyncFilterFocus(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.showFilters = true
	m.inputs.focus = 1
	m.syncFilterFocus()
	if !m.inputs.fields[1].Focused() {
		t.Fatalf("expected second field focused")
	}
}

func TestLoadVideosCmdProducesMessage(t *testing.T) {
	root := t.TempDir()
	video := filepath.Join(root, "clip.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cmd := loadVideosCmd(root, filepath.Join(root, "cache.json"), &loadProgress{})
	msg := cmd()
	if _, ok := msg.(videosLoadedMsg); !ok {
		t.Fatalf("expected videosLoadedMsg")
	}
}

func TestProgressTickerCmdProducesMsg(t *testing.T) {
	progress := &loadProgress{}
	progress.SetTotal(1)
	cmd := progressTickerCmd(progress)
	if cmd == nil {
		t.Fatalf("expected ticker command")
	}
}

func keyMsg(value string) tea.KeyMsg {
	if len(value) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value), Alt: false}
}
