# dotsaver

> [!CAUTION]
> **Internal Use Only & Vibe Coded**  
> This tool is **vibe coded** and intended strictly for **internal / personal use**. It comes with no warranties or guarantees. Use at your own discretion.

> [!WARNING]
> **CRITICAL SECURITY WARNING: Secrets & API Keys**  
> While `dotsaver` is designed to synchronize dotfiles and application configurations across systems via a remote Git repository, **dotfiles frequently contain sensitive secrets, API keys, tokens, and private credentials**.
>
> - **Never commit or push API keys, private tokens, or credentials into the repository.**
> - **Be cautious:** Review any configuration files before adding them to `dotsaver`.
> - **Avoid forward declaration:** Do **not** hardcode secrets or forward-declare API keys directly in configuration files.
> - **Use Environment Variables:** Reference secrets through system environment variables (e.g., `$API_KEY`, `%API_KEY%`) or store them in external, untracked secret managers / local `.env` files that are kept strictly outside the tracked repository.

---

`dotsaver` is a minimalist, cross-platform Terminal User Interface (TUI) tool written in Go for managing, backing up, and synchronizing dotfiles and application configurations across multiple operating systems (specifically **Windows** and **Linux**) through a remote Git repository.

Powered by the [Charm](https://charm.sh/) ecosystem ([Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Bubbles](https://github.com/charmbracelet/bubbles)).

---

## Features

- **System Git Integration:** Relies directly on your system `git` CLI (`os/exec`). Seamlessly inherits your existing SSH keys, GPG commit signing, and credential helpers without storing passwords or tokens.
- **Cross-Platform Target Mapping:** Map configurations for arbitrary applications per operating system (`windows` or `linux`), supporting both single files and entire directories.
- **Path Expansion:** Supports `~` home directory paths, Windows `%ENV_VAR%` syntax (e.g., `%APPDATA%`, `%LOCALAPPDATA%`), and Linux `$VAR` / `${VAR}` syntax.
- **Remote Manifest (`dotsaver.json`):** Your sync specifications live right alongside your data in the Git repository root, ensuring consistent configuration across all your machines.
- **Save (Collect & Push):** Gathers local configuration files/directories based on the manifest, copies them to your local repository cache, commits, and pushes them to the remote Git repository.
- **Sync (Pull & Apply):** Pulls upstream updates and copies files to their target locations on your machine.
- **Safe Conflict Handling & Backups:** Uses SHA-256 checksums to detect local differences. Creates automatic `.bak` backups before overwriting any conflicting local files.
- **Aesthetic TUI:** Clean terminal interface with views for dashboard status, program management, settings, and interactive conflict resolution modals.

---

## Prerequisites

- **Git:** System `git` CLI installed and available in your `PATH` (https://git-scm.com/).
- **Go:** Go 1.22 or newer (to build from source).

---

## Installation & Building

Clone the repository and build the binary:

```bash
# Clone the repository
git clone https://github.com/user/dotsaver.git
cd dotsaver

# Build binary
# On Linux / macOS:
go build -o bin/dotsaver ./cmd/dotsaver

# On Windows (PowerShell):
go build -o bin/dotsaver.exe ./cmd/dotsaver
```

Optionally move the resulting binary into your system `PATH`.

---

## Usage

Launch the TUI:

```bash
dotsaver
```

### CLI Flags

```text
Usage:
  dotsaver [flags]

Flags:
  -c, --config <path>   Path to configuration file
                        (default: %APPDATA%\dotsaver\config.json on Windows,
                         ~/.config/dotsaver/config.json on Linux)
  -v, --version         Show dotsaver version
  -h, --help            Show this help message
```

---

## Keybindings

### Navigation
| Key | Action |
| --- | --- |
| `1` / `2` / `3` | Jump directly to Dashboard (`1`), Programs (`2`), or Settings (`3`) tab |
| `Tab` / `Shift+Tab` | Cycle between tabs |
| `q` / `Ctrl+C` | Quit `dotsaver` |
| `Esc` | Close modal / return from form / cancel operation |

### Dashboard View
| Key | Action |
| --- | --- |
| `s` | **Save & Push:** Copy local configurations to repo cache, commit, and push |
| `y` | **Sync & Pull:** Pull remote updates, check conflicts, and apply to local system |
| `r` | Refresh repository and Git status |

### Programs View
| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Navigate program list |
| `a` | Add a new program to track |
| `t` | Add target path (OS + local file/folder path) to selected program |
| `d` | Delete selected program or target path |

### Settings View
| Key | Action |
| --- | --- |
| `i` | Activate **Input Mode** to edit input fields |
| `Alt+i` / `Esc` | Return to **Select Mode** |
| `↑` / `k`, `↓` / `j` | Select input line (Select Mode) / Navigate lines (Input Mode) |
| `Tab` | Cycle tabs (Select Mode) / Next input line (Input Mode) |
| `Enter` | Save settings & initialize repository cache |

---

## Configuration & Layout

### Local Application Configuration

Stored locally on your machine:
- **Windows:** `%APPDATA%\dotsaver\config.json`
- **Linux:** `~/.config/dotsaver/config.json`

```json
{
  "remote_url": "git@github.com:yourusername/my-dotfiles.git",
  "repo_path": ""
}
```

* `remote_url`: The Git repository where dotfiles and manifests are synchronized.
* `repo_path`: Path to the local working cache clone (if omitted, defaults to `%LOCALAPPDATA%\dotsaver\repo` on Windows or `~/.local/share/dotsaver/repo` on Linux).

### Remote Repository Structure

In your remote Git repository:

```text
<git-repo-root>/
├── dotsaver.json                # Central manifest of tracked programs & paths
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

### Remote Manifest (`dotsaver.json`)

The manifest specifies what files or directories correspond to which program per operating system:

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

---

## Conflict Resolution

When executing **Sync & Pull**, `dotsaver` compares the SHA-256 hashes of the repository versions with existing local files:
- If files match, no action is taken.
- If a local file has modified content, a conflict modal prompts you to either:
  - **Confirm Backup & Overwrite (`Enter`):** Backs up existing local file/folder with a `.bak` (or `.bak.<timestamp>`) suffix and applies the repository version.
  - **Skip (`Esc`):** Skips the conflicting target and keeps your local version intact.
