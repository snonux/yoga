package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type StatusBar struct {
	label    *widget.Label
	progress *widget.ProgressBar
	content  fyne.CanvasObject
}

func NewStatusBar() *StatusBar {
	label := widget.NewLabel("Ready")
	progress := widget.NewProgressBar()
	progress.Hide()

	content := container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.NewVBox(
			label,
			progress,
		),
	)

	return &StatusBar{
		label:    label,
		progress: progress,
		content:  content,
	}
}

func (s *StatusBar) Content() fyne.CanvasObject {
	return s.content
}

func (s *StatusBar) SetText(text string) {
	s.label.SetText(text)
}

func (s *StatusBar) ShowProgress() {
	s.progress.Show()
}

func (s *StatusBar) HideProgress() {
	s.progress.Hide()
}

func (s *StatusBar) SetProgress(value float64) {
	s.progress.SetValue(value)
}
