# az-blob-robber

A professional Terminal User Interface (TUI) for discovering, exploring, and exfiltrating Azure Blob Storage during penetration testing engagements.

![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-blue)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.20-00ADD8?logo=go)

## Features

- 🔍 **Concurrent Discovery** - Brute-force storage accounts and containers from wordlists with configurable concurrency
- 📊 **Real-time Updates** - See discovered resources immediately as scanning progresses
- 🎯 **Direct Targeting** - Skip brute-forcing when you already know the account/container name
- 🔐 **Token Authentication** - Use Azure Storage access tokens for authenticated access to private resources
- 🔄 **Full Versioning Support** - List, view, and download blob versions and snapshots
- 🗑️ **Deleted File Detection** - Identify and recover soft-deleted blobs (when versioning is enabled)
- 📥 **Smart Downloads** - Organized directory structure with automatic versioned filename generation
- ⚡ **No Dependencies** - Pure Go implementation using Azure Storage REST API (no Azure SDK required)

**Note**: During brute-force discovery, only publicly accessible containers are shown. Use the `-t`/`--token` flag with an authentication token to access private containers and storage accounts.

## Installation

### Prerequisites

- Go 1.20 or higher

### Build from Source

```bash
git clone https://github.com/joe-durbin/az-blob-robber.git
cd az-blob-robber
go build -o az-blob-robber ./cmd/az-blob-robber
```

## Quick Start

### Basic Brute-Force Scan

```bash
# Use default wordlists (requires -b flag)
./az-blob-robber -b
./az-blob-robber --brute-force-defaults

# Use custom wordlists
./az-blob-robber -b -A custom-accounts.txt -C custom-containers.txt

# Adjust concurrency (default: 20)
./az-blob-robber -b -n 50
```

### Direct Targeting

```bash
# Target a specific account and container
./az-blob-robber -a mywebsite -c '$webroot'

# Target specific account, scan all containers from wordlist
./az-blob-robber -a mycompany

# Scan all accounts, target specific container
./az-blob-robber -c public
```

### Authenticated Access

```bash
# Use with Azure Storage access token
./az-blob-robber -a mycompany -c secret-reports -t "eyJ0eXAi..."
```

### Debug Mode

```bash
# Output curl equivalents for all successful requests
./az-blob-robber -b -d
./az-blob-robber -a myaccount --debug
```

### Custom User-Agent

```bash
# Use a custom User-Agent string (default: az-blob-robber/1.0)
./az-blob-robber -a myaccount -u "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
./az-blob-robber -b --user-agent "MyCustomAgent/1.0"
```

The User-Agent header is sent with all HTTP requests to Azure Storage. By default, it's set to `az-blob-robber/1.0`. You can customize it using the `-u` or `--user-agent` flag for testing purposes or to match specific client behavior.

### Screenshots

**Brute-Force Discovery in Action**

SCREENSHOT COMING

*Real-time discovery of storage accounts and containers during scanning*

**Direct Targeting with Known Credentials**

SCREENSHOT COMING

*Accessing a specific account and container when you already know the names*

## Usage Guide

### Interface Layout

SCREENSHOT COMING

The interface consists of two main panes:
- **Left Pane**: Displays discovered storage accounts and their containers in a hierarchical tree structure
- **Right Pane**: Shows files within the selected container, with versioning indicators and file metadata

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate lists |
| `Tab` | Switch focus between accounts and files panes |
| `d` | Download selected file |
| `v` | Expand/collapse version history for selected file |
| `b` | Bulk download all latest versions in container |
| `t` | Toggle theme (cycles through all available themes) |
| `Enter` | Confirm in dialogs |
| `y/n` | Confirm/cancel overwrite prompts |
| `q` or `Ctrl+C` | Quit application |

### File Icons

| Icon | Meaning |
|------|---------|
| 📄 | Regular file (no versioning) |
| 🔄 | Versioned file (has version history) |
| 🗑️ | Deleted file (soft-deleted with versioning enabled) |

### Workflow

1. **Launch** - Start the tool with your target parameters
2. **Scan** - Scanning begins automatically, results appear in real-time
3. **Navigate** - Use arrow keys to browse accounts and containers
4. **Tab** - Switch to the files pane when you find an interesting container
5. **Download** - Press `d` to download files
6. **Versions** - Press `v` on versioned files to see history
7. **Quit** - Press `q` when done

### Download Organization

Files are saved to a structured directory:

```
downloads/
└── 2024-12-02/
    └── mywebsite/
        └── $webroot/
            ├── index.html
            └── backup.zip
```

- **Date-based structure**: `downloads/YYYY-MM-DD/account/container/`
- **Versioned files**: Timestamp appended: `filename_YYYYMMDDHHMMSS.ext`
- **Overwrite protection**: Prompts before replacing existing files

## Authentication

### Obtaining Azure Storage Tokens

**Azure CLI:**
```bash
az account get-access-token --resource https://storage.azure.com
```

**FindMeAccess (pentesting tool):**
```bash
python3 findmeaccess.py token -u user@domain.com -p 'password' \
  -c 'client-id' -r https://storage.azure.com
```

Save the access token and pass it via the `-t` or `--token` flag:
```bash
./az-blob-robber -a myaccount -c mycontainer -t "eyJ0eX..."
```

### How Authentication Works

- Token is sent as `Authorization: Bearer <token>` header in all HTTP requests
- Uses Azure Storage REST API version `2019-12-12`
- User-Agent header is set to `az-blob-robber/1.0` by default (customizable via `-u`/`--user-agent`)
- Works for discovery, listing, and downloading operations
- Enables access to private containers and non-public storage accounts

## Versioning & Deleted Files

### Versioning Detection

When versioning is enabled on a storage account, you'll see:
- **Header notice**: `🔄 VERSIONING ENABLED • Press 'v' to expand history`
- **Version icon**: All versioned files show 🔄 icon

### Viewing Version History

1. Navigate to a versioned file (indicated by 🔄)
2. Press `v` to expand version history
3. Versions appear indented below the current file:
   ```
   🔄 database.sql 5.2 MB
     └─ 2024-01-15T12:00:00.0000000Z 5.1 MB
     └─ 2024-01-10T08:30:00.0000000Z 4.9 MB
   ```
4. Navigate to a specific version and press `d` to download it

SCREENSHOT COMING

*Expanded version history showing multiple snapshots of a file*

### Deleted Files

Files marked as deleted (🗑️) are:
- Current versions that have been deleted but are still recoverable
- Only visible when the storage account has versioning enabled
- Downloadable if you have appropriate access

SCREENSHOT COMING

*Deleted files are clearly marked with a trash icon and (Deleted) label*

## Bulk Download

Quickly download all files from a container:

1. Navigate to the container in the left pane
2. Press `Tab` to focus the file list
3. Press `b` to initiate bulk download
4. Confirm with `y`

**Notes:**
- Only downloads latest versions (skips historical versions)
- Includes deleted versioned files
- Shows summary of successful/failed downloads

## Debug Mode

Enable debug mode to output curl equivalents for all successful requests:

```bash
./az-blob-robber -b -d
./az-blob-robber --brute-force-defaults --debug
```

Debug output is written to a timestamped log file (e.g., `debug_2024-12-02_15-04-05.log`) in the current directory.

This outputs commands like:
```bash
curl -X GET -H 'User-Agent: az-blob-robber/1.0' -H 'x-ms-version: 2019-12-12' \
 'https://myaccount.blob.core.windows.net/mycontainer?restype=container&comp=list'
```

Useful for:
- Learning Azure Storage REST API
- Manual verification of operations
- Creating standalone scripts
- Debugging authentication or access issues

## Themes

Toggle between color schemes using the `t` key. The tool includes five built-in themes:

- **Default Theme** - Blue focused borders, classic terminal colors
- **Rose Pine Theme** - Soft pink focused borders, muted pastels inspired by [Rose Pine](https://rosepinetheme.com/)
- **Hacker Theme** - Classic green-on-black hacker aesthetic
- **Catppuccin Theme** - Inspired by Catppuccin Mocha color palette
- **Vibrant Theme** - Super colorful theme with vibrant accents

Theme persists during the session and affects:
- Border colors (focused/unfocused panes)
- Modal highlighting
- Status messages
- Version notices

## Advanced Usage

### Wordlists

Default wordlists are in `wordlists/`:
- `accounts.txt` - Storage account names
- `containers.txt` - Container names

Customize these files or provide your own via `-A`/`--accounts` and `-C`/`--containers` flags.

### Concurrency Tuning

```bash
# Conservative (slower, less network load)
./az-blob-robber -b -n 10
./az-blob-robber --brute-force-defaults --concurrency 10

# Aggressive (faster, higher network load)
./az-blob-robber -b -n 100
./az-blob-robber --brute-force-defaults --concurrency 100
```

Higher concurrency speeds up discovery but may trigger rate limiting on some networks.

### Custom User-Agent

The tool sends a User-Agent header with all HTTP requests. By default, this is set to `az-blob-robber/1.0`. You can customize it using the `-u` or `--user-agent` flag:

```bash
# Use default User-Agent (az-blob-robber/1.0)
./az-blob-robber -b

# Use custom User-Agent
./az-blob-robber -b -u "Mozilla/5.0 (compatible; MSIE 10.0)"
./az-blob-robber -a myaccount --user-agent "MyCustomTool/2.0"
```

**Use cases:**
- Testing how Azure Storage responds to different User-Agent strings
- Mimicking specific client behavior for compatibility testing
- Customizing identification for logging or monitoring purposes

## Architecture

**Language**: Go  
**TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-inspired MVU pattern)  
**Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)  
**HTTP**: Native Go `net/http` (no Azure SDK)

**Project Structure:**
```
az-blob-robber/
├── cmd/az-blob-robber/  # Main entry point
├── pkg/
│   ├── azure/           # Azure Storage REST API client
│   ├── scanner/         # Concurrent brute-force engine
│   └── ui/              # TUI components and logic (themes, styles, app)
├── wordlists/           # Default discovery wordlists
└── downloads/           # Downloaded files (auto-created)
```

## Troubleshooting

### No Results Found

- Verify wordlists contain valid names
- Check network connectivity and DNS resolution
- Try reducing concurrency: `-n 10` or `--concurrency 10`

### Deleted Files Not Showing

- Deleted files only appear when versioning is enabled on the storage account
- Use the `2019-12-12` API version (already configured in the tool)
- Some storage accounts don't support versioning

### Download Failures

- Ensure sufficient disk space in `downloads/` directory
- Check network stability for large files
- Verify you have read access (use `-t` or `--token` if needed)

## Security & Ethics

⚠️ **IMPORTANT**: This tool is for authorized penetration testing only.

- Only scan Azure Storage accounts you have permission to test
- Unauthorized access to cloud resources may violate laws and terms of service
- The authors assume no liability for misuse of this software

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- [Lipgloss](https://github.com/charmbracelet/lipgloss) by Charm
- [Bubbles](https://github.com/charmbracelet/bubbles) by Charm

## License

See LICENSE file for details.
