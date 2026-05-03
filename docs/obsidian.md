# mdm + Obsidian

Bring your AI agent skill library into Obsidian — your markdown vault becomes a living prompt reference, browsable alongside your notes and immediately usable with Obsidian's AI plugins.

## How it works

When you run `mdm skills add` from inside an Obsidian vault directory, MDM detects the vault (by the presence of `.obsidian/`) and offers Obsidian as an install target. Skills are placed in a `Prompts/` folder at the vault root as plain markdown files — fully visible in Obsidian's file explorer, linkable from notes, and compatible with every AI plugin that accepts a folder of prompts.

```
your-vault/
├── .obsidian/          ← vault config (MDM never touches this)
├── Prompts/            ← MDM installs skills here
│   ├── code-review/
│   │   ├── SKILL.md
│   │   └── review.md
│   └── writing-polish/
│       ├── SKILL.md
│       └── polish.md
├── Notes/
└── ...
```

## Install skills into your vault

From inside the vault directory:

```bash
cd ~/my-vault
mdm skills add github-user/my-skill-repo --agent obsidian
```

Or let MDM prompt you interactively — it will offer Obsidian when it detects the vault:

```bash
mdm skills add github-user/my-skill-repo
```

## Wiring skills to Obsidian AI plugins

### Obsidian Copilot

Obsidian Copilot can load any folder of `.md` files as custom prompt commands, invokable from the command palette.

1. Open **Settings → Copilot → Custom Prompts**
2. Set **Custom Prompts Folder Name** to `Prompts`
3. Skills installed by MDM now appear as Copilot custom prompt commands — no restart needed, changes are picked up automatically

### Templates / Templater

Both the core Templates plugin and the Templater community plugin can use `Prompts/` as a template source, letting you insert any skill prompt directly into a note.

**Core Templates plugin:**
1. Enable it under **Settings → Core plugins → Templates**
2. Set **Template folder location** to `Prompts`
3. Use **Insert template** from the command palette to insert a skill into the current note

**Templater community plugin:**
1. Install Templater from Community Plugins
2. Set **Template folder location** to `Prompts` in Templater settings
3. Templater's `tp.*` syntax works inside skill files, so you can add dynamic variables to a skill without changing the base markdown

### Text Generator

In the Text Generator plugin settings, set the **Templates Path** to `Prompts` to make MDM skills available as generation templates.

## Vault vs. repo — when to use each

This is the most important decision in the MDM + Obsidian workflow. The same skill can live in both places for different purposes.

### Install to your Obsidian vault when

- You want a **personal prompt library** — prompts you reach for in writing, research, and thinking, not tied to any single project
- You want to **annotate and remix** skills: add notes, link to related concepts in your vault, track what works
- You're using **Obsidian AI plugins** (Copilot, Text Generator, Templater) and want skills available as commands or templates during writing sessions
- The prompts are about **how you work**, not about a specific codebase — things like "refine this paragraph", "extract action items", "summarize meeting notes"
- You want skills **accessible outside the terminal**, browsable like regular notes

### Install to your repo (agent rules/skills) when

- The skills are **project-specific** — they encode context about your codebase, stack, architecture, or team conventions
- You want them **versioned alongside code** so they evolve with the project and are reviewable in PRs
- They are consumed **directly by a coding AI agent** (Claude Code reads `.claude/skills/`, Cursor reads `.cursor/skills/`, etc.) — the agent picks them up automatically without any user action
- Multiple developers need **the same skill set** when working in the repo
- The skills define **what the AI should know about this repo** — things like "always use the Result type for errors", "our API follows REST conventions defined in docs/api.md"

### The overlap

A "write unit tests" skill might live in both:

| Location | Purpose |
|---|---|
| `repo/.claude/skills/write-tests/` | Claude Code uses it automatically when you ask it to add tests — it knows your test framework, naming conventions, and fixture patterns |
| `vault/Prompts/write-tests/` | You invoke it manually from Copilot when drafting a test spec in your notes, or insert it as a template when planning a testing strategy |

The repo copy is optimized for the AI agent (tightly scoped, references project-specific patterns). The vault copy is optimized for you (readable, annotatable, part of your thinking workflow).

## Tips

**Keep vault prompts general.** Repo skills should reference project specifics (file paths, framework versions, team conventions). Vault prompts should work across projects — they're about your patterns, not a codebase's.

**Link skills to your notes.** Because skills are plain markdown in your vault, you can link to them with `[[Prompts/code-review/review]]` from any note. Reference a skill from a project planning note, a retrospective, or a how-to guide.

**Update skills with mdm.** Rather than editing installed skill files directly, run `mdm skills update` to pull the latest version from the source. If you've made personal annotations, put them in a separate note that links to the skill file.

**Scope your Copilot prompts folder.** If you have a large vault with many prompts, you can point Obsidian Copilot at a subfolder like `Prompts/AI/` instead of the whole `Prompts/` root. Use `--skill` when adding to filter which skills get installed:

```bash
mdm skills add github-user/big-skill-repo --skill writing --agent obsidian
```
