package app

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type stubProgram struct {
	err error
}

func (s stubProgram) Run() (tea.Model, error) {
	return nil, s.err
}

func TestRunInvokesProgram(t *testing.T) {
	t.Helper()
	original := programFactory
	defer func() { programFactory = original }()
	programFactory = func(tea.Model) teaProgram { return stubProgram{} }
	if err := Run(Options{Root: t.TempDir()}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunPropagatesError(t *testing.T) {
	t.Helper()
	original := programFactory
	defer func() { programFactory = original }()
	errRun := errors.New("boom")
	programFactory = func(tea.Model) teaProgram { return stubProgram{err: errRun} }
	err := Run(Options{Root: t.TempDir()})
	if !errors.Is(err, errRun) {
		t.Fatalf("expected error propagation, got %v", err)
	}
}
