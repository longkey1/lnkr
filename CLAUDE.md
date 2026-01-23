# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`lnkr` is a Go CLI tool for managing hard links and symbolic links with configuration files. It helps sync files between local and remote directories (e.g., backing up config files to cloud storage) while maintaining links in the local location.

## Development Commands

### Building
```bash
make build          # Build binary to ./bin/lnkr
make build-dev      # Build with dev version info (includes commit SHA, build time)
```

### Testing
```bash
make test           # Run all tests
go test ./...       # Standard Go test command
go test -v ./internal/lnkr  # Run tests in specific package with verbose output
```

### Code Quality
```bash
make fmt            # Format code with gofmt
make fmt-check      # Check formatting without modifying files
make lint           # Run golangci-lint
```

### Release Management
```bash
make release type=patch dryrun=true   # Preview patch version bump
make release type=minor dryrun=false  # Create and push minor version
make release type=major dryrun=false  # Create and push major version

make re-release tag=v1.0.0 dryrun=false  # Re-release specific version
make release-dry-run                      # Test goreleaser locally
```

When a version tag is pushed, GitHub Actions automatically builds binaries via GoReleaser.

## Architecture

### Package Structure

- **`cmd/`** - Cobra command definitions (CLI interface layer)
  - Each command file (init.go, add.go, link.go, etc.) defines CLI flags and calls internal/lnkr functions
  - Uses cobra for command structure and flag parsing

- **`internal/lnkr/`** - Core business logic
  - **Configuration management**: `config.go`, `globalconfig.go`
  - **Path handling**: `pathvar.go` - Critical for variable expansion/contraction
  - **Operations**: `add.go`, `link.go`, `unlink.go`, `switch.go`, `remove.go`, `status.go`, `clean.go`, `init.go`

- **`internal/version/`** - Version information (set via ldflags during build)

### Key Architectural Concepts

#### Configuration Hierarchy
The tool uses a two-tier config system:

1. **Global config** (`~/.config/lnkr/config.toml`) - viper-based, provides defaults
   - Sets `remote_root`, `local_root`, `link_type`, `git_exclude_path`
   - Priority: Environment variables > Config file > Hardcoded defaults

2. **Project config** (`.lnkr.toml`) - TOML-based, per-project settings
   - Stores local/remote paths and link entries
   - Automatically created as a symlink to remote during `init`

#### Path Variable System (`pathvar.go`)
Critical for portability across machines. Two key functions:

- **`ExpandPath()`** - Converts stored paths to absolute paths at runtime
  - Handles placeholders: `{{remote_root}}`, `{{local_root}}`
  - Handles env vars: `$HOME`, `$PWD`, `$LNKR_REMOTE_ROOT`, etc.
  - Priority: `{{placeholders}}` (env > config > default) > `$ENV_VARS`
  - Returns error if any variable is undefined

- **`ContractPath()`** - Converts absolute paths to portable format for storage
  - Replaces path prefixes with variables (longest match wins)
  - Prefers `{{remote_root}}` and `{{local_root}}` over `$HOME` for lnkr paths
  - Used when saving paths to `.lnkr.toml` during `init` and `add`

This system ensures `.lnkr.toml` files work across different machines/users.

#### Link Direction
**Important**: Links point FROM remote TO local (remote = source, local = link target).

- `add` operation: Moves file from local to remote, then creates link at local pointing to remote
- `link` operation: Creates links from remote (source) to local (target)
- This allows remote to be the "source of truth" backed up to cloud storage

#### Git Integration
Automatically manages git exclusions via `.git/info/exclude` (or custom path):
- Adds `### LNKR START` / `### LNKR END` section markers
- Updates exclusions when links are added/removed
- Cleaned up during `unlink` and `clean` operations

### Testing
Tests exist for core logic modules:
- `config_test.go` - Configuration loading/saving
- `pathvar_test.go` - Path expansion/contraction (critical for portability)
- `switch_test.go` - Link type switching logic

Run tests with `make test` or `go test ./...` for all tests.
