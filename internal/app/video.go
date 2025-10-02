package app

import "time"

type video struct {
	Name     string
	Path     string
	Duration time.Duration
	ModTime  time.Time
	Size     int64
	Err      error
}
