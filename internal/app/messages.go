package app

import "time"

type videosLoadedMsg struct {
	videos   []video
	err      error
	cacheErr error
	pending  []string
	cache    *durationCache
	tagErr   error
}

type playVideoMsg struct {
	path string
	err  error
}

type progressUpdateMsg struct {
	processed int
	total     int
	done      bool
}

type durationUpdateMsg struct {
	path     string
	duration time.Duration
	err      error
}

type tagsSavedMsg struct {
	path string
	tags []string
	err  error
}

type reindexVideosMsg struct{}
