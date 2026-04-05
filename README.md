# MySSH

MySSH is a lightweight desktop SSH client built with Go and Wails.

It is designed to feel fast and native, without Electron, while still supporting secure secret storage, interactive terminal sessions, local shell access, and built-in SFTP.

## Features

- Native desktop app built with Go + Wails
- Dark-mode UI
- SSH profile management
- Local terminal session
- Multiple session tabs
- Rename, reconnect, and close per tab
- Built-in SFTP browser
- Password authentication
- Private key authentication
- Private key passphrase support
- `ssh-agent` authentication
- `agent + fallback key` authentication
- Optional remote `SECRET` environment variable per profile
- Host key verification with `known_hosts`
- Unknown-host trust flow
- OS keyring storage for sensitive values

## Authentication Modes

MySSH currently supports:

- `agent`
- `password`
- `private_key`
- `agent + fallback key`

For key-based profiles you can use:

- `key path`
- `paste key content`

Optional secrets stored in the OS keyring:

- password
- pasted private key content
- private key passphrase
- remote `SECRET` token

## Security Model

Profile metadata is stored locally in `profiles.json`.

Sensitive values are not stored in plain text in that file. Instead, MySSH stores them in the OS keyring when possible.

Examples of protected values:

- passwords
- pasted private keys
- key passphrases
- remote `SECRET` tokens

Host verification uses your SSH `known_hosts` file. Unknown hosts are not silently accepted.

## Built-In Tools

### Terminal

MySSH includes:

- interactive SSH terminal
- local system terminal
- multi-session tabs
- reconnect per tab
- background sessions
- copy / paste integration

### SFTP

MySSH includes a built-in SFTP browser with:

- remote directory listing
- folder navigation
- refresh
- parent directory navigation
- file download to `~/Downloads/MySSH`

## Project Layout

- `cmd/myssh` - desktop app entrypoint and Wails bindings
- `internal/app` - application service layer
- `internal/domain` - core domain model
- `internal/store` - profile persistence
- `internal/secret` - OS keyring integration
- `internal/sshclient` - terminal SSH and local shell sessions
- `internal/sftpclient` - SFTP session management
- `cmd/myssh/frontend/dist` - frontend HTML, CSS, and JS

## Requirements

### Linux

You need a recent Go toolchain and the Linux libraries required by Wails/WebKit.

Typical Debian/Ubuntu packages:

```bash
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libsecret-1-dev
```

## Build Locally

From the project root:

```bash
cd ~/Desktop/MySSH
go mod tidy
go build -tags "desktop,production,webkit2_41" -o myssh ./cmd/myssh
```

Run it with:

```bash
./myssh
```

## Development Notes

- Local shell support is currently Linux-first.
- SFTP currently supports browsing and downloading.
- Terminal sessions can continue in the background after leaving the session screen.
- If a remote host is missing from `known_hosts`, MySSH will ask you to trust it first.

## GitHub Actions

The repository includes a GitHub Actions workflow that builds the app for Linux and uses Go caching to speed up repeated runs.

## Roadmap

- SFTP upload
- rename / delete / mkdir in SFTP
- split terminal + SFTP view
- more session polish
- broader platform support for local terminal mode
