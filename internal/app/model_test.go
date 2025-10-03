package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/table"
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
	m.filters = filterState{name: "flow", minEnabled: true, minMinutes: 5, maxEnabled: true, maxMinutes: 20, tags: "calm"}
	desc := m.describeFilters()
	if !strings.Contains(desc, "flow") || !strings.Contains(desc, ">=5") || !strings.Contains(desc, "calm") {
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

func TestUpdateStatusAfterLoadTagWarning(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	msg := videosLoadedMsg{videos: []video{{Name: "a", Path: "a.mp4"}}, tagErr: errors.New("bad json")}
	modelAny, _ := m.Update(msg)
	m = modelAny.(model)
	if !strings.Contains(m.statusMessage, "tag warning") {
		t.Fatalf("expected tag warning, got %s", m.statusMessage)
	}
}

func TestPassesFiltersBounds(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.filters = filterState{minEnabled: true, minMinutes: 5, maxEnabled: true, maxMinutes: 15}
	video := video{Name: "clip", Duration: 10 * time.Minute, Tags: []string{"calm", "focus"}}
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
	m.filters = filterState{tags: "calm"}
	if !m.passesFilters(video) {
		t.Fatalf("expected tag filter to include video")
	}
	m.filters = filterState{tags: "power"}
	if m.passesFilters(video) {
		t.Fatalf("expected tag filter to exclude video")
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

func TestHandleTagsSavedUpdatesModel(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	vid := video{Name: "clip.mp4", Path: filepath.Join(root, "clip.mp4")}
	m.videos = []video{vid}
	m.filtered = []video{vid}
	m.table.SetRows([]table.Row{videoRow(vid)})
	modelAny, _ := m.handleTagsSaved(tagsSavedMsg{path: vid.Path, tags: []string{"calm", "focus"}})
	m = modelAny.(model)
	if len(m.videos[0].Tags) != 2 {
		t.Fatalf("expected tags recorded")
	}
	if len(m.filtered) != 1 || len(m.filtered[0].Tags) != 2 {
		t.Fatalf("expected filtered list updated")
	}
	if !strings.Contains(m.statusMessage, "Tags updated") {
		t.Fatalf("expected status update, got %s", m.statusMessage)
	}
}

func TestOpenTagEditorLoadsExistingTags(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	vid := video{Name: "clip.mp4", Path: filepath.Join(root, "clip.mp4"), Tags: []string{"calm"}}
	m.videos = []video{vid}
	m.applyFiltersAndSort()
	modelAny, _ := m.openTagEditor()
	m = modelAny.(model)
	if !m.editingTags {
		t.Fatalf("expected tag editor to open")
	}
	if m.tagInput.Value() != "calm" {
		t.Fatalf("expected input prefilled, got %s", m.tagInput.Value())
	}
}

func TestParseTagInput(t *testing.T) {
	result := parseTagInput(" calm , Focus , focus , ")
	if len(result) != 2 {
		t.Fatalf("expected two tags, got %v", result)
	}
	if result[0] != "calm" || result[1] != "Focus" {
		t.Fatalf("unexpected order or casing: %v", result)
	}
	if out := parseTagInput("   "); out != nil {
		t.Fatalf("expected nil for blank input, got %v", out)
	}
}

func TestSaveTagsCmd(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	msg := saveTagsCmd(videoPath, []string{" calm ", "calm", "Focus"})()
	result, ok := msg.(tagsSavedMsg)
	if !ok {
		t.Fatalf("expected tagsSavedMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if len(result.tags) != 2 {
		t.Fatalf("expected deduped tags, got %v", result.tags)
	}
}

func TestHelpLineAfterTagEdit(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	vid := video{Name: "clip.mp4", Path: filepath.Join(root, "clip.mp4")}
	loaded := videosLoadedMsg{videos: []video{vid}, cache: newDurationCache(filepath.Join(root, "cache.json"))}
	modelAny, _ := m.handleVideosLoaded(loaded)
	m = modelAny.(model)
	if view := m.View(); !strings.Contains(view, "Loaded 1 videos") {
		t.Fatalf("expected base status in view: %s", view)
	}
	modelAny, _ = m.openTagEditor()
	m = modelAny.(model)
	m.tagInput.SetValue("calm")
	modelAny, cmd := m.handleTagKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = modelAny.(model)
	if cmd == nil {
		t.Fatalf("expected save command")
	}
	if view := m.View(); !strings.Contains(view, "Loaded 1 videos") || !strings.Contains(view, "Saving tags") {
		t.Fatalf("expected combined status while saving: %s", view)
	}
	msg := cmd().(tagsSavedMsg)
	modelAny, _ = m.handleTagsSaved(msg)
	m = modelAny.(model)
	if view := m.View(); !strings.Contains(view, "Loaded 1 videos") {
		t.Fatalf("expected base status after save: %s", view)
	}
	if view := m.View(); !strings.Contains(view, "↑/↓ navigate") {
		t.Fatalf("expected help line after save: %s", view)
	}
	if !strings.Contains(m.statusMessage, "Tags updated") {
		t.Fatalf("expected status message to report update, got %s", m.statusMessage)
	}
}

func TestToggleHelpKeys(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	vid := video{Name: "clip.mp4", Path: filepath.Join(root, "clip.mp4")}
	loaded := videosLoadedMsg{videos: []video{vid}, cache: newDurationCache(filepath.Join(root, "cache.json"))}
	modelAny, _ := m.handleVideosLoaded(loaded)
	m = modelAny.(model)
	helpLine := "↑/↓ navigate  •  enter play  •  s sort  •  / filter  •  c crop  •  t edit tags  •  q quit"
	if view := m.View(); !strings.Contains(view, helpLine) {
		t.Fatalf("expected help line visible: %s", view)
	}
	modelAny, _ = m.handleKeyMsg(keyMsg("H"))
	m = modelAny.(model)
	if m.showHelp {
		t.Fatalf("expected help to be hidden")
	}
	if view := m.View(); strings.Contains(view, helpLine) {
		t.Fatalf("expected help line hidden: %s", view)
	}
	if !strings.Contains(m.statusMessage, "Help hidden") {
		t.Fatalf("expected help hidden status, got %s", m.statusMessage)
	}
	modelAny, _ = m.handleKeyMsg(keyMsg("h"))
	m = modelAny.(model)
	if !m.showHelp {
		t.Fatalf("expected help to be shown")
	}
	if view := m.View(); !strings.Contains(view, helpLine) {
		t.Fatalf("expected help line visible again: %s", view)
	}
}

func TestWindowResizeExpandsNameColumn(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	initial := m.table.Columns()[0].Width
	modelAny, _ := m.Update(tea.WindowSizeMsg{Width: 180, Height: 40})
	m = modelAny.(model)
	resized := m.table.Columns()[0].Width
	if resized <= initial {
		t.Fatalf("expected name column to expand, initial=%d resized=%d", initial, resized)
	}
}

func TestWindowResizeShrinksColumnsGracefully(t *testing.T) {
	m, err := newModel(Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	modelAny, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 40})
	m = modelAny.(model)
	cols := m.table.Columns()
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns")
	}
	if cols[0].Width < nameColumnFloorWidth {
		t.Fatalf("expected name column >= floor, got %d", cols[0].Width)
	}
	if cols[3].Width < tagsColumnFloorWidth {
		t.Fatalf("expected tags column >= floor, got %d", cols[3].Width)
	}
}

func TestFilterByDurationRange(t *testing.T) {
	root := t.TempDir()
	m, err := newModel(Options{Root: root})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.loading = false
	m.videos = []video{{Name: "short.mp4", Duration: 5 * time.Minute}, {Name: "long.mp4", Duration: 20 * time.Minute}}
	m.applyFiltersAndSort()
	modelAny, _ := m.handleKeyMsg(keyMsg("/"))
	m = modelAny.(model)
	if !m.showFilters {
		t.Fatalf("expected filters to open")
	}
	// Move focus to min minutes field.
	modelAny, _ = m.handleFilterKey(tea.KeyMsg{Type: tea.KeyTab})
	m = modelAny.(model)
	// Enter "10".
	modelAny, _ = m.handleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = modelAny.(model)
	modelAny, _ = m.handleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = modelAny.(model)
	// Apply filter.
	modelAny, _ = m.handleFilterKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = modelAny.(model)
	if len(m.filtered) != 1 || m.filtered[0].Name != "long.mp4" {
		t.Fatalf("expected only long video after filtering, got %+v", m.filtered)
	}
	if !strings.Contains(m.statusMessage, "Filters applied") {
		t.Fatalf("expected status update, got %s", m.statusMessage)
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
