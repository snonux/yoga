package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	yogaApp "codeberg.org/snonux/yoga/internal/app"
)

type App struct {
	fyneApp   fyne.App
	mainApp   *MainWindow
	loader    *yogaApp.Loader
	root      string
	cropValue string
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewApp(root, cropValue string) *App {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(fyne.CurrentApp().Settings().Theme())

	ctx, cancel := context.WithCancel(context.Background())

	cachePath := fmt.Sprintf("%s/.video_duration_cache.json", root)
	loader := yogaApp.NewLoader(root, cachePath)

	return &App{
		fyneApp:   fyneApp,
		loader:    loader,
		root:      root,
		cropValue: cropValue,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (a *App) Run() {
	a.mainApp = NewMainWindow(a, a.root, a.cropValue)
	a.mainApp.Show()
	a.fyneApp.Run()
}

func (a *App) Stop() {
	a.cancel()
}

func (a *App) Context() context.Context {
	return a.ctx
}

func (a *App) Loader() *yogaApp.Loader {
	return a.loader
}
