# gt

A fast, keyboard-driven TUI for the git operations you do every day. Inspired by fond memories of [mutt](http://www.mutt.org/) — a single dense screen, vim-style navigation, and tag-and-act selection so you can stage, diff, and commit without touching the mouse.

**Instant overview of your repo state**

`j`/`k` to move, `l`/`h` to expand or collapse a directory.

![overview](docs/demo-overview.gif)<br><br>

**Diff any changed file or commit from history**

`j`/`G` to navigate, `d` or `space` to open the diff, `q` to close.

![diff](docs/demo-diff.gif)<br><br>

**Stage files and commit inline**

`s` to stage, `c` to open the commit prompt, `Enter` to commit.

![stage and commit](docs/demo-stage-commit.gif)<br><br>

**Tag files across sections and act on all of them at once**

`t` to tag a file (cursor advances automatically), `;s` to stage everything tagged.

![tag and batch](docs/demo-tag-batch.gif)

## Features

- See untracked, unstaged, staged files and recent commits on one screen
- Expandable directory rows — `l` / `→` to drill in, `h` / `←` to collapse
- Working tree browser: explore all tracked files under cwd, untrack or delete them
- `d` on any row to diff it — file, whole section, or a commit
- `s` / `u` to stage / unstage files or entire sections
- Tag multiple files with `t`, then act on all of them with `;s`, `;u`, or `;d`
- Inline commit prompt; `Ctrl-g` drops into `$EDITOR` for longer messages
- Respects your gitconfig: pager, colors, hooks, signing

## Build

```sh
go build -ldflags="-s -w" -o gt ./cmd/gt/
```

Requires Go 1.21+. No other dependencies to install — everything is statically linked.

Optionally put the binary somewhere on your `$PATH`:

```sh
mv gt /usr/local/bin/gt
```

## Usage

Run `gt` inside any git repository:

```sh
gt
```

### Key bindings

| Key | Action |
|---|---|
| `j` / `k` | down / up |
| `g` / `G` | top / bottom |
| `Ctrl-d` / `Ctrl-u` | half page down / up |
| `l` / `→` | expand directory or section |
| `h` / `←` | collapse directory or section |
| `enter` | open — expand dir/commit, diff file or section |
| `d` / `space` | diff at cursor (file, section, or commit) |
| `s` | stage at cursor |
| `u` | unstage at cursor |
| `r` | restore — discard unstaged changes (asks y/n) |
| `x` | untrack file, keep on disk (`git rm --cached`) |
| `X` | untrack and delete file from disk (asks y/n) |
| `v` | view file in pager |
| `e` | open file in editor |
| `t` | toggle tag on row (cursor advances) |
| `;s` / `;u` / `;d` | stage / unstage / diff all tagged |
| `T` | clear all tags |
| `c` | commit prompt (`Ctrl-g` for `$EDITOR`, `Esc` to cancel) |
| `A` | amend last commit — only allowed before push |
| `!` | run a shell command, then refresh |
| `R` | refresh |
| `?` | help |
| `q` / `Ctrl-c` | quit |

Cursor targets are context-aware: `d` on a section header diffs the whole section; `s` on the Unstaged header stages everything unstaged; `l` on an untracked directory expands it to show individual files.
