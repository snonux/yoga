package gui

import (
	"fmt"
	"os"
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
	asyncManager *AsyncManager
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
	mw.asyncManager = NewAsyncManager(app, window)
	mw.setupCallbacks()
	mw.buildContent()
	mw.setupKeyboardShortcuts()
	mw.buildToolbar()

	mw.loadWindowState()

	window.Resize(fyne.NewSize(1200, 800))
	window.SetCloseIntercept(func() {
		mw.saveWindowState()
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
		case fyne.KeyF, fyne.KeySlash:
			m.showFilterDialog()
		case fyne.KeyT:
			video := m.videoList.Selected()
			if video != nil {
				m.showTagDialog(video)
			}
		case fyne.KeyN:
			m.sortVideos("Sort by Name")
		case fyne.KeyL:
			m.sortVideos("Sort by Duration")
		case fyne.KeyA:
			m.sortVideos("Sort by Age")
		case fyne.KeyR:
			m.selectRandom()
		case fyne.KeyI:
			m.reindex()
		case fyne.KeyC:
			m.toggleCrop()
		case fyne.KeyDelete:
			m.resetFilters()
		case fyne.KeyH:
			m.refresh()
		}
	})
}

func (m *MainWindow) Show() {
	m.checkDependencies()
	m.window.Show()
	m.loadVideosAsync()
}

func (m *MainWindow) checkDependencies() {
	if _, err := exec.LookPath("vlc"); err != nil {
		dialog.ShowInformation("Missing Dependency", "VLC is not installed or not in PATH. Video playback will not work.", m.window)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		dialog.ShowInformation("Missing Dependency", "ffmpeg is not installed or not in PATH. Thumbnail generation will not work.", m.window)
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		dialog.ShowInformation("Missing Dependency", "ffprobe is not installed or not in PATH. Duration detection may not work.", m.window)
	}
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
	if len(m.videos) == 0 {
		m.SetStatus("No videos to select from")
		return
	}
	m.videoList.SelectRandom()
	selected := m.videoList.Selected()
	if selected != nil {
		m.SetStatus(fmt.Sprintf("Randomly selected: %s", selected.Name))
	}
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
	m.status.ShowProgress()
	m.status.SetProgress(0)
	m.SetStatus("Loading videos...")

	m.asyncManager.RunAsync(func() UpdateCallback {
		videos, pendingDuration, pendingThumbnail, err := m.app.Loader().LoadVideos(m.app.Context())
		if err != nil {
			return func() {
				dialog.ShowError(err, m.window)
				m.status.HideProgress()
				m.SetStatus("Error loading videos")
			}
		}

		m.generatePendingThumbnailsAsync(pendingThumbnail, videos)

		videoPtrs := make([]*yogaApp.Video, len(videos))
		for i := range videos {
			videoPtrs[i] = &videos[i]
		}

		return func() {
			m.UpdateVideos(videoPtrs)
			m.status.HideProgress()
			status := fmt.Sprintf("Loaded %d videos", len(videos))
			if len(pendingDuration) > 0 {
				status += fmt.Sprintf(", %d duration pending", len(pendingDuration))
			}
			if len(pendingThumbnail) > 0 {
				status += fmt.Sprintf(", %d thumbnails pending", len(pendingThumbnail))
			}
			m.SetStatus(status)
		}
	})
}

func (m *MainWindow) generatePendingThumbnailsAsync(pendingPaths []string, videos []yogaApp.Video) {
	if len(pendingPaths) == 0 {
		return
	}

	videoMap := make(map[string]*yogaApp.Video)
	for i := range videos {
		videoMap[videos[i].Path] = &videos[i]
	}

	for i, path := range pendingPaths {
		path := path
		idx := i
		m.asyncManager.RunAsync(func() UpdateCallback {
			video := videoMap[path]
			if video == nil {
				return nil
			}

			info, err := os.Stat(path)
			if err != nil {
				return func() {
					fmt.Printf("Error getting file info for %s: %v\n", path, err)
				}
			}

			_, err = m.app.Loader().GenerateThumbnail(m.app.Context(), path, info)
			if err != nil {
				return func() {
					fmt.Printf("Thumbnail generation error for %s: %v\n", path, err)
				}
			}

			return func() {
				progress := float64(idx+1) / float64(len(pendingPaths))
				m.status.SetProgress(progress)
				video.ThumbnailGenerated = true
				m.videoList.Refresh()
				if m.videoList.Selected() == video {
					m.previewPanel.SetVideo(video)
				}
			}
		})
	}
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

func (m *MainWindow) toggleCrop() {
	if m.app.cropValue == "" {
		dialog.ShowInformation("Crop", "No crop value set (start with --crop)", m.window)
		return
	}
	m.cropEnabled = !m.cropEnabled
	if m.cropEnabled {
		m.SetStatus(fmt.Sprintf("Crop enabled (%s)", m.app.cropValue))
	} else {
		m.SetStatus("Crop disabled")
	}
}

func (m *MainWindow) resetFilters() {
	m.videoList.Filter(nil)
	m.SetStatus(fmt.Sprintf("Filters cleared (%d videos)", len(m.videos)))
}
