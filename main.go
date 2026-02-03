package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	pattern  string
	debounce int
)

func main() {
	flag.StringVar(&pattern, "pattern", "*", "Glob pattern to match files (e.g., '*.go', '**/*.js')")
	flag.StringVar(&pattern, "p", "*", "Glob pattern to match files (shorthand)")
	flag.IntVar(&debounce, "debounce", 300, "Debounce time in milliseconds")
	flag.IntVar(&debounce, "d", 300, "Debounce time in milliseconds (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] command...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Watch files and execute commands on changes\n\n")
		fmt.Fprintf(os.Stderr, "watchgo monitors files in the current directory and subdirectories.\n")
		fmt.Fprintf(os.Stderr, "When a file matching the glob pattern changes, it executes the specified command.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --pattern \"**/*.go\" go test ./...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -p \"*.js\" npm test\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -p \"src/**/*.ts\" -d 500 npm run build\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	run(args)
}

func run(args []string) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	fmt.Printf("Watching directory: %s\n", cwd)
	fmt.Printf("Pattern: %s\n", pattern)
	fmt.Printf("Command: %s\n", strings.Join(args, " "))
	fmt.Printf("Debounce: %dms\n\n", debounce)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create watcher:", err)
	}
	defer watcher.Close()

	// Add all directories recursively
	if err := addDirsRecursively(watcher, cwd); err != nil {
		log.Fatal("Failed to add directories:", err)
	}

	// Create debouncer
	debouncer := newDebouncer(time.Duration(debounce) * time.Millisecond)

	// Watch for events
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle directory creation
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						addDirsRecursively(watcher, event.Name)
						fmt.Printf("New directory added to watch: %s\n", event.Name)
					}
				}

				// Check if event is a write, create, or remove
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					relPath, err := filepath.Rel(cwd, event.Name)
					if err != nil {
						relPath = event.Name
					}

					// Check if file matches pattern
					if matchPattern(relPath, pattern) {
						fmt.Printf("Change detected: %s (%s)\n", relPath, event.Op)
						debouncer.trigger(func() {
							executeCommand(args)
						})
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	fmt.Println("Watching for changes... Press Ctrl+C to stop")
	<-done
}

// addDirsRecursively adds the directory and all subdirectories to the watcher
func addDirsRecursively(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

// matchPattern checks if the file path matches the glob pattern
func matchPattern(path, pattern string) bool {
	// Handle double-star pattern for recursive matching
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check prefix
			if prefix != "" && !strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")) {
				return false
			}

			// Check suffix using glob match
			if suffix != "" {
				matched, _ := filepath.Match(suffix, filepath.Base(path))
				return matched
			}
			return true
		}
	}

	// Regular glob match
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err != nil {
		return false
	}
	return matched
}

// executeCommand runs the specified command
func executeCommand(args []string) {
	fmt.Printf("\n▶ Executing: %s\n", strings.Join(args, " "))
	fmt.Println(strings.Repeat("─", 60))

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	fmt.Println(strings.Repeat("─", 60))
	if err != nil {
		fmt.Printf("✗ Command failed after %s: %v\n\n", elapsed.Round(time.Millisecond), err)
	} else {
		fmt.Printf("✓ Command completed successfully in %s\n\n", elapsed.Round(time.Millisecond))
	}
}

// debouncer prevents rapid repeated executions
type debouncer struct {
	duration time.Duration
	timer    *time.Timer
	callback func()
}

func newDebouncer(duration time.Duration) *debouncer {
	return &debouncer{
		duration: duration,
	}
}

func (d *debouncer) trigger(callback func()) {
	if d.timer != nil {
		d.timer.Stop()
	}

	d.callback = callback
	d.timer = time.AfterFunc(d.duration, func() {
		if d.callback != nil {
			d.callback()
		}
	})
}
