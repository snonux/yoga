package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type teaProgram interface {
	Run() (tea.Model, error)
}

var programFactory = func(m tea.Model) teaProgram {
	return tea.NewProgram(m, tea.WithAltScreen())
}

// Run bootstraps the Bubble Tea program with the provided options.
func Run(opts Options) error {
	model, err := newModel(opts)
	if err != nil {
		return fmt.Errorf("create model: %w", err)
	}
	program := programFactory(model)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}
	return nil
}
