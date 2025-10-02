package fsutil

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestResolveRootPathCreatesDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, err := ResolveRootPath("", "~/Yoga")
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	expected := filepath.Join(home, "Yoga")
	if root != expected {
		t.Fatalf("expected %s, got %s", expected, root)
	}
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("stat default: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory at %s", expected)
	}
}

func TestResolveRootPathRequiresExisting(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing")
	if _, err := ResolveRootPath(missing, "~/Yoga"); err == nil {
		t.Fatalf("expected error for %s", missing)
	}
}

func TestResolveRootPathAllowsFile(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "video.mp4")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	got, err := ResolveRootPath(file, "~/Yoga")
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if got != file {
		t.Fatalf("expected file path returned, got %s", got)
	}
}

func TestExpandPathWithHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := filepath.Join(home, "custom")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := ResolveRootPath("~/custom", "~/Yoga")
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	expected := filepath.Join(home, "custom")
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
	if v, err := expandPath("~"); err != nil || v != home {
		t.Fatalf("expandPath ~ failed: %v %s", err, v)
	}
	if _, err := expandPath("~no_such_user/foo"); err == nil {
		t.Fatalf("expected error for unknown user")
	}
	if path, err := expandPath("relative/path"); err != nil || path != "relative/path" {
		t.Fatalf("expected relative path unchanged, got %s %v", path, err)
	}
	if current, err := user.Current(); err == nil {
		value := "~" + current.Username
		if p, err := expandPath(value); err != nil || p != current.HomeDir {
			t.Fatalf("expected home dir %s, got %s (%v)", current.HomeDir, p, err)
		}
	}
}

func TestSplitUserPath(t *testing.T) {
	user, rest := splitUserPath("~alice/videos")
	if user != "alice" || rest != "/videos" {
		t.Fatalf("unexpected split %s %s", user, rest)
	}
	user, rest = splitUserPath("~bob")
	if user != "bob" || rest != "" {
		t.Fatalf("unexpected split %s %s", user, rest)
	}
}

func TestNormalizeRootInput(t *testing.T) {
	const fallback = "~/Yoga"
	value, isDefault := normalizeRootInput("", fallback)
	if !isDefault || value != fallback {
		t.Fatalf("unexpected normalize result %s %v", value, isDefault)
	}
	value, isDefault = normalizeRootInput(" /tmp ", fallback)
	if isDefault || value != "/tmp" {
		t.Fatalf("unexpected normalize result %s %v", value, isDefault)
	}
}

func TestEnsureRootExistsErrors(t *testing.T) {
	if _, err := ensureRootExists(filepath.Join(t.TempDir(), "missing"), false); err == nil {
		t.Fatalf("expected error when creation not allowed")
	}
}
