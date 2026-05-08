# Demo GIFs — maintainer notes

## Files

| File | Role |
|------|------|
| `docs/demo-setup.sh` | Creates a throwaway git repo at `/tmp/gt-demo` with a specific working-tree state (shared by all tapes) |
| `docs/demo-overview.tape` | VHS script — launch gt, navigate, expand a directory |
| `docs/demo-diff.tape` | VHS script — inline diff a changed file, inline diff a commit |
| `docs/demo-stage-commit.tape` | VHS script — stage files one by one, commit inline |
| `docs/demo-amend.tape` | VHS script — navigate to latest commit, amend its message |
| `docs/demo-overview.gif` | Output referenced by README.md |
| `docs/demo-diff.gif` | Output referenced by README.md |
| `docs/demo-stage-commit.gif` | Output referenced by README.md |
| `docs/demo-amend.gif` | Output referenced by README.md |

## Regenerating

Run each tape from the project root (each takes ~30 s):

```sh
vhs docs/demo-overview.tape
vhs docs/demo-diff.tape
vhs docs/demo-stage-commit.tape
vhs docs/demo-amend.tape
```

Requires `vhs` (`brew install vhs`). Every tape rebuilds the binary and the fixture repo, so no manual setup is needed.

## What each demo shows

### demo-overview
1. `gt` is typed at the shell prompt
2. Cursor starts at Untracked header; `j` navigates to `docs/`; `l` expands it; `h` collapses
3. Several `j` / `k` presses cruise through all four sections and the commit list

### demo-diff
1. Navigate to `README.md` in Unstaged (4 × `j` from top)
2. `d` opens the inline diff; `q` closes back to the overview
3. `G` jumps to the oldest commit; `k` up to "add install instructions"; `l` expands it
4. `j` to a commit file; `d` opens its inline diff; `q` closes

### demo-stage-commit
1. Navigate to `README.md` in Unstaged (4 × `j`)
2. `s` stages it — it moves to Staged; cursor lands on `src/main.go`
3. `s` stages `src/main.go`
4. `c` opens the inline commit prompt; message typed; Enter commits

### demo-amend
1. `G` jumps to the oldest commit; `k k` navigates up to the most recent commit
2. `A` opens the amend prompt with the current message prefilled
3. Message cleared with `Ctrl+u`, new message typed, Enter amends

## Fixture repo layout

After `demo-setup.sh` runs, `git status --porcelain` shows:

```
 M README.md       ← unstaged modification
 M src/main.go     ← unstaged modification
M  src/util.go     ← staged modification
?? docs/           ← untracked directory (contains api.md, guide.md)
?? src/handler.go  ← untracked file
```

The gt row list (cursor starts at row 0):

```
Row 0:  Untracked · 2
Row 1:    ▶ docs/
Row 2:    ? src/handler.go
Row 3:  Unstaged · 2  +N/-N
Row 4:    M README.md  +N/-N
Row 5:    M src/main.go  +N/-N
Row 6:  Staged · 1  +N/-N
Row 7:    M src/util.go  +N/-N
Row 8:  (separator)
Row 9:  Working tree  ./
Row 10: (separator)
Row 11: Recent commits
Row 12:   ▶ <sha>   <age>  tag v1.0.0
Row 13:   ▶ <sha>   <age>  add install instructions
Row 14:   ▶ <sha>   <age>  initial commit
```

After staging `README.md` the cursor stays at index 4, which is now `src/main.go`.

## VHS notes

- Named keys are bare words: `Enter`, `Escape` — no `Key` prefix
- `Hide` / `Show` surround the setup block; everything in Hide is not recorded
- `TypingSpeed 70ms` applies to `Type` calls; individual key presses (`Enter`) are not affected
- Each tape rebuilds the binary (`go build -o /tmp/gt-demo-bin`) so recordings always reflect current source
