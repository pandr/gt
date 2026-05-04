# Demo GIF — maintainer notes

## Files

| File | Role |
|------|------|
| `docs/demo-setup.sh` | Creates a throwaway git repo at `/tmp/gt-demo` with a specific working-tree state |
| `docs/demo.tape` | VHS script that drives the recording |
| `docs/demo.gif` | Output referenced by README.md |

## Regenerating

```sh
vhs docs/demo.tape   # run from project root; takes ~30 s
```

Requires `vhs` (`brew install vhs`). The tape rebuilds the binary and the fixture repo itself, so no manual setup is needed.

## What the demo shows (in order)

1. `gt` opens on the fixture repo — branch header, four sections, 3 commits
2. `j` navigates to `docs/` in Untracked; `l` expands it (reveals `docs/api.md`, `docs/guide.md`); `h` collapses
3. `j j j` navigates to `README.md` in Unstaged; `d` opens the pager showing the `+## Usage` hunk; `q` closes
4. `s` stages `README.md` — it moves to Staged
5. `t` tags `src/main.go` (cursor lands there after the stage refresh); `k k` reaches `src/handler.go`; `t` again — status bar shows `[2 tagged]` with magenta `*` across Untracked and Unstaged sections
6. `;s` batch-stages both tagged files — Staged(4), Unstaged(0)
7. `c` opens the commit prompt; message typed; Enter commits

## Fixture repo layout (critical for navigation)

After `demo-setup.sh` runs, `git status --porcelain` shows:

```
 M README.md       ← unstaged modification
 M src/main.go     ← unstaged modification
M  src/util.go     ← staged modification
?? docs/           ← untracked directory (contains api.md, guide.md)
?? src/handler.go  ← untracked file
```

Git outputs untracked entries alphabetically, so `docs/` sorts before `src/handler.go`. The gt row list is therefore:

```
Row 0:  Untracked (2)      ← section header; cursor starts here
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

After staging `README.md` the cursor stays at index 4, which is now `src/main.go` (the section shrinks by one). From there `k k` lands on `src/handler.go` (index 2), not `docs/` (index 1) — that's why two `k` presses, not three.

## Key-hint overlay (GT_DEMO_KEYS=1)

The yellow popup box is built into gt, gated by `GT_DEMO_KEYS=1`. Implementation:

- `internal/ui/model.go` — `keyHint string`, `keyHintToken int` fields
- `internal/ui/update.go` — `setKeyHint(m, key)` sets the hint and schedules a `clearKeyHintMsg` after 700 ms; a token counter ensures stale timers from chained keys (`;` then `;s`) don't clobber each other
- `internal/ui/view.go` — `overlayKeyHint()` splices the box into the bottom-right corner of the rendered screen string using `spliceAt()`, which is ANSI-aware; `styleKeyHint` defines the yellow style

The cursor row highlight was also improved here: `styleCursor` background changed from ANSI 237 (nearly invisible on Dracula) to 241.

## VHS notes

- Named keys are bare words: `Enter`, `Escape` — no `Key` prefix (that syntax does not exist in vhs 0.11)
- `Hide` / `Show` surround the setup block; everything in Hide is not recorded
- `TypingSpeed 70ms` applies to `Type` calls; individual key presses (`Enter`) are not affected
- The tape rebuilds the binary on every run (`go build -o /tmp/gt-demo-bin`) so the recording always reflects current source
