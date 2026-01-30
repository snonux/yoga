package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	yogaApp "codeberg.org/snonux/yoga/internal/app"
)

type VideoList struct {
	container *fyne.Container
	list      *widget.List
	videos    []*yogaApp.Video
	filtered  []*yogaApp.Video
	selected  int
	onSelect  func(*yogaApp.Video)
	onPlay    func(*yogaApp.Video)
}

func NewVideoList() *VideoList {
	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Video"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {},
	)

	vl := &VideoList{
		container: container.NewBorder(nil, nil, nil, nil, list),
		list:      list,
		videos:    make([]*yogaApp.Video, 0),
		filtered:  make([]*yogaApp.Video, 0),
		selected:  -1,
	}

	list.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(vl.filtered) {
			vl.selected = id
			if vl.onSelect != nil {
				vl.onSelect(vl.filtered[id])
			}
		}
	}

	return vl
}

func (vl *VideoList) Content() fyne.CanvasObject {
	return vl.container
}

func (vl *VideoList) SetVideos(videos []*yogaApp.Video) {
	vl.videos = videos
	vl.filtered = videos
	vl.refreshList()
}

func (vl *VideoList) Filter(filterFunc func(*yogaApp.Video) bool) {
	vl.filtered = make([]*yogaApp.Video, 0)
	for _, video := range vl.videos {
		if filterFunc == nil || filterFunc(video) {
			vl.filtered = append(vl.filtered, video)
		}
	}
	vl.refreshList()
}

func (vl *VideoList) SelectFirst() {
	if len(vl.filtered) > 0 {
		vl.list.Select(0)
	}
}

func (vl *VideoList) refreshList() {
	vl.list.Length = func() int { return len(vl.filtered) }
	vl.list.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
		if id < len(vl.filtered) {
			video := vl.filtered[id]
			if hbox, ok := item.(*fyne.Container); ok && len(hbox.Objects) >= 2 {
				if label, ok := hbox.Objects[1].(*widget.Label); ok {
					label.SetText(vl.formatVideo(video))
				}
			}
		}
	}
	vl.list.Refresh()
}

func (vl *VideoList) formatVideo(video *yogaApp.Video) string {
	var parts []string
	parts = append(parts, video.Name)

	if video.Duration > 0 {
		minutes := int(video.Duration.Minutes())
		seconds := int(video.Duration.Seconds()) % 60
		parts = append(parts, fmt.Sprintf("%d:%02d", minutes, seconds))
	}

	if len(video.Tags) > 0 {
		tagsStr := strings.Join(video.Tags, ", ")
		if len(tagsStr) > 30 {
			tagsStr = tagsStr[:30] + "..."
		}
		parts = append(parts, tagsStr)
	}

	return strings.Join(parts, " | ")
}

func (vl *VideoList) OnSelect(f func(*yogaApp.Video)) {
	vl.onSelect = f
}

func (vl *VideoList) OnPlay(f func(*yogaApp.Video)) {
	vl.onPlay = f
}

func (vl *VideoList) PlaySelected() {
	if vl.selected >= 0 && vl.selected < len(vl.filtered) {
		if vl.onPlay != nil {
			vl.onPlay(vl.filtered[vl.selected])
		}
	}
}

func (vl *VideoList) Selected() *yogaApp.Video {
	if vl.selected >= 0 && vl.selected < len(vl.filtered) {
		return vl.filtered[vl.selected]
	}
	return nil
}

func (vl *VideoList) Refresh() {
	vl.list.Refresh()
}
