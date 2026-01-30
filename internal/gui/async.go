package gui

import (
	"fyne.io/fyne/v2"
)

type UpdateCallback func()

type AsyncManager struct {
	app    *App
	window fyne.Window
}

func NewAsyncManager(app *App, window fyne.Window) *AsyncManager {
	return &AsyncManager{
		app:    app,
		window: window,
	}
}

func (a *AsyncManager) RunAsync(fn func() UpdateCallback) {
	go func() {
		updateFn := fn()
		if updateFn != nil {
			a.RunOnUIThread(updateFn)
		}
	}()
}

func (a *AsyncManager) RunOnUIThread(fn UpdateCallback) {
	fn()
}
