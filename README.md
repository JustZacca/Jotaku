<p align="center">
  <img src="https://raw.githubusercontent.com/JustZacca/jotaku/main/assets/logo.png" alt="Jotaku Logo" width="200"/>
</p>

<h1 align="center">Jotaku</h1>

<p align="center">
  <strong>A beautiful, encrypted note-taking app for the terminal</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#keyboard-shortcuts">Shortcuts</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#server-sync">Server Sync</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License"/>
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue?style=flat-square" alt="Platform"/>
</p>

---

```
       ██╗ ██████╗ ████████╗ █████╗ ██╗  ██╗██╗   ██╗
       ██║██╔═══██╗╚══██╔══╝██╔══██╗██║ ██╔╝██║   ██║
       ██║██║   ██║   ██║   ███████║█████╔╝ ██║   ██║
  ██   ██║██║   ██║   ██║   ██╔══██║██╔═██╗ ██║   ██║
  ╚█████╔╝╚██████╔╝   ██║   ██║  ██║██║  ██╗╚██████╔╝
   ╚════╝  ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝
```

## Features

- **End-to-End Encryption** - All notes encrypted with AES-256-GCM
- **Folder Organization** - Organize notes in nested folders
- **Version History** - Track and restore previous versions
- **Tag System** - Categorize notes with hashtags
- **Password Protection** - Extra security for sensitive notes/folders
- **Cloud Sync** - Optional sync with self-hosted server
- **Multi-language** - English and Italian support
- **Vim-style Navigation** - Navigate with `j`/`k` keys
- **Fast Search** - Full-text search across all notes
- **Markdown Export/Import** - Seamless integration with other tools

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/JustZacca/jotaku.git
cd jotaku

# Build the client
go build -o jotaku ./cmd/client

# (Optional) Build the server
go build -o jotaku-server ./cmd/server

# Move to your PATH
sudo mv jotaku /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/JustZacca/jotaku/cmd/client@latest
```

## Usage

```bash
# Start Jotaku
jotaku

# First run will ask for a master password
# This password encrypts all your notes
```

<p align="center">
  <img src="https://raw.githubusercontent.com/JustZacca/jotaku/main/assets/screenshot.png" alt="Jotaku Screenshot" width="800"/>
</p>

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Open note/folder |
| `Tab` | Next panel |
| `Shift+Tab` | Previous panel |
| `Ctrl+L` | Go to list |

### Editing

| Key | Action |
|-----|--------|
| `i` | Edit note |
| `Esc` | Exit edit mode |
| `Ctrl+S` | Save |
| `Ctrl+X` | Save and close |

### Actions

| Key | Action |
|-----|--------|
| `Ctrl+N` | New note |
| `d` | Delete note/folder |
| `Ctrl+F` | Search |
| `h` | Version history |
| `t` | Edit tags |
| `p` | Set password |
| `Ctrl+Y` | Sync with server |
| `Ctrl+E` | Export to Markdown |
| `Ctrl+I` | Import Markdown |

### Folders

| Key | Action |
|-----|--------|
| `Ctrl+D` | New folder |
| `Backspace` | Go to parent folder |

### General

| Key | Action |
|-----|--------|
| `Ctrl+H` / `?` | Show help |
| `Ctrl+Q` | Quit |

## Configuration

On first run, Jotaku will ask you to select a language and create `config.yml` automatically.

Configuration is stored in the **same folder as the executable**:

```yaml
# Language: "it" (Italian) or "en" (English)
language: en

# Database file path (default: jotaku.db)
db_path: ""

# Theme (dark/light)
theme: dark

# Auto-save interval
auto_save_interval: 3s

# Server sync configuration (optional)
server:
  enabled: false
  url: http://localhost:5689
  username: myuser
  # token: auto-generated after login
```

See `config.example.yml` for a full example.

## Server Sync

Jotaku supports syncing notes with a self-hosted server.

### Running the Server

```bash
# Using Docker
cd docker
cp .env.example .env
# Edit .env with your JWT_SECRET
docker-compose up -d

# Or run directly
JWT_SECRET=your-secret-key jotaku-server
```

### Server Environment Variables

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret key for JWT tokens (min 32 chars) |
| `PORT` | Server port (default: 5689) |

### Connecting the Client

1. Edit `config.yml` in the same folder as the executable:

```yaml
server:
  enabled: true
  url: http://your-server.com:5689
  username: your-username
```

2. Start Jotaku and enter your master password
3. Auto-login will authenticate and save the token automatically

## Data Storage

All files are stored in the same folder as the executable:

| File | Description |
|------|-------------|
| `config.yml` | Configuration file |
| `jotaku.db` | SQLite database (encrypted) |
| `config.example.yml` | Example configuration |

## Security

- **Master Password** - Derives encryption key using Argon2
- **AES-256-GCM** - Industry-standard encryption
- **Local-first** - Your data stays on your machine by default
- **No telemetry** - Zero tracking or data collection

## Tech Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - TUI framework
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** - Styling
- **SQLite** - Local database
- **Go** - Backend language

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/JustZacca">@JustZacca</a>
</p>
