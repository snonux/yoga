package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const Version = "v0.0.0"

var (
	videoExtensions = map[string]struct{}{
		".mp4": {},
		".mkv": {},
		".mov": {},
		".avi": {},
		".wmv": {},
		".m4v": {},
	}
	tableStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	filterStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("105")).Padding(1, 2)
	statusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
)

type video struct {
	Name     string
	Path     string
	Duration time.Duration
	ModTime  time.Time
	Size     int64
	Err      error
}

type videosLoadedMsg struct {
	videos   []video
	err      error
	cacheErr error
	pending  []string
	cache    *durationCache
}

type playVideoMsg struct {
	path string
	err  error
}

type progressUpdateMsg struct {
	processed int
	total     int
	done      bool
}

type durationUpdateMsg struct {
	path     string
	duration time.Duration
	err      error
}

type loadProgress struct {
	mu        sync.Mutex
	total     int
	processed int
	done      bool
}

func (p *loadProgress) Reset() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = 0
	p.processed = 0
	p.done = false
}

func (p *loadProgress) SetTotal(total int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.total = total
	p.mu.Unlock()
}

func (p *loadProgress) Increment() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.processed++
	p.mu.Unlock()
}

func (p *loadProgress) MarkDone() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
}

func (p *loadProgress) Snapshot() (processed, total int, done bool) {
	if p == nil {
		return 0, 0, true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.processed, p.total, p.done
}

type cacheEntry struct {
	DurationSeconds float64 `json:"duration_seconds"`
	ModTimeUnix     int64   `json:"mod_time_unix"`
	Size            int64   `json:"size"`
}

type durationCache struct {
	path    string
	entries map[string]cacheEntry
	mu      sync.Mutex
	dirty   bool
}

func newDurationCache(path string) *durationCache {
	return &durationCache{
		path:    path,
		entries: make(map[string]cacheEntry),
	}
}

func loadDurationCache(path string) (*durationCache, error) {
	cache := newDurationCache(path)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache.entries); err != nil {
		cache.entries = make(map[string]cacheEntry)
		return cache, err
	}
	return cache, nil
}

func (c *durationCache) Lookup(path string, info os.FileInfo) (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[path]
	if !ok {
		return 0, false
	}
	if entry.ModTimeUnix != info.ModTime().Unix() || entry.Size != info.Size() {
		delete(c.entries, path)
		c.dirty = true
		return 0, false
	}
	if entry.DurationSeconds <= 0 {
		return 0, false
	}
	return time.Duration(entry.DurationSeconds * float64(time.Second)), true
}

func (c *durationCache) Record(path string, info os.FileInfo, dur time.Duration) error {
	if c == nil || dur <= 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]cacheEntry)
	}
	c.entries[path] = cacheEntry{
		DurationSeconds: dur.Seconds(),
		ModTimeUnix:     info.ModTime().Unix(),
		Size:            info.Size(),
	}
	c.dirty = true
	return nil
}

func (c *durationCache) Flush() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if !c.dirty {
		c.mu.Unlock()
		return nil
	}
	snapshot := make(map[string]cacheEntry, len(c.entries))
	for k, v := range c.entries {
		snapshot[k] = v
	}
	c.dirty = false
	c.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

type sortField int

const (
	sortByName sortField = iota
	sortByDuration
	sortByAge
)

type filterState struct {
	name       string
	minEnabled bool
	minMinutes int
	maxEnabled bool
	maxMinutes int
}

type filterInputs struct {
	fields []textinput.Model
	focus  int
}

type model struct {
	table            table.Model
	videos           []video
	filtered         []video
	filters          filterState
	inputs           filterInputs
	showFilters      bool
	sortField        sortField
	sortAscending    bool
	statusMessage    string
	loading          bool
	err              error
	root             string
	progress         *loadProgress
	cachePath        string
	cache            *durationCache
	pendingDurations []string
	durationTotal    int
	durationDone     int
	durationInFlight int
	cropValue        string
	cropEnabled      bool
}

func main() {
	crop := flag.String("crop", "", "Optional crop aspect for VLC (e.g. 5:4)")
	printVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *printVersion {
		fmt.Println("Yoga version", Version)
		os.Exit(0)
	}

	root := mustWorkspaceRoot()
	m := newModel(root, strings.TrimSpace(*crop))
	if err := tea.NewProgram(m, tea.WithAltScreen()).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func mustWorkspaceRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return cwd
}

func newModel(root, vlcCrop string) model {
	columns := []table.Column{
		{Title: headerStyle.Render("Name"), Width: 50},
		{Title: headerStyle.Render("Duration"), Width: 12},
		{Title: headerStyle.Render("Age"), Width: 14},
		{Title: headerStyle.Render("Path"), Width: 40},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	tbl.SetStyles(table.DefaultStyles())

	nameInput := textinput.New()
	nameInput.Placeholder = "substring"
	nameInput.Prompt = "Name: "
	nameInput.CharLimit = 256

	minInput := textinput.New()
	minInput.Placeholder = "min minutes"
	minInput.Prompt = "Min minutes: "
	minInput.CharLimit = 4
	minInput.SetValue("")

	maxInput := textinput.New()
	maxInput.Placeholder = "max minutes"
	maxInput.Prompt = "Max minutes: "
	maxInput.CharLimit = 4
	maxInput.SetValue("")

	inputs := filterInputs{
		fields: []textinput.Model{nameInput, minInput, maxInput},
		focus:  0,
	}
	inputs.fields[0].Focus()

	progress := &loadProgress{}
	cachePath := filepath.Join(root, ".video_duration_cache.json")

	return model{
		table:         tbl,
		inputs:        inputs,
		sortField:     sortByName,
		sortAscending: true,
		statusMessage: "Scanning for videos...",
		loading:       true,
		root:          root,
		progress:      progress,
		cachePath:     cachePath,
		cropValue:     vlcCrop,
		cropEnabled:   vlcCrop != "",
	}
}

func (m model) Init() tea.Cmd {
	if m.progress != nil {
		m.progress.Reset()
	}
	loadCmd := loadVideosCmd(m.root, m.cachePath, m.progress)
	if m.progress != nil {
		return tea.Batch(loadCmd, progressTickerCmd(m.progress))
	}
	return loadCmd
}

func loadVideosCmd(root, cachePath string, progress *loadProgress) tea.Cmd {
	return func() tea.Msg {
		cache, cacheErr := loadDurationCache(cachePath)
		vids, pending, err := loadVideos(root, cache, progress)
		if progress != nil {
			progress.MarkDone()
		}
		return videosLoadedMsg{videos: vids, err: err, cacheErr: cacheErr, pending: pending, cache: cache}
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
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isVideo(path) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
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
			if progress != nil {
				progress.Increment()
			}
			continue
		}
		var dur time.Duration
		if cache != nil {
			if cached, ok := cache.Lookup(path, info); ok {
				dur = cached
			} else {
				pending = append(pending, path)
			}
		} else {
			pending = append(pending, path)
		}
		videos = append(videos, video{
			Name:     filepath.Base(path),
			Path:     path,
			Duration: dur,
			ModTime:  info.ModTime(),
			Size:     info.Size(),
			Err:      nil,
		})
		if progress != nil {
			progress.Increment()
		}
	}

	return videos, pending, nil
}

func playVideoCmd(path, crop string) tea.Cmd {
	return func() tea.Msg {
		args := []string{}
		if crop != "" {
			args = append(args, "--crop", crop)
		}
		args = append(args, path)
		cmd := exec.Command("vlc", args...)
		if err := cmd.Start(); err != nil {
			return playVideoMsg{path: path, err: err}
		}
		go func() {
			_ = cmd.Wait()
		}()
		return playVideoMsg{path: path}
	}
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
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(f * float64(time.Second)), nil
}

func isVideo(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := videoExtensions[ext]
	return ok
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case progressUpdateMsg:
		if m.loading {
			if msg.total > 0 && !msg.done {
				m.statusMessage = fmt.Sprintf("Loading videos %d/%d...", msg.processed, msg.total)
			} else if msg.done {
				if msg.total == 0 {
					m.statusMessage = "No videos found"
				} else {
					m.statusMessage = fmt.Sprintf("Loaded %d videos", msg.total)
				}
			}
		}
		if msg.done {
			return m, nil
		}
		return m, progressTickerCmd(m.progress)
	case durationUpdateMsg:
		if msg.path != "" {
			m.updateVideoDuration(msg.path, msg.duration, msg.err)
			m.durationDone++
			if msg.err != nil {
				m.statusMessage = fmt.Sprintf("Duration error for %s: %v", filepath.Base(msg.path), msg.err)
			} else if m.durationTotal > 0 {
				m.statusMessage = fmt.Sprintf("Probing durations %d/%d...", m.durationDone, m.durationTotal)
			}
		}
		if m.durationInFlight > 0 {
			m.durationInFlight--
		}
		selectedPath := ""
		if idx := m.table.Cursor(); idx >= 0 && idx < len(m.filtered) {
			selectedPath = m.filtered[idx].Path
		}
		m.applyFiltersAndSort()
		if selectedPath != "" {
			m.restoreSelection(selectedPath)
		}
		if m.durationDone >= m.durationTotal && m.durationInFlight == 0 {
			if m.cache != nil {
				if err := m.cache.Flush(); err != nil {
					m.statusMessage = fmt.Sprintf("Duration cache flush error: %v", err)
				} else {
					m.statusMessage = fmt.Sprintf("Durations ready (%d videos)", len(m.filtered))
				}
			} else {
				m.statusMessage = fmt.Sprintf("Durations ready (%d videos)", len(m.filtered))
			}
			m.pendingDurations = nil
			m.durationTotal = 0
			m.durationDone = 0
			m.durationInFlight = 0
			return m, nil
		}
		if cmd := m.dequeueDurationCmd(); cmd != nil {
			return m, cmd
		}
		return m, nil
	case videosLoadedMsg:
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
		if len(m.filtered) == 0 {
			m.statusMessage = "No videos found"
		} else {
			if len(msg.pending) > 0 {
				if msg.cacheErr != nil {
					m.statusMessage = fmt.Sprintf("Loaded %d videos (cache warning: %v), probing durations...", len(m.filtered), msg.cacheErr)
				} else {
					m.statusMessage = fmt.Sprintf("Loaded %d videos, probing durations...", len(m.filtered))
				}
			} else if msg.cacheErr != nil {
				m.statusMessage = fmt.Sprintf("Loaded %d videos (cache warning: %v)", len(m.filtered), msg.cacheErr)
			} else {
				m.statusMessage = fmt.Sprintf("Loaded %d videos", len(m.filtered))
			}
		}
		m.durationInFlight = 0
		if len(msg.pending) == 0 {
			return m, nil
		}
		cmd := m.startDurationWorkers()
		if cmd == nil {
			return m, nil
		}
		return m, cmd
	case playVideoMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to launch VLC: %v", msg.err)
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Playing via VLC: %s", trimPath(msg.path))
		return m, nil
	}

	if m.showFilters {
		updated, cmd := m.updateFilterInputs(msg)
		m.inputs = updated
		return m, cmd
	}

	tbl, cmd := m.table.Update(msg)
	m.table = tbl
	return m, cmd
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	if m.loading {
		return m, nil
	}

	if m.showFilters {
		switch msg.String() {
		case "esc":
			m.showFilters = false
			m.statusMessage = "Filter closed"
			return m, nil
		case "enter":
			if err := m.applyFilterInputs(); err != nil {
				m.statusMessage = err.Error()
			} else {
				m.showFilters = false
				m.applyFiltersAndSort()
				m.statusMessage = fmt.Sprintf("Filters applied (%d videos)", len(m.filtered))
			}
			return m, nil
		case "tab":
			m.inputs.focus = (m.inputs.focus + 1) % len(m.inputs.fields)
		case "shift+tab":
			m.inputs.focus = (m.inputs.focus - 1 + len(m.inputs.fields)) % len(m.inputs.fields)
		default:
			// no-op; handled below
		}
		for i := range m.inputs.fields {
			if i == m.inputs.focus {
				m.inputs.fields[i].Focus()
			} else {
				m.inputs.fields[i].Blur()
			}
		}
		updated, cmd := m.updateFilterInputs(msg)
		m.inputs = updated
		return m, cmd
	}

	switch msg.String() {
	case "f":
		m.showFilters = true
		m.statusMessage = "Editing filters"
		return m, nil
	case "enter":
		if len(m.filtered) == 0 {
			return m, nil
		}
		idx := m.table.Cursor()
		if idx < 0 || idx >= len(m.filtered) {
			return m, nil
		}
		vid := m.filtered[idx]
		m.statusMessage = fmt.Sprintf("Launching VLC: %s", vid.Name)
		return m, playVideoCmd(vid.Path, m.activeCrop())
	case "n":
		m.toggleSort(sortByName)
	case "l":
		m.toggleSort(sortByDuration)
	case "a":
		m.toggleSort(sortByAge)
	case "c":
		if m.cropValue == "" {
			m.statusMessage = "No crop value set (start with --crop)"
			return m, nil
		}
		m.cropEnabled = !m.cropEnabled
		if m.cropEnabled {
			m.statusMessage = fmt.Sprintf("Crop enabled (%s)", m.cropValue)
		} else {
			m.statusMessage = "Crop disabled"
		}
		return m, nil
	case "r":
		m.resetFilters()
		m.applyFiltersAndSort()
		m.statusMessage = fmt.Sprintf("Filters cleared (%d videos)", len(m.filtered))
	default:
		tbl, cmd := m.table.Update(msg)
		m.table = tbl
		return m, cmd
	}

	m.applyFiltersAndSort()
	m.statusMessage = fmt.Sprintf("Sorted %d videos", len(m.filtered))
	return m, nil
}

func (m model) updateFilterInputs(msg tea.Msg) (filterInputs, tea.Cmd) {
	inputs := m.inputs
	var cmds []tea.Cmd
	for i := range inputs.fields {
		var cmd tea.Cmd
		inputs.fields[i], cmd = inputs.fields[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return inputs, tea.Batch(cmds...)
}

func (m *model) applyFilterInputs() error {
	name := strings.TrimSpace(m.inputs.fields[0].Value())
	minText := strings.TrimSpace(m.inputs.fields[1].Value())
	maxText := strings.TrimSpace(m.inputs.fields[2].Value())

	filters := filterState{name: name}

	if minText != "" {
		minVal, err := strconv.Atoi(minText)
		if err != nil {
			return fmt.Errorf("invalid min minutes: %q", minText)
		}
		if minVal < 0 {
			return fmt.Errorf("min minutes must be positive")
		}
		filters.minEnabled = true
		filters.minMinutes = minVal
	}

	if maxText != "" {
		maxVal, err := strconv.Atoi(maxText)
		if err != nil {
			return fmt.Errorf("invalid max minutes: %q", maxText)
		}
		if maxVal < 0 {
			return fmt.Errorf("max minutes must be positive")
		}
		filters.maxEnabled = true
		filters.maxMinutes = maxVal
	}

	if filters.minEnabled && filters.maxEnabled && filters.minMinutes > filters.maxMinutes {
		return errors.New("min minutes cannot exceed max minutes")
	}

	m.filters = filters
	return nil
}

func (m *model) resetFilters() {
	m.filters = filterState{}
	for i := range m.inputs.fields {
		m.inputs.fields[i].SetValue("")
	}
}

func (m *model) updateVideoDuration(path string, dur time.Duration, err error) {
	for i := range m.videos {
		if m.videos[i].Path == path {
			m.videos[i].Duration = dur
			if err != nil {
				m.videos[i].Err = err
			} else {
				m.videos[i].Err = nil
			}
			break
		}
	}
}

func (m *model) restoreSelection(path string) {
	for i, v := range m.filtered {
		if v.Path == path {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m model) activeCrop() string {
	if m.cropEnabled && m.cropValue != "" {
		return m.cropValue
	}
	return ""
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
		if cmd := m.dequeueDurationCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *model) toggleSort(target sortField) {
	if m.sortField == target {
		m.sortAscending = !m.sortAscending
	} else {
		m.sortField = target
		m.sortAscending = true
	}
}

func (m *model) applyFiltersAndSort() {
	filtered := make([]video, 0, len(m.videos))
	for _, v := range m.videos {
		if !m.passesFilters(v) {
			continue
		}
		filtered = append(filtered, v)
	}

	sort.Slice(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		less := false
		switch m.sortField {
		case sortByName:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case sortByDuration:
			less = a.Duration < b.Duration
		case sortByAge:
			less = a.ModTime.Before(b.ModTime)
		}
		if m.sortAscending {
			return less
		}
		return !less
	})

	m.filtered = filtered
	rows := make([]table.Row, 0, len(filtered))
	for _, v := range filtered {
		rows = append(rows, videoRow(v))
	}
	m.table.SetRows(rows)
	if len(rows) > 0 {
		m.table.SetCursor(0)
	}
}

func (m model) passesFilters(v video) bool {
	f := m.filters
	if f.name != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(f.name)) {
		return false
	}
	durMinutes := int(v.Duration.Round(time.Minute) / time.Minute)
	if f.minEnabled && (v.Duration == 0 || durMinutes < f.minMinutes) {
		return false
	}
	if f.maxEnabled && (v.Duration == 0 || durMinutes > f.maxMinutes) {
		return false
	}
	return true
}

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
	now := time.Now()
	dur := now.Sub(t)
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
	if err == nil {
		if strings.HasPrefix(path, home) {
			return "~" + strings.TrimPrefix(path, home)
		}
	}
	return path
}

func (m model) View() string {
	if m.loading {
		return statusStyle.Render("Loading videos, please wait...")
	}

	if m.err != nil && len(m.filtered) == 0 {
		return statusStyle.Render(fmt.Sprintf("Failed to load videos: %v", m.err))
	}

	cropHelp := "Crop: (no crop configured; start with --crop)"
	if m.cropValue != "" {
		state := "off"
		if m.cropEnabled {
			state = "on"
		}
		cropHelp = fmt.Sprintf("Crop: c=toggle (%s %s)", state, m.cropValue)
	}
	helpLines := []string{
		"Controls: ↑/↓ move • Enter selects (noop) • q quits",
		"Sorting: n=name • l=length • a=age • r=reset filters",
		"Filters: f=toggle filter editor (tab to navigate, enter to apply, esc to cancel)",
		cropHelp,
	}
	info := statusStyle.Render(m.statusMessage)

	progressLine := ""
	if m.durationTotal > 0 {
		bar := renderProgressBar(m.durationDone, m.durationTotal, 24)
		progressLine = statusStyle.Render(fmt.Sprintf("Duration scan %s %d/%d", bar, m.durationDone, m.durationTotal))
	}

	content := tableStyle.Render(m.table.View())
	help := strings.Join(helpLines, "\n")

	var parts []string
	parts = append(parts, content)
	if progressLine != "" {
		parts = append(parts, progressLine)
	}
	parts = append(parts, info, help)
	body := strings.Join(parts, "\n")

	if m.showFilters {
		return body + "\n\n" + m.renderFilterModal()
	}

	return body
}

func (m model) renderFilterModal() string {
	var b strings.Builder
	b.WriteString("Filter videos\n")
	b.WriteString("(Enter to apply, Esc to cancel)\n\n")
	labels := []string{"Name contains:", "Min length (minutes):", "Max length (minutes):"}
	for i, field := range m.inputs.fields {
		line := fmt.Sprintf("%s %s", labels[i], field.View())
		if i == m.inputs.focus {
			line = highlightStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.filters.minEnabled || m.filters.maxEnabled || m.filters.name != "" {
		b.WriteString("\nCurrent filter: ")
		b.WriteString(m.describeFilters())
		b.WriteString("\n")
	}
	return filterStyle.Render(b.String())
}

func (m model) describeFilters() string {
	parts := []string{}
	if m.filters.name != "" {
		parts = append(parts, fmt.Sprintf("name contains %q", m.filters.name))
	}
	if m.filters.minEnabled {
		parts = append(parts, fmt.Sprintf(">=%d min", m.filters.minMinutes))
	}
	if m.filters.maxEnabled {
		parts = append(parts, fmt.Sprintf("<=%d min", m.filters.maxMinutes))
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ", ")
}
