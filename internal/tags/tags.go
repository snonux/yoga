package tags

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PathFor returns the path to the tag metadata file for the given video path.
func PathFor(videoPath string) string {
	ext := filepath.Ext(videoPath)
	if strings.EqualFold(ext, ".mp4") {
		return strings.TrimSuffix(videoPath, ext) + ".json"
	}
	return videoPath + ".json"
}

// Load reads the tags associated with a video. Missing files yield an empty slice.
func Load(videoPath string) ([]string, error) {
	metadataPath := PathFor(videoPath)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var parsed []string
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	return sanitize(parsed), nil
}

// Save persists the tags for a video to its metadata file.
func Save(videoPath string, tagValues []string) error {
	metadataPath := PathFor(videoPath)
	cleaned := sanitize(tagValues)
	payload, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath, payload, 0o644)
}

func sanitize(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(raw))
	var cleaned []string
	for _, tag := range raw {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		normalized := trimmed
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		cleaned = append(cleaned, normalized)
	}
	sort.Strings(cleaned)
	return cleaned
}
