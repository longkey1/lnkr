# lnkr

A simple CLI tool for managing hard links and symbolic links with configuration files.

## Overview

`lnkr` helps you manage links between local and remote directories using a `.lnkr.toml` configuration file. It automatically handles git exclusions and supports both hard links and symbolic links.

When you add a file with `lnkr add`, it moves the file to the remote directory and creates a link back to the local location. This is useful for syncing configuration files to cloud storage while keeping them accessible in your project.

## Installation

```bash
# From source
go build -o lnkr .

# Using go install
go install github.com/longkey1/lnkr@latest
```

## Quick Start

```bash
# 1. Initialize project
lnkr init --remote /backup/project

# 2. Add files to link (moves to remote and creates link)
lnkr add important.txt
lnkr add config/

# 3. Check status
lnkr status

# 4. Remove links when done
lnkr unlink

# 5. Re-create links (e.g., after cloning)
lnkr link
```

## Commands

### init
Initialize a new lnkr project.

```bash
# Basic initialization
lnkr init

# With remote directory
lnkr init --remote /path/to/remote

# Create remote directory if it doesn't exist
lnkr init --remote /path/to/remote --with-create-remote

# Custom git exclude path
lnkr init --git-exclude-path .gitignore
```

### add
Add files or directories to the link configuration. This command:
1. Moves the specified file/directory from local to remote
2. Creates a link from remote back to local
3. Updates the GitExclude file

```bash
# Add single file (symbolic link by default)
lnkr add file.txt

# Add with hard link
lnkr add file.txt --type hard

# Add directory (symbolic link)
lnkr add directory/

# Add directory recursively with hard links (for all files)
lnkr add directory/ --type hard --recursive
```

### link
Create links based on configuration. Links are created from remote to local (remote is the source, local is the link target). This is useful when setting up a new machine or after cloning a repository.

```bash
lnkr link
```

### unlink
Remove all links from the filesystem. This will also remove all link paths from the GitExclude file.

```bash
lnkr unlink
```

### status
Check the status of configured links.

```bash
lnkr status
```

### remove
Remove entries from the configuration. This will also update the GitExclude file with the remaining link paths.

```bash
lnkr remove path/to/remove
```

### clean
Remove configuration file and clean up git exclusions.

```bash
lnkr clean
```

## Configuration (.lnkr.toml)

```toml
local = "/workspace"
remote = "/backup/project"
link_type = "symbolic" # or "hard"; default is "symbolic"
git_exclude_path = ".git/info/exclude"

[[links]]
path = "file.txt"
type = "symbolic"

[[links]]
path = "config/"
type = "symbolic"
```

## Environment Variables

- `LNKR_REMOTE_ROOT`: Base directory for remote paths (default: `$HOME/.config/lnkr`)
- `LNKR_REMOTE_DEPTH`: Directory levels to include in default remote path (default: 2)

## Link Types

- **Symbolic Links**: Point to the original file/directory (default, use `--type symbolic` or no flag)
- **Hard Links**: Share the same inode as the original file (use `--type hard`)

Note: Hard links can only be created for files, not directories. Use `--recursive` flag to add all files in a directory as hard links.

## How It Works

1. **Add**: When you run `lnkr add myfile.txt`, the file is moved from `local/myfile.txt` to `remote/myfile.txt`, and a link is created at `local/myfile.txt` pointing to `remote/myfile.txt`.

2. **Link**: When you run `lnkr link` (e.g., on a new machine), it creates links from remote to local for all configured entries.

3. **Unlink**: Removes the links from the local directory (remote files remain intact).

## Platform Support

- Linux (AMD64, ARM64, ARMv6, ARMv7)
- macOS (AMD64, ARM64)

Windows is not supported due to filesystem differences.
