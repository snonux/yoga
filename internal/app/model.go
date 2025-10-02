package app

import (
	"fmt"
	"path/filepath"
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

func newModel(opts Options) (model, error) {
	tbl := buildTable()
	inputs := buildFilterInputs()
	inputs.fields[0].Focus()

	progress := &loadProgress{}
	cachePath := filepath.Join(opts.Root, ".video_duration_cache.json")

	return model{
		table:         tbl,
		inputs:        inputs,
		sortField:     sortByName,
		sortAscending: true,
		statusMessage: "Scanning for videos...",
		loading:       true,
		root:          opts.Root,
		progress:      progress,
		cachePath:     cachePath,
		cropValue:     opts.Crop,
		cropEnabled:   opts.Crop != "",
	}, nil
}

func buildTable() table.Model {
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

	return filterInputs{
		fields: []textinput.Model{nameInput, minInput, maxInput},
		focus:  0,
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
	default:
		return m.updateTable(msg)
	}
}

func (m model) View() string {
	if m.loading {
		return statusStyle.Render("Loading videos, please wait...")
	}
	body := m.renderBody()
	if m.showFilters {
		return body + "\n\n" + m.renderFilterModal()
	}
	return body
}

func (m model) renderBody() string {
	helpLines := []string{
		"↑/↓ navigate  •  enter play  •  s sort  •  / filter  •  c copy path  •  q quit",
	}
	info := statusStyle.Render(m.statusMessage)
	progressLine := m.renderProgressLine()
	content := tableStyle.Render(m.table.View())
	help := strings.Join(helpLines, "\n")
	parts := []string{content}
	if progressLine != "" {
		parts = append(parts, progressLine)
	}
	parts = append(parts, info, help)
	return strings.Join(parts, "\n")
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
