package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) openTagEditor() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		m.statusMessage = "No videos to edit"
		return m, nil
	}
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filtered) {
		m.statusMessage = "No selection"
		return m, nil
	}
	video := m.filtered[cursor]
	m.editingTags = true
	m.tagEditPath = video.Path
	m.tagInput = cloneInput(m.tagInput)
	m.tagInput.SetValue(strings.Join(video.Tags, ", "))
	m.tagInput.CursorEnd()
	m.tagInput.Focus()
	m.statusMessage = fmt.Sprintf("Editing tags for %s", video.Name)
	return m, nil
}

func (m model) handleTagKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editingTags = false
		m.tagEditPath = ""
		m.tagInput.Blur()
		m.statusMessage = "Tag edit cancelled"
		return m, nil
	case "enter":
		return m.commitTags()
	}
	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (m model) commitTags() (tea.Model, tea.Cmd) {
	if m.tagEditPath == "" {
		m.editingTags = false
		m.tagInput.Blur()
		m.statusMessage = "No video selected"
		return m, nil
	}
	value := m.tagInput.Value()
	tags := parseTagInput(value)
	m.editingTags = false
	m.tagInput.Blur()
	path := m.tagEditPath
	m.tagEditPath = ""
	name := filepath.Base(path)
	m.statusMessage = fmt.Sprintf("Saving tags for %s", name)
	return m, saveTagsCmd(path, tags)
}

func (m model) handleTagsSaved(msg tagsSavedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.editingTags = false
		m.tagEditPath = ""
		m.tagInput.Blur()
		m.showHelp = true
		m.statusMessage = fmt.Sprintf("Tag save error: %v", msg.err)
		return m, nil
	}
	m.editingTags = false
	m.tagEditPath = ""
	m.tagInput.Blur()
	m.showHelp = true
	m.setVideoTags(msg.path, msg.tags)
	m.applyFiltersAndSort()
	m.restoreSelection(msg.path)
	if len(msg.tags) == 0 {
		m.statusMessage = "Tags cleared"
		return m, nil
	}
	m.statusMessage = fmt.Sprintf("Tags updated (%d)", len(msg.tags))
	return m, nil
}

func (m *model) setVideoTags(path string, tags []string) {
	for i := range m.videos {
		if m.videos[i].Path == path {
			m.videos[i].Tags = append([]string{}, tags...)
			return
		}
	}
}

func parseTagInput(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var tags []string
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		tags = append(tags, trimmed)
	}
	return tags
}

func cloneInput(in textinput.Model) textinput.Model {
	copy := in
	return copy
}

func (m model) renderTagModal() string {
	var b strings.Builder
	b.WriteString("Edit tags\n")
	b.WriteString("(comma separated)\n\n")
	b.WriteString(m.tagInput.View())
	b.WriteString("\n\n")
	b.WriteString("Enter to save, Esc to cancel")
	return filterStyle.Render(b.String())
}
