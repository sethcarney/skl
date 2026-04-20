# skl

The agent skill management CLI. No telemetry · Fully open source.

## Install

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/sethcarney/skl/main/install.sh | bash
```

**Windows** (PowerShell)

```powershell
irm https://raw.githubusercontent.com/sethcarney/skl/main/install.ps1 | iex
```

Both installers place the binary in `~/.local/bin/skl` and will warn if that directory isn't in your `PATH`.

To install to a different directory, set `INSTALL_DIR` before running:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/sethcarney/skl/main/install.sh | bash
```

## Usage

```
skl add <package>     Add a skill from GitHub or URL
skl remove            Remove installed skills
skl list              List installed skills
skl find [query]      Search the registry
skl update            Update to latest versions
```

Run `skl --help` for the full command reference.
