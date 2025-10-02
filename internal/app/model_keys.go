package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if cmd, handled := globalKeyHandler(msg); handled {
		return m, cmd
	}
	if m.loading {
		return m, nil
	}
	if m.showFilters {
		return m.handleFilterKey(msg)
	}
	return m.handleTableKey(msg)
}

func globalKeyHandler(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit, true
	default:
		return nil, false
	}
}

func (m model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showFilters = false
		m.statusMessage = "Filter closed"
		return m, nil
	case "enter":
		cmd := m.applyFiltersFromInputs()
		return m, cmd
	case "tab":
		m.inputs.focus = (m.inputs.focus + 1) % len(m.inputs.fields)
	case "shift+tab":
		m.inputs.focus = (m.inputs.focus - 1 + len(m.inputs.fields)) % len(m.inputs.fields)
	}
	m.syncFilterFocus()
	updated, cmd := m.updateFilterInputs(msg)
	m.inputs = updated
	return m, cmd
}

func (m *model) applyFiltersFromInputs() tea.Cmd {
	if err := m.applyFilterInputs(); err != nil {
		m.statusMessage = err.Error()
		return nil
	}
	m.showFilters = false
	m.applyFiltersAndSort()
	m.statusMessage = fmt.Sprintf("Filters applied (%d videos)", len(m.filtered))
	return nil
}

func (m *model) syncFilterFocus() {
	for i := range m.inputs.fields {
		if i == m.inputs.focus {
			m.inputs.fields[i].Focus()
			continue
		}
		m.inputs.fields[i].Blur()
	}
}

func (m model) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/", "f":
		return m.openFilters()
	case "enter":
		return m.playSelection()
	case "n":
		return m.sortAndReport(sortByName)
	case "l":
		return m.sortAndReport(sortByDuration)
	case "a":
		return m.sortAndReport(sortByAge)
	case "c":
		return m.toggleCrop()
	case "r":
		return m.resetFilterState()
	default:
		return m.updateTable(msg)
	}
}

func (m model) openFilters() (tea.Model, tea.Cmd) {
	m.showFilters = true
	m.statusMessage = "Editing filters"
	return m, nil
}

func (m model) playSelection() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.filtered) {
		return m, nil
	}
	video := m.filtered[idx]
	m.statusMessage = fmt.Sprintf("Launching VLC: %s", video.Name)
	return m, playVideoCmd(video.Path, m.activeCrop())
}

func (m model) sortAndReport(field sortField) (tea.Model, tea.Cmd) {
	m.toggleSort(field)
	m.applyFiltersAndSort()
	m.statusMessage = fmt.Sprintf("Sorted %d videos", len(m.filtered))
	return m, nil
}

func (m model) toggleCrop() (tea.Model, tea.Cmd) {
	if m.cropValue == "" {
		m.statusMessage = "No crop value set (start with --crop)"
		return m, nil
	}
	m.cropEnabled = !m.cropEnabled
	if m.cropEnabled {
		m.statusMessage = fmt.Sprintf("Crop enabled (%s)", m.cropValue)
		return m, nil
	}
	m.statusMessage = "Crop disabled"
	return m, nil
}

func (m model) resetFilterState() (tea.Model, tea.Cmd) {
	m.resetFilters()
	m.applyFiltersAndSort()
	m.statusMessage = fmt.Sprintf("Filters cleared (%d videos)", len(m.filtered))
	return m, nil
}
