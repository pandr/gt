# gt — v1 plan

A fast, mutt-inspired TUI for the git operations Peter does most often. Inspired by `gitu`, but trimmed to the actions actually used and built around fast per-file/per-section diffing.

See `IDEA.txt` for the original sketch and motivation.

## Principles

- **Slim v1.** Only build what's needed for daily `gs` → stage → commit flow. Every additional feature must be motivated by a real missing-it moment, not speculation.
- **Mutt ergonomics.** Single dense screen, vim-style navigation, tag-and-act selection model. Discoverable via `?`.
- **Trust git.** Shell out to the `git` CLI; rely on the user's gitconfig for pager, color, hooks, signing. No `go-git`.
- **No file watching, no daemon.** Refresh on action. ~100ms cold start is acceptable.

## Tech choices

- **Language:** Go.
- **TUI:** `bubbletea` + `bubbles` (textinput) + `lipgloss` for styling. `tea.ExecProcess` for suspending the TUI to run `less` / `$EDITOR`.
- **Invocation:** `gt` only. Full-screen TUI. Errors out if cwd is not inside a git repo.
- **Paths:** displayed repo-root-relative (matches `git status`, unambiguous across subdirs).

## Architecture

```
cmd/gt/main.go           entry point — verify git repo, launch bubbletea
internal/git/            thin wrappers around the git CLI
  status.go              parse `git status --porcelain=v2 --branch -z`
  log.go                 `git log -n 6 --oneline --decorate=short`
  diff.go                build commands for `git diff [--staged] [-- <file>]` / `git show <sha>`
  stage.go               `git add <file>` / `git restore --staged <file>`
  commit.go              `git commit -m <msg>` / `git commit -F <file>`
internal/ui/
  model.go               bubbletea Model: status, log, cursor, tag set, mode (normal / commit-input / tag-prefix)
  update.go              key handling, action dispatch, refresh after mutations
  view.go                rendering — sections, branch header, recent commits, status bar
  keys.go                key bindings table (single source of truth, also feeds `?` help)
  pager.go               ExecProcess helpers for diff and `$EDITOR`
```

**Why git CLI, not go-git:** parity with the user's pager, color, hooks, signing, and config comes for free. Porcelain v2 is trivial to parse. The git CLI is not the bottleneck for this tool.

## Screen layout

```
On branch feature/user-dashboard        ↑2 ↓0 origin/feature/...

Untracked (3)
   scripts/bundle.sh
   TODO.md
   tests/.../legacy-module/

Unstaged (8)
 M .gitignore
 M README.md
*M tests/.../index.json                       ← cursor
 M tests/.../foo.csproj
 ...

Staged (0)

Recent commits
 a1b2c3d fix cache invalidation
 e4f5a6b add info command
 ...

[1 tagged]                              c=commit  d=diff  ?=help
```

- Cursor moves over every row, **including section headers**.
- Section headers are valid action targets: `d` on "Unstaged" diffs every unstaged file together; `s` on "Unstaged" stages all of them.
- Tagged rows are visually marked (e.g. `*` prefix or color).

## Key bindings (mutt-flavored)

| Key | Action |
|---|---|
| `j` / `k` | down / up |
| `g` / `G` | top / bottom |
| `Ctrl-d` / `Ctrl-u` | half page down / up |
| `d` | diff at cursor (file → that file; section header → whole section; commit row → `git show`) |
| `s` | stage at cursor (file or whole Unstaged/Untracked section) |
| `u` | unstage at cursor (file or whole Staged section) |
| `t` | toggle tag on cursor row |
| `;<cmd>` | apply `<cmd>` to all tagged rows (`;s` stage all tagged, `;u` unstage all tagged, `;d` diff all tagged) |
| `T` | clear all tags |
| `c` | open one-line commit prompt at bottom |
| `R` | force refresh (auto-runs after every mutation anyway) |
| `?` | help overlay |
| `q` | quit |

**Commit prompt** (entered via `c`):
- `Enter` — commit with the typed message.
- `Ctrl-g` — write current buffer to a tempfile, `ExecProcess` `$EDITOR <tempfile>`, read back, run `git commit -F <tempfile>`. (Same UX as Claude Code's editor escape.)
- `Esc` — cancel.

## Implementation notes

**Status parsing.** Use `git status --porcelain=v2 --branch -z`. v2 gives stable, scriptable output with rename info; `-z` avoids quoting headaches. Bucket entries into Untracked / Unstaged / Staged based on the porcelain fields (a single file can appear in both Unstaged and Staged if it has both index and worktree changes — render in both sections).

**Recent commits.** `git log -n 6 --oneline --decorate=short`. Show alongside any decorations (branch/tag refs).

**Diff dispatch (`d`).** Build the right `git` invocation for the cursor target, then `tea.ExecProcess` it. Git handles pager (`less` or whatever `core.pager` is set to) and color automatically when stdout is a tty.
- File in Untracked → `git diff --no-index /dev/null <file>` (or just show the file content; pick whichever feels right when building).
- File in Unstaged → `git diff -- <file>`.
- File in Staged → `git diff --staged -- <file>`.
- Section header → same commands without the `-- <file>`, scoped by section.
- Commit row → `git show <sha>`.
- Tagged set with `;d` → one combined diff if all tags are in the same section; otherwise show them sequentially or refuse with a hint.

**Tag state.** `map[string]bool` keyed by repo-root-relative path. Survives refresh; prune entries whose paths no longer appear in status.

**Refresh model.** After any mutating action: re-run status + log, redraw. No filesystem watcher.

**Error display.** Transient single-line toast at the bottom of the screen for non-fatal git errors (e.g. commit hook failure). Cleared on next keypress.

**Repo detection.** On startup, run `git rev-parse --show-toplevel`; if it fails, print a friendly error and exit non-zero.

## v1 punch list (build in this order)

1. **Skeleton.** `cmd/gt` builds a binary; bubbletea loop runs and quits cleanly; errors if not in a git repo.
2. **Readers.** `internal/git/status.go` and `log.go` — parse porcelain v2 and oneline log into typed structs.
3. **Render.** `view.go` — branch header, four sections (Untracked / Unstaged / Staged / Recent commits), status bar.
4. **Navigation.** `j/k/g/G/Ctrl-d/Ctrl-u`; cursor over rows and section headers.
5. **Diff.** `d` shells out via `ExecProcess`; file / section / commit cases.
6. **Stage / unstage.** `s` / `u`; refresh after; section-header targeting.
7. **Tagging.** `t`, `;<cmd>`, `T`; visual marker on tagged rows.
8. **Commit.** `c` prompt; `Enter` commits; `Ctrl-g` opens `$EDITOR`; `Esc` cancels.
9. **Polish.** Toast for git errors; `?` help overlay generated from the keys table.

Target: ~800–1000 LOC.

## Explicitly out of scope for v1

Push / pull / fetch · branch create/switch/delete · stash apply/pop/drop · rebase · merge · conflict resolution UI · hunk-level staging (`git add -p`) · log browsing beyond last 6 · file-watcher live refresh · non-TUI subcommands (`gt status`, etc.) · search / filter · settings file · themes.

Each of these gets added only when Peter actually reaches for it and finds it missing.

## Ideas / todo

Untracked folders. I need to be able to 'see' them and perhaps add them? I can stage them now, but its a bit yolo

Push. I should be able to navigate to the top / headline where gt shows the branch stuff and do push.
