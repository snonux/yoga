//go:build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var Default = Build

// Build compiles the yoga binary.
func Build() error {
	return run("go", "build", "./cmd/yoga")
}

// Test runs the unit test suite.
func Test() error {
	return run("go", "test", "./...")
}

// Run runs the unit test suite.
func Run() error {
	return run("go", "run", "./cmd/yoga")
}

// Install installs the yoga binary into GOPATH/bin or GOBIN.
func Install() error {
	return run("go", "install", "./cmd/yoga")
}

// Coverage runs the unit tests with coverage and enforces the minimum target.
func Coverage() error {
	profile := filepath.Join(os.TempDir(), "yoga-coverage.out")
	if err := run("go", "test", "-coverprofile="+profile, "./..."); err != nil {
		return err
	}
	defer os.Remove(profile)
	out, err := exec.Command("go", "tool", "cover", "-func="+profile).CombinedOutput()
	if err != nil {
		fmt.Print(string(out))
		return err
	}
	fmt.Print(string(out))
	total, err := parseTotalCoverage(string(out))
	if err != nil {
		return err
	}
	if total < 85.0 {
		return fmt.Errorf("coverage %.1f%% below required 85%%", total)
	}
	return nil
}

func parseTotalCoverage(report string) (float64, error) {
	lines := strings.Split(strings.TrimSpace(report), "\n")
	if len(lines) == 0 {
		return 0, errors.New("empty coverage report")
	}
	last := lines[len(lines)-1]
	fields := strings.Fields(last)
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected coverage line: %s", last)
	}
	value := strings.TrimSuffix(fields[len(fields)-1], "%")
	percent, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return percent, nil
}

func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
