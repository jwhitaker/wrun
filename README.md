# wrun

A simple file watcher that executes commands when files change.

## Features

- 📁 Recursively watches current directory and subdirectories
- 🎯 Glob pattern matching for file filtering
- ⚡ Debouncing to prevent rapid repeated executions
- 🔄 Automatically watches new directories as they're created
- 🎨 Clean output with execution timing

## Installation

```bash
go install github.com/jwhitaker/wrun@latest
```

Or build from source:

```bash
git clone https://github.com/jwhitaker/wrun.git
cd wrun
go build
```

## Usage

```bash
wrun [flags] -- [command to execute]
```

### Flags

- `-p, --pattern`: Glob pattern to match files (default: `*`)
- `-d, --debounce`: Debounce time in milliseconds (default: `300`)

### Examples

Watch all Go files and run tests:
```bash
wrun --pattern "**/*.go" -- go test ./...
```

Watch JavaScript files and run build:
```bash
wrun -p "*.js" -- npm run build
```

Watch TypeScript files in src directory with custom debounce:
```bash
wrun -p "src/**/*.ts" -d 500 -- npm run build
```

Watch all files (default pattern):
```bash
wrun -- make build
```

## Glob Patterns

- `*` - Matches any files in the current directory
- `*.go` - Matches all Go files in any directory
- `**/*.js` - Matches all JavaScript files in any subdirectory
- `src/**/*.ts` - Matches TypeScript files under src directory

## How It Works

1. Watches the current directory and all subdirectories for file changes
2. When a file is created, modified, or deleted, checks if it matches the glob pattern
3. If it matches, waits for the debounce period (to batch rapid changes)
4. Executes the specified command
5. Displays the command output and execution time

## Notes

- Hidden directories (starting with `.`) are automatically excluded from watching
- New directories created during watching are automatically added
- The command receives full access to stdin/stdout/stderr
