# mdm skills update

Update installed skills to their latest versions.

## Usage

```
mdm skills update [skills...]
```

Re-fetches each skill from its recorded source and ref in the lock file. Skills that are already up to date are skipped.

Alias: `check`

## Up-to-date detection

| Source | Method |
|---|---|
| GitHub | Skill folder hash — only triggers when the skill's own directory changed, not the rest of the repo. Falls back to commit SHA on API error. |
| GitLab / other git | Commit SHA via `git ls-remote` — no clone needed |
| Local path | `version` field in `SKILL.md` — bump it to trigger an update |

Remote skills use commit-based detection because it's automatic — every push is detectable without any author action. Local skills have no remote to query, so version comparison is used instead: mdm reads the `version` field from the source `SKILL.md` and compares it to the value recorded at install time. If no version is present in either the source or the lock entry, the skill is skipped with a warning.

If no hash is recorded for a remote skill (older install), it is re-fetched once to populate tracking data.

## Scope

Without `--global` or `--project`, a prompt asks which scope to update:

```
? Update which scope?
  ● Both (project and global)
    Project
    Global
```

With `--yes`, both scopes are updated without prompting.

## Flags

| Flag | Description |
|---|---|
| `--global, -g` | Update global skills only |
| `--project, -p` | Update project skills only |
| `--yes, -y` | Skip scope prompt, update both scopes |

## Examples

```bash
# Update all skills (both scopes, with prompt)
mdm skills update

# Update a specific skill
mdm skills update my-skill

# Update only global skills
mdm skills update -g

# Update both scopes without prompting
mdm skills update -y
```
