# Specification: dotsaver TUI

**Date:** 2026-10-15  
**Status:** Approved for Implementation Planning  
**Target:** Go 1.22+ cross-platform TUI application (Windows & Linux)

---

## 1. Executive Summary & Goals

`dotsaver` is a cross-platform Terminal User Interface (TUI) tool written in Go for managing, backing up, and synchronizing dotfiles and application configuration files across multiple operating systems (specifically Windows and Linux) through a remote Git repository.

### Key Goals
- **Remote Git Independence:** Set target remote Git repository; rely on system `git` CLI to inherit existing SSH keys, GPG signing, and credential managers without custom credential prompts or secret storage.
- **Multi-OS Configuration Mapping:** Support tracking arbitrary applications by name, with target file or directory paths explicitly mapped per operating system (`windows` or `linux`).
- **Versioned Remote Manifest:** The manifest (`dotsaver.json`) and tracked files are stored together in the remote Git repository.
- **Sync (Pull & Apply):** Pull latest repository commits and copy the files appropriate for the current operating system to their specified local paths, with automatic backup (`.bak`) when local changes conflict.
- **Save (Collect & Push):** Read local config files/directories according to the manifest, copy them into the repository cache, commit, and push to the remote repository.
- **Aesthetic & Minimalist TUI:** Built using the Charm ecosystem (`bubbletea`, `lipgloss`, `bubbles`) to provide an intuitive, responsive, and distraction-free terminal dashboard.

---

## 2. System Architecture

The project follows a clean, decoupled Go package layout where all business logic (Git, Syncer, Manifest, Config) is independent of the TUI presentation layer.

```text
cmd/
  dotsaver/
    main.go                   # Application entry point, CLI flags, initializes TUI
pkg/
  config/
    config.go                 # Local app config (remote URL, repo cache path)
  manifest/
    manifest.go               # dotsaver.json data structures, parser, serializer
  git/
    git.go                    # Shells out to system git CLI (clone, pull, commit, push, status)
  syncer/
    path.go                   # Environment variable & tilde path expansion
    copy.go                   # Recursive file and directory copy routines
    syncer.go                 # Save, Sync, Conflict detection (SHA-256), Backup (.bak)
  tui/
    app.go                    # Root Bubble Tea model, navigation, view switching
    styles.go                 # Lip Gloss colors, borders, layouts
    views/
      dashboard.go            # Overview screen, status badges, quick actions
      programs.go             # Program & path manager list and entry forms
      settings.go             # Remote URL & cache path editor
      modal_conflict.go       # Review & confirmation dialog for sync conflicts
      modal_spinner.go        # Async progress spinner for network/git operations
```

---

## 3. Configuration & Repository Layout

### 3.1 Local Application Configuration (`config.json`)

Stored in the OS-standard config directory:
- **Windows:** `%APPDATA%\dotsaver\config.json`
- **Linux:** `~/.config/dotsaver/config.json`

Schema:
```json
{
  "remote_url": "git@github.com:username/my-dotfiles.git",
  "repo_path": ""
}
```
* If `repo_path` is empty, defaults to:
  * **Windows:** `%LOCALAPPDATA%\dotsaver\repo`
  * **Linux:** `~/.local/share/dotsaver/repo`

### 3.2 Remote Repository Manifest (`dotsaver.json`)

Stored at the root of the remote Git repository:
```json
{
  "version": 1,
  "programs": [
    {
      "name": "nvim",
      "targets": [
        {
          "os": "windows",
          "path": "%LOCALAPPDATA%/nvim",
          "alias": "nvim"
        },
        {
          "os": "linux",
          "path": "~/.config/nvim",
          "alias": "nvim"
        }
      ]
    },
    {
      "name": "git",
      "targets": [
        {
          "os": "windows",
          "path": "~/.gitconfig",
          "alias": ".gitconfig"
        },
        {
          "os": "linux",
          "path": "~/.gitconfig",
          "alias": ".gitconfig"
        }
      ]
    }
  ]
}
```

* `version`: Schema version (integer, currently `1`).
* `programs`: Array of program specifications.
* `targets`: Array of target mappings for a program:
  * `os`: Target operating system (`"windows"` or `"linux"`).
  * `path`: Path to file or directory. Supports `~` and environment variables.
  * `alias`: Subdirectory/file name under `data/<program>/<os>/` (defaults to `filepath.Base(path)`).

### 3.3 Remote Repository Directory Structure

```text
<git-repo-root>/
├── dotsaver.json
└── data/
    ├── nvim/
    │   ├── windows/
    │   │   └── nvim/
    │   │       ├── init.lua
    │   │       └── lua/
    │   └── linux/
    │       └── nvim/
    │           ├── init.lua
    │           └── lua/
    └── git/
        ├── windows/
        │   └── .gitconfig
        └── linux/
            └── .gitconfig
```

---

## 4. Component Details & Data Flow

### 4.1 Git Engine (`pkg/git`)
Wraps system `git` using `os/exec`.
- **`EnsureRepo(remoteURL, repoPath)`**:
  - Checks if `repoPath/.git` exists.
  - If not: runs `git clone <remoteURL> <repoPath>`. If the remote is empty, initializes repo with `git init`, `git remote add origin <remoteURL>`, creates an initial `dotsaver.json`, and branches to `main`.
  - If exists: verifies origin remote matches `remoteURL`, updating with `git remote set-url origin <remoteURL>` if needed.
- **`Pull(repoPath)`**:
  - Runs `git pull --rebase origin HEAD`.
- **`CommitAndPush(repoPath, commitMsg)`**:
  - Runs `git add -A`.
  - Runs `git commit -m "<commitMsg>"`.
  - Runs `git push origin HEAD`.
- **`Status(repoPath)`**:
  - Runs `git status --porcelain` to report clean/dirty working tree.

### 4.2 Path Expander (`pkg/syncer/path.go`)
- Replaces leading `~` or `~/` with `os.UserHomeDir()`.
- Windows environment variables: matches `%VAR%` and replaces via `os.Getenv(VAR)`.
- Linux/Unix environment variables: matches `$VAR` / `${VAR}` via `os.ExpandEnv`.
- Cleans and returns OS-native absolute path (`filepath.Clean`).

### 4.3 Synchronization Engine (`pkg/syncer/syncer.go`)

#### Save Workflow:
1. Validates that `dotsaver.json` exists in `repoPath`.
2. Inspects all programs and filters targets matching `runtime.GOOS`.
3. For each target:
   - Expands local path.
   - If path does not exist on disk, registers a non-fatal warning.
   - Copies file or directory recursively to `<repoPath>/data/<program>/<os>/<alias>`.
4. Saves updated `dotsaver.json` to `<repoPath>/dotsaver.json`.
5. Calls `git.CommitAndPush` with message `dotsaver: save on <hostname> (<os>)`.

#### Sync Workflow:
1. Calls `git.Pull` to fetch upstream updates.
2. Reads `dotsaver.json` from repo.
3. For each target matching `runtime.GOOS`:
   - Checks repo source path `<repoPath>/data/<program>/<os>/<alias>`.
   - If not in repo, skips.
   - If repo source is present:
     - Expands local target destination path.
     - Compares repo source and local destination:
       - Uses SHA-256 checksums to test if content matches.
       - If identical, skips.
       - If local file exists and differs:
         - Marks as a conflict item.
         - If user confirms backup: renames existing local file/folder to `<path>.bak` (appends timestamp `.bak.YYYYMMDDHHMMSS` if `.bak` already exists).
         - Copies repo version to local destination.
       - If local file does not exist: creates parent directories and copies cleanly.
4. Returns detailed result report (copied, backed up, unchanged, warnings, errors).

---

## 5. Terminal User Interface (`pkg/tui`)

Built with `bubbletea`, `bubbles`, and `lipgloss`.

### 5.1 Screens & Modes
1. **Dashboard Tab (`TabDashboard`)**:
   - Status header: Remote repo URL, local cache path, clean/dirty git status.
   - Summary of registered programs and their target paths for current OS.
   - Direct action triggers: `[s]` Save & Push, `[y]` Sync & Pull, `[r]` Refresh Git Status.
2. **Programs Tab (`TabPrograms`)**:
   - List/table of programs with count of targets.
   - Select program to view and edit its Windows/Linux target paths.
   - Action `[a]`: Add new program (interactive text input prompt).
   - Action `[t]`: Add target path to selected program (prompts for OS selection and path).
   - Action `[d]`: Delete program or target path.
3. **Settings Tab (`TabSettings`)**:
   - Input fields for Remote Git Repository URL and local cache path override.
   - Action `[Enter]`: Save settings and trigger repo clone/verification.
4. **Conflict Review Modal (`ModalConflict`)**:
   - Displays list of conflicting files detected during `Sync`.
   - Options:
     - `[Enter]`: Backup modified files to `.bak` and overwrite with repo version.
     - `[Esc]`: Skip conflicting files and apply only non-conflicting updates.
5. **Progress Spinner Modal (`ModalBusy`)**:
   - Full-width non-blocking overlay spinner rendered during asynchronous `tea.Cmd` tasks (e.g. Git clone, fetch, push, large directory sync).

### 5.2 Keybindings
- `1`, `2`, `3` or `Tab` / `Shift+Tab`: Navigate between tabs.
- `s`: Trigger Save & Push.
- `y`: Trigger Sync & Pull.
- `a`: Add new item (Program in Programs tab).
- `d`: Delete selected item.
- `esc`: Cancel form/modal or go back.
- `q` / `Ctrl+C`: Quit application.

---

## 6. Error Handling & Edge Cases

- **Git Not Found:** At startup, `exec.LookPath("git")` verifies `git` is available. If missing, displays an alert dialog informing the user to install Git.
- **Empty / Unreachable Remote:** If the remote URL is invalid or SSH keys fail, Git stderr is captured and presented in a scrollable message box.
- **Missing Source on Save:** When a path listed in `dotsaver.json` does not exist on the local system, `dotsaver` does not fail the entire batch; it skips the missing item, notes it in the operation log, and continues.
- **Permissions & Read-Only Files:** Copied files are written with normal user permissions (`0644` for files, `0755` for directories).
- **Graceful Shutdown:** All terminal states, alternate screen buffers, and raw terminal modes are cleanly restored on program termination.

---

## 7. Testing & Verification Plan

1. **Unit Testing:**
   - `pkg/config`: Default path creation, JSON read/write, environment fallback.
   - `pkg/manifest`: Serialization, deserialization, target additions, duplicate checks.
   - `pkg/syncer`:
     - Path expansion (`~`, `$VAR`, `%VAR%`).
     - Single file and recursive directory copying.
     - SHA-256 hash calculation and conflict detection.
     - `.bak` backup creation and collision handling.
2. **Integration Testing:**
   - In-memory/temp-dir Git integration test creating a local bare Git repository (`git init --bare`).
   - Simulates end-to-end user lifecycle:
     1. Configure bare remote as origin.
     2. Add programs with test dotfiles.
     3. Save & Push to bare remote.
     4. Modify local dotfiles.
     5. Sync & Pull with conflict backup verification.
3. **TUI Smoke Test:**
   - Verification that TUI starts and renders views cleanly without runtime panics.
