# zburn

Disposable identities — never give a service your real information again.

zburn generates disposable identities for online services: burner emails, names, addresses, phone numbers, and passwords. No account required, no tracking. Download, run, done.

## Install

### Homebrew

```bash
brew install zarlcorp/tap/zburn
```

### From Source

```bash
go install github.com/zarlcorp/zburn/cmd/zburn@latest
```

## Usage

### TUI Mode (default)

Run zburn without arguments to launch the interactive terminal interface:

```bash
zburn
```

### CLI Commands

Check the version:

```bash
zburn version
```

Planned commands (not yet implemented):
- `zburn email` — generate a burner email
- `zburn identity` — generate a complete identity
- `zburn list` — list saved identities
- `zburn forget <id>` — delete a saved identity

## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Run linter:

```bash
make lint
```

Start the TUI in development mode:

```bash
make run
```

## Learn More

- [zarlcorp/core](https://github.com/zarlcorp/core) — shared Go packages
- [MANIFESTO.md](https://github.com/zarlcorp/core/blob/main/MANIFESTO.md) — philosophy and architecture

---

MIT License
