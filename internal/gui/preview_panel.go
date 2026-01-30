package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	yogaApp "codeberg.org/snonux/yoga/internal/app"
)

type PreviewPanel struct {
	container *fyne.Container
	card      *widget.Card
	nameLabel *widget.Label
	metaLabel *widget.Label
	tagsLabel *widget.Label
	playBtn   *widget.Button
	editBtn   *widget.Button
	video     *yogaApp.Video
	onPlay    func()
	onEdit    func()
}

func NewPreviewPanel() *PreviewPanel {
	nameLabel := widget.NewLabel("No video selected")
	metaLabel := widget.NewLabel("")
	tagsLabel := widget.NewLabel("")
	playBtn := widget.NewButton("▶ Play", nil)
	editBtn := widget.NewButton("✏ Edit Tags", nil)

	playBtn.Disable()
	editBtn.Disable()

	content := container.NewVBox(
		nameLabel,
		widget.NewSeparator(),
		metaLabel,
		widget.NewSeparator(),
		tagsLabel,
		widget.NewSeparator(),
		container.NewHBox(playBtn, editBtn),
	)

	card := widget.NewCard("Preview", "", content)

	panel := &PreviewPanel{
		container: container.NewVBox(card),
		card:      card,
		nameLabel: nameLabel,
		metaLabel: metaLabel,
		tagsLabel: tagsLabel,
		playBtn:   playBtn,
		editBtn:   editBtn,
	}

	panel.playBtn.OnTapped = panel.play
	panel.editBtn.OnTapped = panel.edit

	return panel
}

func (p *PreviewPanel) Content() fyne.CanvasObject {
	return p.container
}

func (p *PreviewPanel) SetVideo(video *yogaApp.Video) {
	p.video = video
	p.updateUI()
}

func (p *PreviewPanel) updateUI() {
	if p.video == nil {
		p.nameLabel.SetText("No video selected")
		p.metaLabel.SetText("")
		p.tagsLabel.SetText("")
		p.playBtn.Disable()
		p.editBtn.Disable()
		return
	}

	p.nameLabel.SetText(p.video.Name)
	p.metaLabel.SetText(p.formatMetadata())
	p.tagsLabel.SetText(p.formatTags())
	p.playBtn.Enable()
	p.editBtn.Enable()
}

func (p *PreviewPanel) formatMetadata() string {
	duration := p.video.Duration
	if duration > 0 {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60
		return fmt.Sprintf("Duration: %d:%02d:%02d | Size: %.1f MB",
			hours, minutes, seconds, float64(p.video.Size)/(1024*1024))
	}
	return "Duration: unknown | Size: unknown"
}

func (p *PreviewPanel) formatTags() string {
	if len(p.video.Tags) == 0 {
		return "Tags: none"
	}
	return "Tags: " + fmt.Sprintf("%v", p.video.Tags)
}

func (p *PreviewPanel) play() {
	if p.onPlay != nil && p.video != nil {
		p.onPlay()
	}
}

func (p *PreviewPanel) edit() {
	if p.onEdit != nil && p.video != nil {
		p.onEdit()
	}
}

func (p *PreviewPanel) OnPlay(f func()) {
	p.onPlay = f
}

func (p *PreviewPanel) OnEdit(f func()) {
	p.onEdit = f
}
