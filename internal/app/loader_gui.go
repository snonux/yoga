package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"codeberg.org/snonux/yoga/internal/tags"
	"codeberg.org/snonux/yoga/internal/thumbnail"
)

type Loader struct {
	root           string
	durationCache  *durationCache
	thumbnailCache *thumbnail.Cache
	generator      *thumbnail.Generator
}

func NewLoader(root string, durationCachePath string) *Loader {
	durationCache, _ := loadDurationCache(durationCachePath)
	return &Loader{
		root:           root,
		durationCache:  durationCache,
		thumbnailCache: thumbnail.NewCache(root),
		generator:      thumbnail.NewGenerator(),
	}
}

func (l *Loader) LoadVideos(ctx context.Context) ([]video, []string, []string, error) {
	paths, err := collectVideoPathsForLoader(ctx, l.root)
	if err != nil {
		return nil, nil, nil, err
	}

	videos := make([]video, 0, len(paths))
	durationPending := make([]string, 0)
	thumbnailPending := make([]string, 0)
	var tagErrors []string

	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			videos = append(videos, video{
				Name: filepath.Base(path),
				Path: path,
				Err:  statErr,
				Tags: []string{},
			})
			continue
		}

		dur := cachedDuration(l.durationCache, path, info)
		if dur == 0 {
			durationPending = append(durationPending, path)
		}

		thumbPath, hasThumb := l.checkThumbnail(path, info)
		if !hasThumb {
			thumbnailPending = append(thumbnailPending, path)
		}

		tagList, tagErr := tags.Load(path)
		if tagErr != nil {
			tagErrors = append(tagErrors, fmt.Sprintf("%s: %v", filepath.Base(path), tagErr))
		}

		videos = append(videos, video{
			Name:               filepath.Base(path),
			Path:               path,
			Duration:           dur,
			ModTime:            info.ModTime(),
			Size:               info.Size(),
			Tags:               tagList,
			Thumbnail:          thumbPath,
			ThumbnailGenerated: hasThumb,
		})
	}

	sort.Strings(durationPending)
	sort.Strings(thumbnailPending)
	sort.Strings(tagErrors)

	return videos, durationPending, thumbnailPending, joinErrors(tagErrors)
}

func (l *Loader) checkThumbnail(videoPath string, info os.FileInfo) (string, bool) {
	if l.thumbnailCache == nil {
		return "", false
	}

	return l.thumbnailCache.Lookup(videoPath, info.ModTime())
}

func (l *Loader) GenerateThumbnail(ctx context.Context, videoPath string, modTime os.FileInfo) (string, error) {
	if l.thumbnailCache == nil {
		return "", fmt.Errorf("thumbnail cache not initialized")
	}

	thumbPath, err := l.generator.Generate(ctx, videoPath)
	if err != nil {
		return "", err
	}

	if err := l.thumbnailCache.Store(videoPath, modTime.ModTime(), thumbPath); err != nil {
		return "", fmt.Errorf("store thumbnail cache: %w", err)
	}

	return thumbPath, nil
}

func collectVideoPathsForLoader(ctx context.Context, root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if isVideo(root) {
			return []string{root}, nil
		}
		return nil, nil
	}

	paths, err := collectVideoPaths(root)
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}
