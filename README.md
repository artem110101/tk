# tk

`tk` is a minimal to-do list for coding agents. Ask your agent to save a follow-up during a coding session, then pick it up in a later one. The list lives in a local SQLite file and is shared by agent sessions that use the same database.

## Security and offline use

- **No runtime network access.** `tk` makes no internet calls and has no telemetry, account, or sync service. It uses the Go standard library and one third-party Go module, `go-sqlite3` v1.14.53, which bundles SQLite 3.54.1. [Dependencies](go.mod)
- **Air-gapped operation.** Transfer a trusted binary built for the target OS and architecture, along with the skill file. To build inside the air gap, bring the Go toolchain, C compiler, and cached dependency. A first build with an empty cache normally downloads the dependency.
- **Agent permissions.** The skill is a single Markdown file. It directs the agent to use an approved absolute path to `tk`, read only what the request needs, write only when asked, and treat task text as data. It uses the agent’s existing permissions and requires no credentials, hooks, or MCP server. [Skill source](skills/tk/SKILL.md)
- **Task data.** `tk` stores task text in a local SQLite database without application-level encryption. On Linux, a newly created store was verified to use `0600` for the database and `0700` for its directory. The agent host controls whether task text reaches a model provider or logs. Use a local agent and model to keep that text inside the air gap.

**Verification for v0.1.0.** The Linux task workflow passed with networking disabled, and `go mod verify` passed. `govulncheck` v1.7.0 reported no vulnerabilities in the Go code it scanned. That scan did not cover the bundled SQLite C code or the agent host.

## What you can do

| Use | How it helps |
| --- | --- |
| **Inbox** | Capture a loose idea or follow-up while you’re working. File it as a task or project when you know where it belongs. |
| **Projects and subtasks** | Group related work and break a task into smaller steps. Every item has a stable ID, so the agent can refer to it again after a move or rename. |
| **Tags** | Keep a view of work across projects, such as `release` or `follow-up`, and see how many unfinished tasks use each tag. |
| **Notes** | Keep links, decisions, and handoff details with the task while keeping the title short. |
| **Next actions** | Mark what to do next, what’s blocked, and what’s delegated. Add a priority, planned date, or deadline when it helps. |

## Install on Linux or macOS

You’ll need Make, Go 1.22 or later, and a C compiler. Use GCC or Clang on Linux, or the Xcode Command Line Tools on macOS. SQLite is bundled.

From the repository root, run

```sh
make install-skill
```

This installs `tk` to `~/.local/bin/tk` and adds the skill for Claude Code, Codex, or Pi when their CLI is on `PATH` or in `~/.local/bin`. It preserves existing skill files. If your agent isn’t detected, use the workspace setup below. Restart your agent after installation.

## A typical session

1. **Capture a follow-up while you work.** “Add ‘confirm the rollback plan’ to my inbox.”
2. **File it when you’re ready.** “Move that item under Launch, tag it `release`, and mark it next.”
3. **Leave context for later.** “Add a note that the rollback steps are in the deploy runbook.”
4. **Pick it back up in a new session.** “Show my next actions tagged `release`.”

## Use a different list

The default database is `tk/tasks.db` under your user config directory. Set `TK_FILE` to an absolute database path in the environment where your agent runs to use a separate list.

## Set up the skill in one workspace

Run `make install` to install the binary. Copy the skill file to the workspace path for your agent. [Skill file](skills/tk/SKILL.md)

| Agent | Workspace path |
| --- | --- |
| Claude Code | `.claude/skills/tk/SKILL.md` |
| Codex, Pi, Cursor | `.agents/skills/tk/SKILL.md` |

Replace `/absolute/path/to/tk` in the copied file with the full path to the installed binary, then restart your agent.
