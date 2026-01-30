package app

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
)

func (m *model) toggleSort(target sortField) {
	if m.sortField == target {
		m.sortAscending = !m.sortAscending
		return
	}
	m.sortField = target
	m.sortAscending = true
}

func (m *model) applyFiltersAndSort() {
	filtered := make([]Video, 0, len(m.videos))
	for _, v := range m.videos {
		if m.passesFilters(v) {
			filtered = append(filtered, v)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return m.less(filtered[i], filtered[j])
	})
	m.filtered = filtered
	m.updateTableRows()
}

func (m *model) less(a, b Video) bool {
	var less bool
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
}

func (m *model) updateTableRows() {
	rows := make([]table.Row, 0, len(m.filtered))
	for _, v := range m.filtered {
		rows = append(rows, videoRow(v))
	}
	m.table.SetRows(rows)
	if len(rows) > 0 {
		m.table.SetCursor(0)
	}
}
