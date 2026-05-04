# Demo GIFs — maintainer notes

## Files

| File | Role |
|------|------|
| `docs/demo-setup.sh` | Creates a throwaway git repo at `/tmp/gt-demo` with a specific working-tree state (shared by all tapes) |
| `docs/demo-overview.tape` | VHS script — launch gt, navigate, expand a directory |
| `docs/demo-diff.tape` | VHS script — diff a changed file, diff a commit |
| `docs/demo-stage-commit.tape` | VHS script — stage files one by one, commit inline |
| `docs/demo-tag-batch.tape` | VHS script — tag files across sections, batch-stage |
| `docs/demo-overview.gif` | Output referenced by README.md |
| `docs/demo-diff.gif` | Output referenced by README.md |
| `docs/demo-stage-commit.gif` | Output referenced by README.md |
| `docs/demo-tag-batch.gif` | Output referenced by README.md |

## Regenerating

Run each tape from the project root (each takes ~30 s):

```sh
vhs docs/demo-overview.tape
vhs docs/demo-diff.tape
vhs docs/demo-stage-commit.tape
vhs docs/demo-tag-batch.tape
```

Requires `vhs` (`brew install vhs`). Every tape rebuilds the binary and the fixture repo, so no manual setup is needed.

## What each demo shows

### demo-overview
1. `gt` is typed at the shell prompt (binary is on PATH as `gt`)
2. Cursor starts at Untracked header; `j` navigates to `docs/`; `l` expands it; `h` collapses
3. Several `j` / `k` presses cruise through all four sections and the commit list

### demo-diff
1. Navigate to `README.md` in Unstaged (4 × `j` from top)
2. `d` opens the pager showing the `+## Usage` hunk; `q` closes
3. `G` jumps to the oldest commit; `d` diffs the whole commit; `q` closes

### demo-stage-commit
1. Navigate to `README.md` in Unstaged (4 × `j`)
2. `s` stages it — it moves to Staged; cursor lands on `src/main.go`
3. `s` stages `src/main.go`
4. `c` opens the inline commit prompt; message typed; Enter commits

### demo-tag-batch
1. Navigate to `src/handler.go` in Untracked (2 × `j`)
2. `t` tags it — magenta `*` appears
3. 3 × `j` reaches `src/main.go` in Unstaged; `t` tags it — status bar shows `[2 tagged]`
4. `;s` batch-stages both tagged files

## Fixture repo layout

After `demo-setup.sh` runs, `git status --porcelain` shows:

```
 M README.md       ← unstaged modification
 M src/main.go     ← unstaged modification
M  src/util.go     ← staged modification
?? docs/           ← untracked directory (contains api.md, guide.md)
?? src/handler.go  ← untracked file
```

Git outputs untracked entries alphabetically, so `docs/` sorts before `src/handler.go`. The gt row list is:

```
Row 0:  Untracked (2)      ← cursor starts here
Row 1:    ▶ docs/
Row 2:    ? src/handler.go
Row 3:  Unstaged (2)
Row 4:    M README.md
Row 5:    M src/main.go
Row 6:  Staged (1)
Row 7:    M src/util.go
Row 8:  (separator)
Row 9:  Working tree  ./
Row 10: (separator)
Row 11: Recent commits
Row 12:   <sha>  tag v1.0.0
Row 13:   <sha>  add install instructions
Row 14:   <sha>  initial commit
```

After staging `README.md` the cursor stays at index 4, which is now `src/main.go` (the section shrinks by one).

## Key-hint overlay (GT_DEMO_KEYS=1)

The yellow popup box is built into gt, gated by `GT_DEMO_KEYS=1`. Implementation:

- `internal/ui/model.go` — `keyHint string`, `keyHintToken int` fields
- `internal/ui/update.go` — `setKeyHint(m, key)` sets the hint and schedules a `clearKeyHintMsg` after 700 ms; a token counter ensures stale timers from chained keys (`;` then `;s`) don't clobber each other
- `internal/ui/view.go` — `overlayKeyHint()` splices the box into the bottom-right corner of the rendered screen string using `spliceAt()`, which is ANSI-aware; `styleKeyHint` defines the yellow style

## Terminal identity trick

Each tape's hidden setup section copies the demo binary to `/tmp/gt-demo-path/gt` and prepends that directory to `$PATH`. This means the visible `Show` section simply types `gt` — matching what a real user would run.

## VHS notes

- Named keys are bare words: `Enter`, `Escape` — no `Key` prefix (that syntax does not exist in vhs 0.11)
- `Hide` / `Show` surround the setup block; everything in Hide is not recorded
- `TypingSpeed 70ms` applies to `Type` calls; individual key presses (`Enter`) are not affected
- Each tape rebuilds the binary (`go build -o /tmp/gt-demo-bin`) so recordings always reflect current source
