# tk — tasks as an outline

## Data model

| Kind | Purpose | Allowed parent |
| --- | --- | --- |
| Project | A top-level initiative, e.g. *Office move*. | None |
| Task | A piece of work. A subtask is just a task with a task as its parent. | None, one project, or one task |
| Inbox item | Text to review later. No filing decisions at capture. | None until filed as a task |

**Project membership follows the outline.** Follow a task’s parents to the root: if the root is a project, the whole branch belongs to it. If the root is a task, the branch has no project. Tasks can nest to any depth. Moving a task moves its branch; you cannot move it under itself or a descendant.

Each item has a stable ID within its list, a title, and one state: `open` (default), `next`, `blocked`, `delegated`, or `done`. Completing a parent does not complete its children.

Optional details: `priority` (`1` high to `3` low), `when` (planned work date), `due` (deadline), and ordered free-text notes for links, blockers, or a delegate’s name. Dates use `YYYY-MM-DD`. Clear an optional field with `priority=`, `when=`, or `due=`.

## Build an outline

Starting with an empty list, these commands create the outline below. Writes print the affected ID.

```sh
tk project "Office move"                  # 1
tk add "Move the network"                 # 2 — starts without a project
tk add "Pack the router" parent=2         # 3
tk add "Label the cables" parent=3        # 4
tk set 2 parent=1                         # attach the whole branch to Office move
tk add "Submit expenses"                  # 5 — stays independent
```

```org
* Office move                 (project 1)
** Move the network           (task 2)
*** Pack the router           (task 3)
**** Label the cables         (task 4)
* Submit expenses             (task 5)
```

To browse one level, `tk list 2` shows immediate **unfinished** children: task 3, not task 4. `tk list 3` shows task 4. To take the network branch out of the project, use `tk set 2 parent=`; tasks 3 and 4 stay nested under it.

## Capture, review, act

```sh
tk inbox "Ask Mei about access to the new office"       # 6; or use `tk inbox -` for stdin
tk list inbox
tk set 6 kind=task parent=1 "title=Confirm key pickup" state=next
tk note 6 "Mei is at the front desk after 09:00."
tk set 6 when=2028-04-03 due=2028-04-05 priority=1
tk list next
tk show 6                                               # full item with notes
tk done 6
```

Filing keeps the capture’s ID. Retitle it in the same command and the original text becomes a note. Omit `parent` when filing for a standalone task, or use `kind=project` for a new initiative. To reopen an item, use `tk set ID state=open`.

`tk list` shows all unfinished work. Other views: `inbox`, `project`, `next`, `blocked`, `delegated`, `done`, `all`, or a parent ID. For skills, put `--json` first: `tk --json list all` returns an array with parent IDs where present; `tk --json add "Review draft"` returns the new item. Errors go to stderr with a nonzero exit status.

## Review work by tag

Any item can have flat tags; children do not inherit them. Use comma-separated names such as `office,follow-up` (letters `a-z`, digits, `-`, `_`; start with a letter or digit). Names are lowercased and duplicates count once.

```sh
tk add "Return spare keys" tags=office,errand
tk set 5 tags=office,finance   # replace all tags on item 5; tags= clears them
tk tags                        # each tag, then its unfinished task count
tk list tag=office             # the tasks behind that count
```

`tk tags` lists every attached tag alphabetically. It counts directly tagged **tasks** in `open`, `next`, `blocked`, or `delegated`, even if their parent is done. A tag used only on projects, inbox items, or done tasks shows `0`; removing its last use removes it from the report. `tk --json tags` returns rows like `{"tag":"office","count":2}` in an array.

Optional: [install the Markdown skill for Claude Code, Codex, Pi, or Cursor](skills/README.md).

## Build and choose a list

On Linux and macOS, use Go 1.22+ and a C compiler (GCC or Clang on Linux, Xcode Command Line Tools on macOS). Run `make` to test and build `bin/tk`, or `make install` to install it to `~/.local/bin`. No separate SQLite installation is needed.

Put the binary on `PATH` for the examples above. By default, `tk` uses `tk/tasks.db` under your user config directory. For separate work and personal lists, point `TK_FILE` at a different file for each, for example `export TK_FILE="$HOME/work/todos.db"`. The skill pins only the executable path; it does not change how `tk` selects a list.
