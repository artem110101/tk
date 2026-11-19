# Install the tk skill

The entire skill is [tk/SKILL.md](tk/SKILL.md). Review it before copying. It is plain text: no executable, installer, hooks, MCP server, credentials, or tool grants. Use an approved `tk` binary from a trusted location, not a workspace-supplied executable.

**User install (Linux/macOS).** Run `make install-skill` from the repository root. It installs the binary to `~/.local/bin/tk` and looks for agent CLIs on `PATH` or in `~/.local/bin`. It installs `~/.claude/skills/tk/SKILL.md` for Claude Code and `~/.agents/skills/tk/SKILL.md` for Codex or Pi, skipping any existing skill file. If none are detected, it installs only the binary.

**Project-only install:** In a trusted workspace, create the folder below for your agent and copy `SKILL.md` into it. A file manager is enough. If a `tk` skill already exists, compare it rather than overwriting it. Do not copy the binary or the database into the skill folder.

| Agent with Agent Skills support | Destination relative to the workspace root |
| --- | --- |
| Claude Code | `.claude/skills/tk/SKILL.md` |
| Codex, Pi, Cursor | `.agents/skills/tk/SKILL.md` |

In your copy, replace both instances of `/absolute/path/to/tk` with the literal path to your approved binary (e.g. `/home/alex/.local/bin/tk` on Linux or `/Users/alex/.local/bin/tk` on macOS). Don't use `~`, `$HOME`, or a bare `tk` command. The skill does not set environment variables or choose a database; `tk` uses its usual list (by default, `tk/tasks.db` under your user config directory). Restart the agent session to pick up the change. No admin rights or changes to approval settings are needed. If your organization disables local skills or command execution, do not work around that policy.

**Smoke test:** Run the absolute executable path with `help` to check it without opening a database. To test the skill itself without reading task data, ask the agent to use the `tk` skill to show CLI help. To uninstall, remove only the copied `SKILL.md` and restart; no tasks are deleted.

**Corporate use:** The skill requests no network access, but command output enters the agent’s context and may be sent to its model provider or logged. Use only data approved for that agent. The instructions are not a sandbox: keep your organization’s normal tool approvals, filesystem restrictions, and review process in place.
