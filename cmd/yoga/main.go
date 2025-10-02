package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"yoga/internal/app"
	"yoga/internal/fsutil"
	"yoga/internal/meta"
)

const defaultRoot = "~/Yoga"

var (
	runApp = app.Run
	exit   = os.Exit
)

func main() {
	exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yoga", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rootFlag := fs.String("root", "", "Directory containing yoga videos (default ~/Yoga)")
	cropFlag := fs.String("crop", "", "Optional crop aspect for VLC (e.g. 5:4)")
	versionFlag := fs.Bool("version", false, "Print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *versionFlag {
		fmt.Fprintf(stdout, "Yoga version %s\n", meta.Version)
		return 0
	}
	root, err := fsutil.ResolveRootPath(*rootFlag, defaultRoot)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	opts := app.Options{Root: root, Crop: strings.TrimSpace(*cropFlag)}
	if err := runApp(opts); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}
