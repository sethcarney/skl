# skills

A custom fork of the open agent skills ecosystem CLI with several enhancements over the upstream project.

## What's Different in This Fork

This fork adds a handful of new features on top of the upstream [agent-skills](https://agentskills.io) CLI:

- **Self-update command** — Run `skills update-cli` to update the CLI binary in place, no package manager needed
- **Project skills lock** — Track both local and remote skills in `skills-lock.json`; commit it and teammates get the same set with `skills install`
- **OSV vulnerability scanning** — Checks the [OSV database](https://osv.dev) for known advisories against any GitHub source before you confirm installation. No API key required.
- **No telemetry** — All usage tracking and third-party audit API calls have been removed; no data is collected or sent
- **Bun binary distribution** — Compiled directly to standalone native executables via Bun instead of npm/npx; no Node.js runtime required

<!-- agent-list:start -->

Supports **OpenCode**, **Claude Code**, **Codex**, **Cursor**, and [41 more](#available-agents).

<!-- agent-list:end -->

## Installation

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/sethcarney/skills/main/install.sh | bash
```

Installs to `~/.local/bin/skills`. To use a custom directory:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/sethcarney/skills/main/install.sh | bash
```

**Windows** (PowerShell)

```powershell
irm https://raw.githubusercontent.com/sethcarney/skills/main/install.ps1 | iex
```

Installs to `%USERPROFILE%\.local\bin\skills.exe`. To use a custom directory:

```powershell
$env:INSTALL_DIR = "C:\tools"; irm https://raw.githubusercontent.com/sethcarney/skills/main/install.ps1 | iex
```

Both installers download a prebuilt binary and warn if the install directory is not in your `PATH`.

## Install a Skill

```bash
skills add vercel-labs/agent-skills
```

### Source Formats

```bash
# GitHub shorthand (owner/repo)
skills add vercel-labs/agent-skills

# Full GitHub URL
skills add https://github.com/vercel-labs/agent-skills

# Direct path to a skill in a repo
skills add https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines

# GitLab URL
skills add https://gitlab.com/org/repo

# Any git URL
skills add git@github.com:vercel-labs/agent-skills.git

# Local path
skills add ./my-local-skills
```

### Options

| Option                    | Description                                                                                                                                        |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-p, --project`           | Force project-scope install without prompting (writes to `skills-lock.json`)                                                                       |
| `-g, --global`            | Force global install without prompting (writes to `~/.agents/.skill-lock.json`)                                                                    |
| `-a, --agent <agents...>` | <!-- agent-names:start -->Target specific agents (e.g., `claude-code`, `codex`). See [Available Agents](#available-agents)<!-- agent-names:end --> |
| `-s, --skill <skills...>` | Install specific skills by name (use `'*'` for all skills)                                                                                         |
| `-l, --list`              | List available skills without installing                                                                                                           |
| `--copy`                  | Copy files instead of symlinking to agent directories                                                                                              |
| `-y, --yes`               | Skip all confirmation prompts                                                                                                                      |
| `--all`                   | Install all skills to all agents without prompts                                                                                                   |

### Examples

```bash
# List skills in a repository
skills add vercel-labs/agent-skills --list

# Install specific skills
skills add vercel-labs/agent-skills --skill frontend-design --skill skill-creator

# Install a skill with spaces in the name (must be quoted)
skills add owner/repo --skill "Convex Best Practices"

# Install to specific agents
skills add vercel-labs/agent-skills -a claude-code -a opencode

# Non-interactive installation (CI/CD friendly)
skills add vercel-labs/agent-skills --skill frontend-design -g -a claude-code -y

# Install all skills from a repo to all agents
skills add vercel-labs/agent-skills --all

# Install all skills to specific agents
skills add vercel-labs/agent-skills --skill '*' -a claude-code

# Install specific skills to all agents
skills add vercel-labs/agent-skills --agent '*' --skill frontend-design
```

### Installation Scope

| Scope       | Flag      | Location            | Use Case                                      |
| ----------- | --------- | ------------------- | --------------------------------------------- |
| **Project** | (default) | `./<agent>/skills/` | Committed with your project, shared with team |
| **Global**  | `-g`      | `~/<agent>/skills/` | Available across all projects                 |

### Project Skills and `skills-lock.json`

`skills-lock.json` is the project-level lock file, designed to be committed to version control. It tracks every skill installed at project scope — both local paths and remote GitHub sources — so teammates can reproduce the exact same set with a single command.

```bash
# Install a remote skill at project scope (no global prompt)
skills add owner/repo --project

# Install a local skill at project scope
skills add ./src/my-skill --project

# Restore all skills from skills-lock.json (e.g., after git clone)
skills install
```

You can also declare skills manually in `skills-lock.json` without installing them first:

```json
{
  "version": 1,
  "skills": {
    "my-local-skill": {
      "source": "./src/skills/my-skill",
      "sourceType": "local"
    },
    "shared-skill": {
      "source": "owner/shared-skills",
      "sourceType": "github",
      "ref": "main"
    }
  }
}
```

Running `skills install` will fetch and install all declared skills. The `computedHash` field is written automatically after first install and used to detect changes.

### `skills install` options

| Option                    | Description                                                                    |
| ------------------------- | ------------------------------------------------------------------------------ |
| `-a, --agent <agents...>` | Install to specific agents, skipping the selection prompt                      |
| `-y, --yes`               | Skip the agent selection prompt; installs to universal agents (`.agents/skills/`) only |

When run without flags, `skills install` shows an interactive agent selection prompt. Universal agents (`.agents/skills/`) are always included and shown as locked — you can additionally select any other agents you want the skills symlinked into.

### Security Scanning (OSV)

When installing skills from a GitHub source, the CLI queries the [OSV (Open Source Vulnerability) database](https://osv.dev) for any known advisories filed against that repository. This check:

- Runs in parallel with the agent/scope prompts — zero added latency
- Requires no API key or authentication
- Times out silently after 3 seconds if OSV is unreachable
- **Never blocks installation** — the result is advisory only

If advisories are found, a note is shown before the confirmation prompt:

```
┌─ Security Advisories (OSV) ────────────────────────────────────┐
│ ⚠  2 known advisories — highest severity: High                  │
│                                                                  │
│   GHSA-xxxx-xxxx-xxxx — Example advisory summary               │
│   GHSA-yyyy-yyyy-yyyy — Another advisory                        │
│                                                                  │
│ Details: https://osv.dev/?q=owner%2Frepo                        │
└──────────────────────────────────────────────────────────────────┘
```

If no advisories are found, nothing is shown (clean installs stay quiet).

### Installation Methods

When installing interactively, you can choose:

| Method                    | Description                                                                                 |
| ------------------------- | ------------------------------------------------------------------------------------------- |
| **Symlink** (Recommended) | Creates symlinks from each agent to a canonical copy. Single source of truth, easy updates. |
| **Copy**                  | Creates independent copies for each agent. Use when symlinks aren't supported.              |

## Other Commands

| Command                      | Description                                   |
| ---------------------------- | --------------------------------------------- |
| `skills list`                | List installed skills (alias: `ls`)          |
| `skills find [query]`        | Search for skills interactively or by keyword |
| `skills remove [skills]`     | Remove installed skills from agents           |
| `skills check`               | Check for available skill updates             |
| `skills update [skills]`     | Update installed skills to latest versions    |
| `skills init [name]`         | Create a new SKILL.md template                |

### `skills list`

List all installed skills. Similar to `npm ls`.

```bash
# List all installed skills (project and global)
skills list

# List only global skills
skills ls -g

# Filter by specific agents
skills ls -a claude-code -a cursor
```

### `skills find`

Search for skills interactively or by keyword.

```bash
# Interactive search (fzf-style)
skills find

# Search by keyword
skills find typescript
```

### `skills update`

```bash
# Check if any installed skills have updates
skills check

# Update all skills to latest versions
skills update

# Update all skills (interactive scope prompt)
skills update

# Update a single skill by name
skills update my-skill

# Update multiple specific skills
skills update frontend-design web-design-guidelines

# Update only global or project skills
skills update -g
skills update -p

# Non-interactive (auto-detects scope: project if in a project, else global)
skills update -y
```

| Option          | Description                                                               |
| --------------- | ------------------------------------------------------------------------- |
| `-g, --global`  | Only update global skills                                                 |
| `-p, --project` | Only update project skills                                                |
| `-y, --yes`     | Skip scope prompt (auto-detect: project if in a project dir, else global) |
| `[skills...]`   | Update specific skills by name instead of all                             |

### `skills init`

```bash
# Create SKILL.md in current directory
skills init

# Create a new skill in a subdirectory
skills init my-skill
```

### `skills remove`

Remove installed skills from agents.

```bash
# Remove interactively (select from installed skills)
skills remove

# Remove specific skill by name
skills remove web-design-guidelines

# Remove multiple skills
skills remove frontend-design web-design-guidelines

# Remove from global scope
skills remove --global web-design-guidelines

# Remove from specific agents only
skills remove --agent claude-code cursor my-skill

# Remove all installed skills without confirmation
skills remove --all

# Remove all skills from a specific agent
skills remove --skill '*' -a cursor

# Remove a specific skill from all agents
skills remove my-skill --agent '*'

# Use 'rm' alias
skills rm my-skill
```

| Option         | Description                                      |
| -------------- | ------------------------------------------------ |
| `-g, --global` | Remove from global scope (~/) instead of project |
| `-a, --agent`  | Remove from specific agents (use `'*'` for all)  |
| `-s, --skill`  | Specify skills to remove (use `'*'` for all)     |
| `-y, --yes`    | Skip confirmation prompts                        |
| `--all`        | Shorthand for `--skill '*' --agent '*' -y`       |

## What are Agent Skills?

Agent skills are reusable instruction sets that extend your coding agent's capabilities. They're defined in `SKILL.md`
files with YAML frontmatter containing a `name` and `description`.

Skills let agents perform specialized tasks like:

- Generating release notes from git history
- Creating PRs following your team's conventions
- Integrating with external tools (Linear, Notion, etc.)

Discover skills at **[skills.sh](https://skills.sh)**

## Supported Agents

Skills can be installed to any of these agents:

<!-- supported-agents:start -->

| Agent                                 | `--agent`                                | Project Path           | Global Path                     |
| ------------------------------------- | ---------------------------------------- | ---------------------- | ------------------------------- |
| Amp, Kimi Code CLI, Replit, Universal | `amp`, `kimi-cli`, `replit`, `universal` | `.agents/skills/`      | `~/.config/agents/skills/`      |
| Antigravity                           | `antigravity`                            | `.agents/skills/`      | `~/.gemini/antigravity/skills/` |
| Augment                               | `augment`                                | `.augment/skills/`     | `~/.augment/skills/`            |
| IBM Bob                               | `bob`                                    | `.bob/skills/`         | `~/.bob/skills/`                |
| Claude Code                           | `claude-code`                            | `.claude/skills/`      | `~/.claude/skills/`             |
| OpenClaw                              | `openclaw`                               | `skills/`              | `~/.openclaw/skills/`           |
| Cline, Warp                           | `cline`, `warp`                          | `.agents/skills/`      | `~/.agents/skills/`             |
| CodeBuddy                             | `codebuddy`                              | `.codebuddy/skills/`   | `~/.codebuddy/skills/`          |
| Codex                                 | `codex`                                  | `.agents/skills/`      | `~/.codex/skills/`              |
| Command Code                          | `command-code`                           | `.commandcode/skills/` | `~/.commandcode/skills/`        |
| Continue                              | `continue`                               | `.continue/skills/`    | `~/.continue/skills/`           |
| Cortex Code                           | `cortex`                                 | `.cortex/skills/`      | `~/.snowflake/cortex/skills/`   |
| Crush                                 | `crush`                                  | `.crush/skills/`       | `~/.config/crush/skills/`       |
| Cursor                                | `cursor`                                 | `.agents/skills/`      | `~/.cursor/skills/`             |
| Deep Agents                           | `deepagents`                             | `.agents/skills/`      | `~/.deepagents/agent/skills/`   |
| Droid                                 | `droid`                                  | `.factory/skills/`     | `~/.factory/skills/`            |
| Firebender                            | `firebender`                             | `.agents/skills/`      | `~/.firebender/skills/`         |
| Gemini CLI                            | `gemini-cli`                             | `.agents/skills/`      | `~/.gemini/skills/`             |
| GitHub Copilot                        | `github-copilot`                         | `.agents/skills/`      | `~/.copilot/skills/`            |
| Goose                                 | `goose`                                  | `.goose/skills/`       | `~/.config/goose/skills/`       |
| Junie                                 | `junie`                                  | `.junie/skills/`       | `~/.junie/skills/`              |
| iFlow CLI                             | `iflow-cli`                              | `.iflow/skills/`       | `~/.iflow/skills/`              |
| Kilo Code                             | `kilo`                                   | `.kilocode/skills/`    | `~/.kilocode/skills/`           |
| Kiro CLI                              | `kiro-cli`                               | `.kiro/skills/`        | `~/.kiro/skills/`               |
| Kode                                  | `kode`                                   | `.kode/skills/`        | `~/.kode/skills/`               |
| MCPJam                                | `mcpjam`                                 | `.mcpjam/skills/`      | `~/.mcpjam/skills/`             |
| Mistral Vibe                          | `mistral-vibe`                           | `.vibe/skills/`        | `~/.vibe/skills/`               |
| Mux                                   | `mux`                                    | `.mux/skills/`         | `~/.mux/skills/`                |
| OpenCode                              | `opencode`                               | `.agents/skills/`      | `~/.config/opencode/skills/`    |
| OpenHands                             | `openhands`                              | `.openhands/skills/`   | `~/.openhands/skills/`          |
| Pi                                    | `pi`                                     | `.pi/skills/`          | `~/.pi/agent/skills/`           |
| Qoder                                 | `qoder`                                  | `.qoder/skills/`       | `~/.qoder/skills/`              |
| Qwen Code                             | `qwen-code`                              | `.qwen/skills/`        | `~/.qwen/skills/`               |
| Roo Code                              | `roo`                                    | `.roo/skills/`         | `~/.roo/skills/`                |
| Trae                                  | `trae`                                   | `.trae/skills/`        | `~/.trae/skills/`               |
| Trae CN                               | `trae-cn`                                | `.trae/skills/`        | `~/.trae-cn/skills/`            |
| Windsurf                              | `windsurf`                               | `.windsurf/skills/`    | `~/.codeium/windsurf/skills/`   |
| Zencoder                              | `zencoder`                               | `.zencoder/skills/`    | `~/.zencoder/skills/`           |
| Neovate                               | `neovate`                                | `.neovate/skills/`     | `~/.neovate/skills/`            |
| Pochi                                 | `pochi`                                  | `.pochi/skills/`       | `~/.pochi/skills/`              |
| AdaL                                  | `adal`                                   | `.adal/skills/`        | `~/.adal/skills/`               |

<!-- supported-agents:end -->

> [!NOTE]
> **Kiro CLI users:** After installing skills, manually add them to your custom agent's `resources` in
> `.kiro/agents/<agent>.json`:
>
> ```json
> {
>   "resources": ["skill://.kiro/skills/**/SKILL.md"]
> }
> ```

The CLI automatically detects which coding agents you have installed. If none are detected, you'll be prompted to select
which agents to install to.

## Creating Skills

Skills are directories containing a `SKILL.md` file with YAML frontmatter:

```markdown
---
name: my-skill
description: What this skill does and when to use it
---

# My Skill

Instructions for the agent to follow when this skill is activated.

## When to Use

Describe the scenarios where this skill should be used.

## Steps

1. First, do this
2. Then, do that
```

### Required Fields

- `name`: Unique identifier (lowercase, hyphens allowed)
- `description`: Brief explanation of what the skill does

### Optional Fields

- `metadata.internal`: Set to `true` to hide the skill from normal discovery. Internal skills are only visible and
  installable when `INSTALL_INTERNAL_SKILLS=1` is set. Useful for work-in-progress skills or skills meant only for
  internal tooling.

```markdown
---
name: my-internal-skill
description: An internal skill not shown by default
metadata:
  internal: true
---
```

### Skill Discovery

The CLI searches for skills in these locations within a repository:

<!-- skill-discovery:start -->

- Root directory (if it contains `SKILL.md`)
- `skills/`
- `skills/.curated/`
- `skills/.experimental/`
- `skills/.system/`
- `.agents/skills/`
- `.augment/skills/`
- `.bob/skills/`
- `.claude/skills/`
- `./skills/`
- `.codebuddy/skills/`
- `.commandcode/skills/`
- `.continue/skills/`
- `.cortex/skills/`
- `.crush/skills/`
- `.factory/skills/`
- `.goose/skills/`
- `.junie/skills/`
- `.iflow/skills/`
- `.kilocode/skills/`
- `.kiro/skills/`
- `.kode/skills/`
- `.mcpjam/skills/`
- `.vibe/skills/`
- `.mux/skills/`
- `.openhands/skills/`
- `.pi/skills/`
- `.qoder/skills/`
- `.qwen/skills/`
- `.roo/skills/`
- `.trae/skills/`
- `.windsurf/skills/`
- `.zencoder/skills/`
- `.neovate/skills/`
- `.pochi/skills/`
- `.adal/skills/`
<!-- skill-discovery:end -->

### Plugin Manifest Discovery

If `.claude-plugin/marketplace.json` or `.claude-plugin/plugin.json` exists, skills declared in those files are also discovered:

```json
// .claude-plugin/marketplace.json
{
  "metadata": { "pluginRoot": "./plugins" },
  "plugins": [
    {
      "name": "my-plugin",
      "source": "my-plugin",
      "skills": ["./skills/review", "./skills/test"]
    }
  ]
}
```

This enables compatibility with the [Claude Code plugin marketplace](https://code.claude.com/docs/en/plugin-marketplaces) ecosystem.

If no skills are found in standard locations, a recursive search is performed.

## Compatibility

Skills are generally compatible across agents since they follow a
shared [Agent Skills specification](https://agentskills.io). However, some features may be agent-specific:

| Feature         | OpenCode | OpenHands | Claude Code | Cline | CodeBuddy | Codex | Command Code | Kiro CLI | Cursor | Antigravity | Roo Code | Github Copilot | Amp | OpenClaw | Neovate | Pi  | Qoder | Zencoder |
| --------------- | -------- | --------- | ----------- | ----- | --------- | ----- | ------------ | -------- | ------ | ----------- | -------- | -------------- | --- | -------- | ------- | --- | ----- | -------- |
| Basic skills    | Yes      | Yes       | Yes         | Yes   | Yes       | Yes   | Yes          | Yes      | Yes    | Yes         | Yes      | Yes            | Yes | Yes      | Yes     | Yes | Yes   | Yes      |
| `allowed-tools` | Yes      | Yes       | Yes         | Yes   | Yes       | Yes   | Yes          | No       | Yes    | Yes         | Yes      | Yes            | Yes | Yes      | Yes     | Yes | Yes   | No       |
| `context: fork` | No       | No        | Yes         | No    | No        | No    | No           | No       | No     | No          | No       | No             | No  | No       | No      | No  | No    | No       |
| Hooks           | No       | No        | Yes         | Yes   | No        | No    | No           | No       | No     | No          | No       | No             | No  | No       | No      | No  | No    | No       |

## Troubleshooting

### "No skills found"

Ensure the repository contains valid `SKILL.md` files with both `name` and `description` in the frontmatter.

### Skill not loading in agent

- Verify the skill was installed to the correct path
- Check the agent's documentation for skill loading requirements
- Ensure the `SKILL.md` frontmatter is valid YAML

### Permission errors

Ensure you have write access to the target directory.

## Environment Variables

| Variable                  | Description                                                                |
| ------------------------- | -------------------------------------------------------------------------- |
| `INSTALL_INTERNAL_SKILLS` | Set to `1` or `true` to show and install skills marked as `internal: true` |

```bash
# Install internal skills
INSTALL_INTERNAL_SKILLS=1 skills add vercel-labs/agent-skills --list
```

## Related Links

- [Agent Skills Specification](https://agentskills.io)
- [Skills Directory](https://skills.sh)
- [Amp Skills Documentation](https://ampcode.com/manual#agent-skills)
- [Antigravity Skills Documentation](https://antigravity.google/docs/skills)
- [Factory AI / Droid Skills Documentation](https://docs.factory.ai/cli/configuration/skills)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [OpenClaw Skills Documentation](https://docs.openclaw.ai/tools/skills)
- [Cline Skills Documentation](https://docs.cline.bot/features/skills)
- [CodeBuddy Skills Documentation](https://www.codebuddy.ai/docs/ide/Features/Skills)
- [Codex Skills Documentation](https://developers.openai.com/codex/skills)
- [Command Code Skills Documentation](https://commandcode.ai/docs/skills)
- [Crush Skills Documentation](https://github.com/charmbracelet/crush?tab=readme-ov-file#agent-skills)
- [Cursor Skills Documentation](https://cursor.com/docs/context/skills)
- [Firebender Skills Documentation](https://docs.firebender.com/multi-agent/skills)
- [Gemini CLI Skills Documentation](https://geminicli.com/docs/cli/skills/)
- [GitHub Copilot Agent Skills](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)
- [iFlow CLI Skills Documentation](https://platform.iflow.cn/en/cli/examples/skill)
- [Kimi Code CLI Skills Documentation](https://moonshotai.github.io/kimi-cli/en/customization/skills.html)
- [Kiro CLI Skills Documentation](https://kiro.dev/docs/cli/custom-agents/configuration-reference/#skill-resources)
- [Kode Skills Documentation](https://github.com/shareAI-lab/kode/blob/main/docs/skills.md)
- [OpenCode Skills Documentation](https://opencode.ai/docs/skills)
- [Qwen Code Skills Documentation](https://qwenlm.github.io/qwen-code-docs/en/users/features/skills/)
- [OpenHands Skills Documentation](https://docs.openhands.ai/modules/usage/how-to/using-skills)
- [Pi Skills Documentation](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/skills.md)
- [Qoder Skills Documentation](https://docs.qoder.com/cli/Skills)
- [Replit Skills Documentation](https://docs.replit.com/replitai/skills)
- [Roo Code Skills Documentation](https://docs.roocode.com/features/skills)
- [Trae Skills Documentation](https://docs.trae.ai/ide/skills)
- [Vercel Agent Skills Repository](https://github.com/vercel-labs/agent-skills)

## License

MIT
