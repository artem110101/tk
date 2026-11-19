# tk

`tk` is a minimal to-do list for coding agents. Ask your agent to save a follow-up during a coding session, then pick it up in a later one. The list lives in a local SQLite file and is shared by agent sessions that use the same database.

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

## Security and offline use

- **Local and offline.** The `tk` binary makes no internet calls and has no telemetry, account, or sync service. Its only third-party Go dependency is `github.com/mattn/go-sqlite3`, which bundles SQLite. The rest uses the Go standard library.
- **Air-gapped use.** Build the binary on a trusted machine for the target OS and architecture, then transfer it with the skill file. Runtime use needs no network. To build from source inside the air gap, bring the toolchain, C compiler, and cached Go dependency with you. A first build with an empty cache normally downloads the dependency.
- **Agent access.** `tk` stores task text locally. When an agent reads a task, the text enters the agent’s context and is handled according to that agent’s settings. Use a local agent and model if task text must stay inside the air gap.

## Set up the skill in one workspace

Run `make install` to install the binary. Copy the skill file to the workspace path for your agent. [Skill file](skills/tk/SKILL.md)

| Agent | Workspace path |
| --- | --- |
| Claude Code | `.claude/skills/tk/SKILL.md` |
| Codex, Pi, Cursor | `.agents/skills/tk/SKILL.md` |

Replace `/absolute/path/to/tk` in the copied file with the full path to the installed binary, then restart your agent.
