package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type sortField int

const (
	sortByName sortField = iota
	sortByDuration
	sortByAge
)

const (
	preferredNameColumnWidth     = 40
	preferredDurationColumnWidth = 12
	preferredAgeColumnWidth      = 14
	preferredTagsColumnWidth     = 28
	nameColumnFloorWidth         = 16
	durationColumnFloorWidth     = 8
	ageColumnFloorWidth          = 10
	tagsColumnFloorWidth         = 12
)

type model struct {
	table            table.Model
	videos           []video
	filtered         []video
	filters          filterState
	inputs           filterInputs
	showFilters      bool
	editingTags      bool
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
	tagInput         textinput.Model
	tagEditPath      string
	baseStatus       string
	showHelp         bool
	viewportWidth    int
}

func newModel(opts Options) (model, error) {
	tbl := buildTable()
	inputs := buildFilterInputs()
	inputs.fields[0].Focus()
	tagInput := buildTagInput()

	progress := &loadProgress{}
	cachePath := filepath.Join(opts.Root, ".video_duration_cache.json")

	return model{
		table:         tbl,
		inputs:        inputs,
		tagInput:      tagInput,
		sortField:     sortByName,
		sortAscending: true,
		statusMessage: "Scanning for videos...",
		loading:       true,
		root:          opts.Root,
		progress:      progress,
		cachePath:     cachePath,
		cropValue:     opts.Crop,
		cropEnabled:   opts.Crop != "",
		showHelp:      true,
	}, nil
}

func buildTable() table.Model {
	columns := makeColumns(
		preferredNameColumnWidth,
		preferredDurationColumnWidth,
		preferredAgeColumnWidth,
		preferredTagsColumnWidth,
	)
	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	tbl.SetStyles(table.DefaultStyles())
	return tbl
}

func buildFilterInputs() filterInputs {
	nameInput := textinput.New()
	nameInput.Placeholder = "substring"
	nameInput.Prompt = "Name: "
	nameInput.CharLimit = 256

	minInput := textinput.New()
	minInput.Placeholder = "min minutes"
	minInput.Prompt = "Min minutes: "
	minInput.CharLimit = 4

	maxInput := textinput.New()
	maxInput.Placeholder = "max minutes"
	maxInput.Prompt = "Max minutes: "
	maxInput.CharLimit = 4

	tagInput := textinput.New()
	tagInput.Placeholder = "tag substring"
	tagInput.Prompt = "Tags: "
	tagInput.CharLimit = 256

	return filterInputs{
		fields: []textinput.Model{nameInput, minInput, maxInput, tagInput},
		focus:  0,
	}
}

func buildTagInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "comma-separated tags"
	input.Prompt = "Tags: "
	input.CharLimit = 512
	return input
}

func makeColumns(nameWidth, durationWidth, ageWidth, tagsWidth int) []table.Column {
	return []table.Column{
		{Title: headerStyle.Render("Name"), Width: nameWidth},
		{Title: headerStyle.Render("Duration"), Width: durationWidth},
		{Title: headerStyle.Render("Age"), Width: ageWidth},
		{Title: headerStyle.Render("Tags"), Width: tagsWidth},
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(typed)
	case progressUpdateMsg:
		return m.handleProgressUpdate(typed)
	case durationUpdateMsg:
		return m.handleDurationUpdate(typed)
	case videosLoadedMsg:
		return m.handleVideosLoaded(typed)
	case playVideoMsg:
		return m.handlePlayVideo(typed), nil
	case reindexVideosMsg:
		return m.handleReindexVideos(typed)
	case tagsSavedMsg:
		return m.handleTagsSaved(typed)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(typed)
	default:
		return m.updateTable(msg)
	}
}

func (m model) View() string {
	if m.loading {
		return statusStyle.Render("Loading videos, please wait...")
	}
	body := m.renderBody()
	if m.editingTags {
		return body + "\n\n" + m.renderTagModal()
	}
	if m.showFilters {
		return body + "\n\n" + m.renderFilterModal()
	}
	return body
}

func (m model) renderBody() string {
	helpLines := []string{
		"↑/↓ navigate  •  enter play  •  s sort  •  / filter  •  c crop  •  t edit tags  •  i re-index  •  q quit",
	}
	info := statusStyle.Render(m.statusText())
	progressLine := m.renderProgressLine()
	content := tableStyle.Render(m.table.View())
	parts := []string{content}
	if progressLine != "" {
		parts = append(parts, progressLine)
	}
	parts = append(parts, info)
	if m.showHelp {
		help := strings.Join(helpLines, "\n")
		parts = append(parts, help)
	}
	return strings.Join(parts, "\n")
}

func (m model) statusText() string {
	status := strings.TrimSpace(m.statusMessage)
	base := strings.TrimSpace(m.baseStatus)
	if base == "" {
		return status
	}
	if status == "" || status == base {
		return base
	}
	return fmt.Sprintf("%s • %s", base, status)
}

func (m model) showHelpBar() (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m, nil
	}
	m.showHelp = true
	if strings.Contains(m.statusMessage, "Help hidden") {
		m.statusMessage = ""
	}
	return m, nil
}

func (m model) hideHelpBar() (tea.Model, tea.Cmd) {
	if !m.showHelp {
		return m, nil
	}
	m.showHelp = false
	m.statusMessage = "Help hidden (press h to show)"
	return m, nil
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.viewportWidth = msg.Width
	m.resizeColumns(msg.Width)
	tbl, cmd := m.table.Update(msg)
	m.table = tbl
	if cmd == nil {
		return m, nil
	}
	return m, cmd
}

func (m *model) resizeColumns(totalWidth int) {
	if totalWidth <= 0 {
		return
	}
	frame := tableStyle.GetHorizontalFrameSize()
	contentWidth := totalWidth - frame
	minWidth := nameColumnFloorWidth + durationColumnFloorWidth + ageColumnFloorWidth + tagsColumnFloorWidth
	if contentWidth < minWidth {
		contentWidth = minWidth
	}
	preferred := preferredNameColumnWidth + preferredDurationColumnWidth + preferredAgeColumnWidth + preferredTagsColumnWidth
	nameWidth := preferredNameColumnWidth
	durationWidth := preferredDurationColumnWidth
	ageWidth := preferredAgeColumnWidth
	tagsWidth := preferredTagsColumnWidth
	if contentWidth >= preferred {
		extra := contentWidth - preferred
		nameWidth += extra
	} else {
		deficit := preferred - contentWidth
		if deficit > 0 {
			reduce := min(deficit, nameWidth-nameColumnFloorWidth)
			nameWidth -= reduce
			deficit -= reduce
		}
		if deficit > 0 {
			reduce := min(deficit, tagsWidth-tagsColumnFloorWidth)
			tagsWidth -= reduce
			deficit -= reduce
		}
		if deficit > 0 {
			reduce := min(deficit, ageWidth-ageColumnFloorWidth)
			ageWidth -= reduce
			deficit -= reduce
		}
		if deficit > 0 {
			reduce := min(deficit, durationWidth-durationColumnFloorWidth)
			durationWidth -= reduce
		}
	}
	m.table.SetColumns(makeColumns(nameWidth, durationWidth, ageWidth, tagsWidth))
	m.table.SetWidth(contentWidth)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m model) renderProgressLine() string {
	if m.durationTotal == 0 {
		return ""
	}
	bar := renderProgressBar(m.durationDone, m.durationTotal, 24)
	return statusStyle.Render(fmt.Sprintf("Duration scan %s %d/%d", bar, m.durationDone, m.durationTotal))
}

func (m model) updateTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	tbl, cmd := m.table.Update(msg)
	m.table = tbl
	return m, cmd
}

func (m model) handlePlayVideo(msg playVideoMsg) model {
	if msg.err != nil {
		m.statusMessage = fmt.Sprintf("Failed to launch VLC: %v", msg.err)
		return m
	}
	m.statusMessage = fmt.Sprintf("Playing via VLC: %s", trimPath(msg.path))
	return m
}

func (m model) handleReindexVideos(msg reindexVideosMsg) (tea.Model, tea.Cmd) {
	m.statusMessage = "Re-indexing videos..."
	return m, loadVideosCmd(m.root, m.cachePath, m.progress)
}

func (m model) handleVideosLoaded(msg videosLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		m.statusMessage = fmt.Sprintf("error: %v", msg.err)
	}

	if len(m.videos) == 0 {
		m.videos = msg.videos
	} else {
		existingVideos := make(map[string]int)
		for i, v := range m.videos {
			existingVideos[v.Path] = i
		}

		for _, newVideo := range msg.videos {
			if i, ok := existingVideos[newVideo.Path]; ok {
				m.videos[i] = newVideo
			} else {
				m.videos = append(m.videos, newVideo)
			}
		}
	}

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

func (m *model) dequeueDurationCmd() tea.Cmd {
	if len(m.pendingDurations) == 0 {
		return nil
	}
	path := m.pendingDurations[0]
	m.pendingDurations = m.pendingDurations[1:]
	m.durationInFlight++
	return probeDurationsCmd(path, m.cache)
}

func (m model) activeCrop() string {
	if m.cropEnabled && m.cropValue != "" {
		return m.cropValue
	}
	return ""
}

func (m model) handleProgressUpdate(msg progressUpdateMsg) (tea.Model, tea.Cmd) {
	if !m.loading {
		return m, nil
	}
	if msg.total == 0 && msg.done {
		m.statusMessage = "No videos found"
		return m, nil
	}
	if msg.done {
		m.statusMessage = fmt.Sprintf("Loaded %d videos", msg.total)
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Loading videos %d/%d...", msg.processed, msg.total)
	return m, progressTickerCmd(m.progress)
}
