```
   ________  ________  ________  ______    ________  ________  ________  ________
  ╱        ╲╱        ╲╱        ╲╱      ╲  ╱        ╲╱        ╲╱        ╲╱        ╲
 ╱-        ╱    /    ╱     /   ╱       ╱ ╱         ╱    /    ╱    /    ╱    /    ╱
╱        _╱         ╱        _╱       ╱_╱       --╱         ╱        _╱       __╱
╲________╱╲___╱____╱╲____╱___╱╲________╱╲________╱╲________╱╲____╱___╱╲______╱
```

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

### Interactive TUI

Run zburn without arguments to launch the interactive terminal interface:

```bash
zburn
```

The TUI walks you through:
1. Master password prompt (create on first run, then enter to unlock)
2. Menu with options to generate, browse, or quick-copy a burner email
3. Generate view to create and save new identities
4. Browse view to list and manage saved identities
5. Detail view to inspect and copy individual identity fields

All generated data is encrypted at rest using your master password.

### CLI Commands

Generate a burner email and print to stdout:

```bash
zburn email
```

Generate a complete identity:

```bash
zburn identity
```

Options:
- `--json` — output as JSON instead of formatted text
- `--save` — encrypt and save to the store (prompts for master password)

Example:

```bash
zburn identity --json --save
```

List all saved identities:

```bash
zburn list
```

Options:
- `--json` — output as JSON instead of formatted text

Delete a saved identity by ID:

```bash
zburn forget abc123def456
```

Print version:

```bash
zburn version
```

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

- [Website](https://zarlcorp.github.io/zburn) — Documentation and install instructions
- [zarlcorp/core](https://github.com/zarlcorp/core) — shared Go packages
- [MANIFESTO.md](https://github.com/zarlcorp/core/blob/main/MANIFESTO.md) — philosophy and architecture

---

MIT License
