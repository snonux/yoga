package app

import "time"

type Video struct {
	Name               string
	Path               string
	Duration           time.Duration
	ModTime            time.Time
	Size               int64
	Err                error
	Tags               []string
	Thumbnail          string
	ThumbnailGenerated bool
}
