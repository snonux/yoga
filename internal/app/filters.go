package app

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type filterState struct {
	name       string
	minEnabled bool
	minMinutes int
	maxEnabled bool
	maxMinutes int
	tags       string
}

type filterInputs struct {
	fields []textinput.Model
	focus  int
}

func (m *model) applyFilterInputs() error {
	name := strings.TrimSpace(m.inputs.fields[0].Value())
	minText := strings.TrimSpace(m.inputs.fields[1].Value())
	maxText := strings.TrimSpace(m.inputs.fields[2].Value())
	tags := strings.TrimSpace(m.inputs.fields[3].Value())

	filters := filterState{name: name, tags: tags}
	if err := populateMinFilter(&filters, minText); err != nil {
		return err
	}
	if err := populateMaxFilter(&filters, maxText); err != nil {
		return err
	}
	if filters.minEnabled && filters.maxEnabled && filters.minMinutes > filters.maxMinutes {
		return errors.New("min minutes cannot exceed max minutes")
	}
	m.filters = filters
	return nil
}

func populateMinFilter(dst *filterState, value string) error {
	if value == "" {
		return nil
	}
	minutes, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid min minutes: %q", value)
	}
	if minutes < 0 {
		return errors.New("min minutes must be positive")
	}
	dst.minEnabled = true
	dst.minMinutes = minutes
	return nil
}

func populateMaxFilter(dst *filterState, value string) error {
	if value == "" {
		return nil
	}
	minutes, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid max minutes: %q", value)
	}
	if minutes < 0 {
		return errors.New("max minutes must be positive")
	}
	dst.maxEnabled = true
	dst.maxMinutes = minutes
	return nil
}

func (m *model) resetFilters() {
	m.filters = filterState{}
	for i := range m.inputs.fields {
		m.inputs.fields[i].SetValue("")
	}
}

func (m *model) updateFilterInputs(msg tea.Msg) (filterInputs, tea.Cmd) {
	inputs := m.inputs
	var cmds []tea.Cmd
	for i := range inputs.fields {
		var cmd tea.Cmd
		inputs.fields[i], cmd = inputs.fields[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return inputs, tea.Batch(cmds...)
}

func (m model) describeFilters() string {
	parts := []string{}
	if m.filters.name != "" {
		parts = append(parts, fmt.Sprintf("name contains %q", m.filters.name))
	}
	if m.filters.tags != "" {
		parts = append(parts, fmt.Sprintf("tags contain %q", m.filters.tags))
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

func (m *model) passesFilters(v Video) bool {
	if m.filters.name != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(m.filters.name)) {
		return false
	}
	durMinutes := int(v.Duration.Round(time.Minute) / time.Minute)
	if m.filters.minEnabled && (v.Duration == 0 || durMinutes < m.filters.minMinutes) {
		return false
	}
	if m.filters.maxEnabled && (v.Duration == 0 || durMinutes > m.filters.maxMinutes) {
		return false
	}
	if m.filters.tags != "" {
		query := strings.ToLower(m.filters.tags)
		matched := false
		for _, tag := range v.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (m *model) renderFilterModal() string {
	var b strings.Builder
	b.WriteString("Filter videos\n")
	b.WriteString("(Enter to apply, Esc to cancel)\n\n")
	labels := []string{"Name contains:", "Min length (minutes):", "Max length (minutes):", "Tags contain:"}
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
