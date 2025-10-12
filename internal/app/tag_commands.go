package app

import (
	"codeberg.org/snonux/yoga/internal/tags"
	tea "github.com/charmbracelet/bubbletea"
)

func saveTagsCmd(path string, entries []string) tea.Cmd {
	// Copy slice to avoid accidental mutation after scheduling command.
	values := append([]string{}, entries...)
	return func() tea.Msg {
		if err := tags.Save(path, values); err != nil {
			return tagsSavedMsg{path: path, err: err}
		}
		sanitized, err := tags.Load(path)
		if err != nil {
			return tagsSavedMsg{path: path, err: err}
		}
		return tagsSavedMsg{path: path, tags: sanitized}
	}
}
