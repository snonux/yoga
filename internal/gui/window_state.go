package gui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

const windowStateFile = ".yoga_window_state.json"

type windowState struct {
	Width  float64
	Height float64
	Split  float64
}

func (mw *MainWindow) saveWindowState() error {
	state := windowState{
		Width:  float64(mw.window.Canvas().Size().Width),
		Height: float64(mw.window.Canvas().Size().Height),
		Split:  mw.split.Offset,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	rootDir := mw.app.root
	path := filepath.Join(rootDir, windowStateFile)

	return os.WriteFile(path, data, 0o644)
}

func (mw *MainWindow) loadWindowState() error {
	rootDir := mw.app.root
	path := filepath.Join(rootDir, windowStateFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state windowState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	if state.Width > 0 && state.Height > 0 {
		mw.window.Resize(fyne.NewSize(float32(state.Width), float32(state.Height)))
	}

	if state.Split > 0 && state.Split < 1 {
		mw.split.SetOffset(state.Split)
	}

	return nil
}
