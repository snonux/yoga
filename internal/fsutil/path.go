package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ResolveRootPath expands and validates the supplied root path. When the
// caller did not specify a value, defaultValue is used and created on demand.
func ResolveRootPath(input, defaultValue string) (string, error) {
	value, isDefault := normalizeRootInput(input, defaultValue)
	expanded, err := expandPath(value)
	if err != nil {
		return "", fmt.Errorf("cannot expand root path %q: %w", value, err)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("cannot resolve root path %q: %w", expanded, err)
	}
	info, err := ensureRootExists(abs, isDefault)
	if err != nil {
		return "", err
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return "", fmt.Errorf("root path %q is not a file or directory", abs)
	}
	return abs, nil
}

func normalizeRootInput(input, defaultValue string) (value string, isDefault bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return defaultValue, true
	}
	return trimmed, false
}

func ensureRootExists(path string, allowCreate bool) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot access root path %q: %w", path, err)
	}
	if !allowCreate {
		return nil, fmt.Errorf("root path does not exist: %s", path)
	}
	if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
		return nil, fmt.Errorf("cannot create default directory %q: %w", path, mkErr)
	}
	info, err = os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot stat default directory %q: %w", path, err)
	}
	return info, nil
}

func expandPath(p string) (string, error) {
	if p == "" || p[0] != '~' {
		return p, nil
	}
	if len(p) == 1 {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if p[1] == '/' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	username, rest := splitUserPath(p)
	usr, err := user.Lookup(username)
	if err != nil {
		return "", err
	}
	if rest == "" {
		return usr.HomeDir, nil
	}
	return filepath.Join(usr.HomeDir, rest), nil
}

func splitUserPath(p string) (string, string) {
	sep := strings.IndexRune(p, '/')
	if sep == -1 {
		return p[1:], ""
	}
	return p[1:sep], p[sep:]
}
