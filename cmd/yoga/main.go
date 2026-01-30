package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"codeberg.org/snonux/yoga/internal/fsutil"
	"codeberg.org/snonux/yoga/internal/gui"
	"codeberg.org/snonux/yoga/internal/meta"
)

const defaultRoot = "~/Yoga"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
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
	app := gui.NewApp(root, *cropFlag)
	app.Run()
	return 0
}
