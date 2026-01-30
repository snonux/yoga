package gui

import (
	"fmt"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	yogaApp "codeberg.org/snonux/yoga/internal/app"
)

type MainWindow struct {
	window       fyne.Window
	app          *App
	content      fyne.CanvasObject
	split        *container.Split
	status       *StatusBar
	videoList    *VideoList
	previewPanel *PreviewPanel
	videos       []*yogaApp.Video
	toolbar      *fyne.Container
	filterDialog *dialog.FormDialog
	tagDialog    *dialog.FormDialog
	cropEnabled  bool
}

func NewMainWindow(app *App, root, cropValue string) *MainWindow {
	window := app.fyneApp.NewWindow("Yoga - " + root)

	mw := &MainWindow{
		window:      window,
		app:         app,
		cropEnabled: cropValue != "",
	}

	mw.status = NewStatusBar()
	mw.videoList = NewVideoList()
	mw.previewPanel = NewPreviewPanel()
	mw.setupCallbacks()
	mw.buildContent()
	mw.setupKeyboardShortcuts()
	mw.buildToolbar()

	window.Resize(fyne.NewSize(1200, 800))
	window.SetCloseIntercept(func() {
		app.Stop()
		window.Close()
	})

	return mw
}

func (m *MainWindow) buildContent() {
	m.content = container.NewBorder(
		m.toolbar,
		m.status.Content(),
		nil,
		nil,
		m.buildMainArea(),
	)
	m.window.SetContent(m.content)
}

func (m *MainWindow) buildMainArea() fyne.CanvasObject {
	m.split = container.NewHSplit(
		m.previewPanel.Content(),
		m.videoList.Content(),
	)
	m.split.SetOffset(0.35)
	return m.split
}

func (m *MainWindow) buildToolbar() {
	filterBtn := widget.NewButton("Filter", m.showFilterDialog)
	sortBtn := widget.NewButton("Sort", m.showSortMenu)
	randomBtn := widget.NewButton("Random", m.selectRandom)
	reindexBtn := widget.NewButton("Re-index", m.reindex)
	refreshBtn := widget.NewButton("Refresh", m.refresh)
	quitBtn := widget.NewButton("Quit", m.quit)

	m.toolbar = container.NewHBox(
		filterBtn,
		sortBtn,
		randomBtn,
		reindexBtn,
		refreshBtn,
		quitBtn,
	)
}

func (m *MainWindow) setupCallbacks() {
	m.videoList.OnSelect(func(video *yogaApp.Video) {
		m.previewPanel.SetVideo(video)
	})

	m.videoList.OnPlay(func(video *yogaApp.Video) {
		m.playVideo(video)
	})

	m.previewPanel.OnPlay(func() {
		video := m.videoList.Selected()
		if video != nil {
			m.playVideo(video)
		}
	})

	m.previewPanel.OnEdit(func() {
		video := m.videoList.Selected()
		if video != nil {
			m.showTagDialog(video)
		}
	})
}

func (m *MainWindow) setupKeyboardShortcuts() {
	canvas := m.window.Canvas()
	canvas.SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyEscape, fyne.KeyQ:
			m.quit()
		case fyne.KeyEnter:
			m.videoList.PlaySelected()
		}
	})
}

func (m *MainWindow) Show() {
	m.window.Show()
	m.loadVideosAsync()
}

func (m *MainWindow) SetStatus(text string) {
	m.status.SetText(text)
}

func (m *MainWindow) UpdateVideos(videos []*yogaApp.Video) {
	m.videos = videos
	m.videoList.SetVideos(videos)
	m.SetStatus(fmt.Sprintf("Loaded %d videos", len(videos)))
	if len(videos) > 0 {
		m.videoList.SelectFirst()
	}
}

func (m *MainWindow) playVideo(video *yogaApp.Video) {
	var args []string
	if m.cropEnabled {
		args = append(args, "--crop", m.app.cropValue)
	}
	args = append(args, video.Path)

	cmd := exec.Command("vlc", args...)
	if err := cmd.Start(); err != nil {
		dialog.ShowError(err, m.window)
		return
	}
	m.SetStatus(fmt.Sprintf("Playing: %s", video.Name))
	go func() {
		cmd.Wait()
	}()
}

func (m *MainWindow) showFilterDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Name contains")

	minEntry := widget.NewEntry()
	minEntry.SetPlaceHolder("Min minutes")

	maxEntry := widget.NewEntry()
	maxEntry.SetPlaceHolder("Max minutes")

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("Tags contain")

	items := []*widget.FormItem{
		{Text: "Name", Widget: nameEntry},
		{Text: "Min Duration", Widget: minEntry},
		{Text: "Max Duration", Widget: maxEntry},
		{Text: "Tags", Widget: tagEntry},
	}

	m.filterDialog = dialog.NewForm("Filter Videos", "Apply", "Cancel", items, func(submitted bool) {
		if submitted {
			m.applyFilters(nameEntry.Text, minEntry.Text, maxEntry.Text, tagEntry.Text)
		}
	}, m.window)

	m.filterDialog.Show()
}

func (m *MainWindow) applyFilters(name, min, max, tags string) {
	m.videoList.Filter(func(video *yogaApp.Video) bool {
		if name != "" && !contains(video.Name, name) {
			return false
		}
		if tags != "" && !containsTags(video.Tags, tags) {
			return false
		}
		return true
	})
	m.SetStatus(fmt.Sprintf("Filter applied"))
}

func (m *MainWindow) showSortMenu() {
	options := []string{
		"Sort by Name",
		"Sort by Duration",
		"Sort by Age",
	}
	entry := widget.NewSelect(options, func(selected string) {})
	entry.SetSelected(options[0])

	dialog.NewForm("Sort Videos", "Apply", "Cancel",
		[]*widget.FormItem{{Text: "Sort by", Widget: entry}},
		func(submitted bool) {
			if submitted {
				m.sortVideos(entry.Selected)
			}
		}, m.window).Show()
}

func (m *MainWindow) sortVideos(sortBy string) {
	m.SetStatus(fmt.Sprintf("Sorted by %s", sortBy))
}

func (m *MainWindow) showTagDialog(video *yogaApp.Video) {
	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Comma-separated tags")

	if len(video.Tags) > 0 {
		tagsStr := ""
		for i, tag := range video.Tags {
			if i > 0 {
				tagsStr += ", "
			}
			tagsStr += tag
		}
		entry.SetText(tagsStr)
	}

	items := []*widget.FormItem{
		{Text: "Tags", Widget: entry},
	}

	m.tagDialog = dialog.NewForm("Edit Tags", "Save", "Cancel", items, func(submitted bool) {
		if submitted {
			m.saveTags(video, entry.Text)
		}
	}, m.window)

	m.tagDialog.Show()
}

func (m *MainWindow) saveTags(video *yogaApp.Video, tagsStr string) {
	tags := parseTags(tagsStr)
	video.Tags = tags
	m.previewPanel.SetVideo(video)
	m.videoList.Refresh()
	m.SetStatus(fmt.Sprintf("Tags saved for %s", video.Name))
}

func (m *MainWindow) selectRandom() {
	m.SetStatus("Random selection")
}

func (m *MainWindow) reindex() {
	m.SetStatus("Re-indexing...")
	m.loadVideosAsync()
}

func (m *MainWindow) refresh() {
	m.SetStatus("Refreshing...")
	m.loadVideosAsync()
}

func (m *MainWindow) quit() {
	m.app.Stop()
	m.window.Close()
}

func (m *MainWindow) loadVideosAsync() {
	m.SetStatus("Loading videos...")
	go func() {
		videos, _, _, err := m.app.Loader().LoadVideos(m.app.Context())
		if err != nil {
			dialog.ShowError(err, m.window)
			return
		}
		videoPtrs := make([]*yogaApp.Video, len(videos))
		for i := range videos {
			videoPtrs[i] = &videos[i]
		}
		m.UpdateVideos(videoPtrs)
	}()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsTags(tags []string, query string) bool {
	for _, tag := range tags {
		if contains(tag, query) {
			return true
		}
	}
	return false
}

func parseTags(s string) []string {
	var tags []string
	parts := strings.Split(s, ",")
	seen := make(map[string]bool)
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" && !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}
