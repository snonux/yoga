package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"yoga/internal/app"
)

func TestRunPrintsVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Yoga version")) {
		t.Fatalf("expected version output, got %s", stdout.String())
	}
}

func TestRunSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := t.TempDir()
	orig := runApp
	runApp = func(opts app.Options) error { return nil }
	defer func() { runApp = orig }()
	code := run([]string{"--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunAppError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := t.TempDir()
	orig := runApp
	runApp = func(app.Options) error { return errors.New("boom") }
	defer func() { runApp = orig }()
	code := run([]string{"--root", root}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("error:")) {
		t.Fatalf("expected error output, got %s", stderr.String())
	}
}

func TestRunDefaultRootCreated(t *testing.T) {
	var stdout, stderr bytes.Buffer
	home := t.TempDir()
	t.Setenv("HOME", home)
	orig := runApp
	runApp = func(opts app.Options) error {
		if _, err := os.Stat(filepath.Join(home, "Yoga")); err != nil {
			t.Fatalf("expected default directory: %v", err)
		}
		return nil
	}
	defer func() { runApp = orig }()
	code := run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestMainUsesExit(t *testing.T) {
	root := t.TempDir()
	origRun := runApp
	origExit := exit
	runApp = func(opts app.Options) error { return nil }
	var code int
	exit = func(c int) { code = c }
	defer func() {
		runApp = origRun
		exit = origExit
	}()
	os.Args = []string{"yoga", "--root", root}
	main()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}
