# Handoff: `gt` — Overview Screen Refinement

## Overview

This package contains a design refinement for the **overview screen** of [`gt`](https://github.com/pandr/gt) — a fast, mutt-inspired TUI for daily git operations, built in Go on top of `bubbletea` / `lipgloss`.

The work focuses on sharpening visual hierarchy, color discipline, and density on the main overview screen — without changing the keymap, the section model, or the mutt-flavored conventions the tool already gets right. It also sketches a few adjacent surfaces (inline diff, stash) that follow the same grammar.

---

## About the Design Files

The HTML files in this bundle are **design references**, not production code.

They are 1920×1080 deck slides that simulate a terminal in CSS — every "row" is an HTML `<span>` styled to align on a fixed character cell. This is a fidelity tool for showing intent; **it is not the implementation**.

The implementation target is the existing Go codebase at `github.com/pandr/gt`. The work is to translate the design into:

- **`internal/ui/view.go`** — render functions, lipgloss styles
- **`internal/ui/model.go`** — only if state needs new fields (e.g. for stash, expanded commits)
- **`internal/ui/keys.go`** — only if new key bindings are added

Use the existing `lipgloss` patterns. Do not introduce a new rendering library.

---

## Fidelity

**High-fidelity.** Final colors, glyphs, type rhythm, and spacing are all decided. Hex values below are authoritative — they map directly to lipgloss `lipgloss.Color("#…")` calls.

The mockups exaggerate the terminal cell to ~14px / 19px line-height for legibility on a slide; in a real terminal these are character cells, so all spacing in the design is expressed in **cells**, not pixels.

---

## What's in the Deck

The deck is organized as a design narrative. In implementation terms, the slides that matter are:

| Slide | What it shows | What to implement |
|---|---|---|
| 03 — Baseline | Current overview, annotated. | Reference only. |
| 05 — Overview refined | **The target design.** | Yes. This is the main deliverable. |
| 06 — Before vs after | Side-by-side comparison + diff list. | Use the diff list as a checklist. |
| 07 — Inline diff (unified) | Refined diff view. | Yes — replaces current diff render. |
| 08 — Inline diff (split) | Two-pane variant of the diff. | Optional — gate behind `v` (toggle). |
| 09 — Stash | Stash as a section in the overview. | Yes — adds a new section. |
| Other slides | Title, palette, glyph kit, system pages | Reference. |

Open `gt - TUI design.html` in a browser and use ←/→ to navigate. The right column on each slide is design rationale — read it.

---

## Design Tokens

### Colors

Drop these into `internal/ui/theme.go` (new file) as `lipgloss.Color` constants. Background is the terminal default — do not paint it.

| Token | Hex | Use |
|---|---|---|
| `term.bg` | `#14141a` | Terminal background (assumed; do not set) |
| `term.bg.soft` | `#1a1a22` | Subtle band (rarely used) |
| `term.fg` | `#d8d4c7` | Default text |
| `term.fg.soft` | `#a39e8d` | De-emphasized text (counts, "Recent commits" header) |
| `term.fg.faint` | `#5b5849` | Hint bar, separators, inactive metadata |
| `term.fg.ghost` | `#34322b` | The dotted rail when section has no work |
| `iris` | `#7fb8c4` | Branch names, ahead arrow, identity |
| `iris.dim` | `#4d7e8a` | Behind arrow, commit SHAs |
| `amber` | `#d6a96a` | Section headers WITH work, modified indicator |
| `amber.dim` | `#8a6e44` | Rail under an active section's files |
| `add` (moss) | `#94b87a` | `?` `A` `+` |
| `del` (terracotta) | `#c87766` | `D` `-` |
| `tag` (mauve) | `#b491c8` | `*` (tagged-row marker) |
| `cursor.bg` | `#2a2a35` | Cursor row fill (background, not bar) |

Contrast targets met: fg/bg 13.4:1, iris/bg 6.1:1, amber/bg 7.2:1 — all WCAG AA at body size.

### Glyph Kit

| Glyph | Meaning | Color |
|---|---|---|
| `┊` (U+250A) | Left rail (1 cell wide, all rows) | amber when section has work · ghost when empty · faint at the log |
| `▶` / `▼` | Collapsed / expanded commit or directory | faint |
| `*` | Tagged row — replaces leading indicator column | tag (mauve) |
| `·` | File bullet inside an expanded commit | faint |
| `↑N ↓M` | Ahead / behind upstream | iris / iris.dim, count in faint |

### Typography

Single monospace font (whatever the terminal provides). One size. Weight via `lipgloss.Bold()` only — no italic, no underline outside diff context.

---

## The Refined Overview (Slide 05) — Detailed Spec

This is the main screen. Layout, top to bottom:

```
┊ main  origin/main  ↑2 ↓0                                       v0.1.0+a3f9
┊ Untracked · 2                                       ← cursor row, fills text width only
┊   ? .gitignore
┊   ? go.mod
┊ Unstaged  · 1  +12/-3
┊   M main.go                                         +12/-3
┊ Staged    · 1  +24/-8
┊   M README.md                                       +24/-8
┊ Recent commits
┊   ▶ cfdc578   3m Refactor data pipeline  (main, origin/main)
┊   ▶ 4bc871d  30m Update dependencies
┊   ▶ a978566   2h Fix authentication bug
s=stage t=tag c=commit d=diff                              ?=help q=quit
```

### Six surgical changes from baseline

1. **Left rail.** A 1-cell `┊` column on every row. Color carries section state:
   - `amber` on the section header row when it has files
   - `amber.dim` on each file row beneath an active section
   - `ghost` (`#34322b`) on blank rows and on empty sections
   - `faint` on the "Recent commits" group
   The rail is the *attendance light* — you can read the screen from across the room.

2. **Header rhythm.** Section headers become `Untracked · 2` instead of `Untracked (2)` — `·` separator with the count in `term.fg.soft`. Lighter, mutt-faithful.

3. **Diff-stat on headers.** When a section has changes, append ` +12/-3` after the count, in `add` and `del` colors. Format: ` +N/-N` with a single space before.

4. **Right-aligned per-file stats.** On each file row, after the filename, pad to column ~78 then write ` +N/-N`. Use the empty space we already have.

5. **"Recent commits" demoted.** Header is `term.fg.soft` bold — no amber. It's reference, not action.

6. **Commit row layout.** `▶ {sha7} {age:>4} {title}  ({refs})`
   - `▶` in faint (collapsed), `▼` when expanded
   - `sha7` in `iris.dim`
   - `age` in faint, right-padded to 4 cells (`3m`, `30m`, `2h`, `1d`, `4d`)
   - `title` in default fg
   - `(refs)` only on commits that carry one — `main` in iris-bold, `origin/main` in iris.dim, parens in faint

### Cursor behavior

The cursor is a **pin**, not a bar. It paints `cursor.bg` only on the content width of the row (text length + a couple cells of padding), not the full terminal width. This was the single biggest readability win in user testing of the design — it preserves the rail and makes multi-tag selection visually correct.

### HEAD ref

In the baseline, HEAD is rendered as `(HEAD → main)` in amber-bold inside the commit log. **Remove the inline `HEAD →`.** The top status line already tells you the branch. In the commit log, just show the branch ref normally: `(main, origin/main)`.

### Section order

Unchanged: Untracked → Unstaged → Staged → (Stash) → Recent commits.

---

## Stash Section (Slide 09)

Add stash as a **section in the overview**, between Staged and Recent commits — not a separate screen. This preserves the single-screen mutt grammar.

```
┊ Stash     · 2
┊   * stash@{0}  WIP on main: experimenting with refactor
┊     stash@{1}  2h ago — pre-rebase snapshot
```

- Header in `tag` (mauve) when populated, ghost when empty (or omit the section entirely if no stashes).
- Rows show `stash@{N}` in iris.dim, then the stash message in default fg.
- The leading `*` glyph indicates "tag-marked for action" — the existing `t` key tags any row, including stash rows.

### Two new keys

- `S` (capital) — push working tree to a new stash. Prompts for message.
- `;p` — pop the tagged stash (uses the existing `;` tag-prefix command grammar from `PLAN.md`).

No new screens. Stash apply/pop happens from the overview.

---

## Inline Diff — Unified (Slide 07)

Replaces the current diff render. Layout:

```
┊ main · diff main.go (unstaged)                         +12/-3 · hunk 1/2
┊
┊  @@ -42,7 +42,9 @@ func parseFlags() {
┊      if *verbose {
┊  -     log.SetOutput(os.Stderr)
┊  +     log.SetOutput(io.MultiWriter(os.Stderr, logFile))
┊  +     log.SetFlags(log.LstdFlags | log.Lshortfile)
┊      }
┊
─────────────────────────────────────────────────────────────────
j/k=line ]/[=hunk s=stage·hunk v=split e=editor L=less        q=back
```

### Rules

- Keep the same rail behavior (faint on diff screens — work is in the diff itself).
- Hunk header (`@@…@@`) in `iris.dim`, function context after the second `@@` in `term.fg.soft`.
- `+` lines: `add` color foreground, no background fill on the row.
- `-` lines: `del` color foreground, no background fill.
- Cursor row gets `cursor.bg` — pin, same rule as overview.
- Faint background tint (10% alpha of `add`/`del`) on changed rows is **optional** and should be configurable; off by default to preserve the dense look.
- `+12/-3 · hunk 1/2` in the title bar — running total on the left, current hunk position on the right.
- Stage-by-hunk on `s`. Stage-by-line on `S` (capital). The current behavior of `s` from the overview is unchanged.

### Fallback to `less`

If the diff exceeds N lines (configurable, default 1000), pipe to `less` instead of rendering inline. `L` always forces less.

---

## Inline Diff — Split (Slide 08, optional)

Toggle from unified with `v`. Two-pane, left = before, right = after. Same color and rail rules. Width: 50/50, with a 1-cell `│` rule between.

Only worth implementing once the unified view is solid. Gate behind a config flag if shipping incrementally.

---

## Implementation Notes

### Lipgloss styles

Group styles by purpose, not by color. A non-exhaustive sketch:

```go
var (
    railActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a"))
    railUnder    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a6e44"))
    railGhost    = lipgloss.NewStyle().Foreground(lipgloss.Color("#34322b"))
    railFaint    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b5849"))

    sectHeader   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a")).Bold(true)
    sectHeaderQ  = lipgloss.NewStyle().Foreground(lipgloss.Color("#a39e8d")).Bold(true) // "Recent commits"

    branchIris   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7fb8c4")).Bold(true)
    shaIris      = lipgloss.NewStyle().Foreground(lipgloss.Color("#4d7e8a"))

    addStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#94b87a"))
    delStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#c87766"))
    modStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a"))
    tagStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#b491c8"))

    fgDefault    = lipgloss.NewStyle().Foreground(lipgloss.Color("#d8d4c7"))
    fgSoft       = lipgloss.NewStyle().Foreground(lipgloss.Color("#a39e8d"))
    fgFaint      = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b5849"))

    cursorRow    = lipgloss.NewStyle().Background(lipgloss.Color("#2a2a35"))
)
```

### Cursor as pin (not bar)

Today `cursorRow` likely fills the terminal width via `lipgloss.NewStyle().Width(termWidth)`. Don't. Compose the row first, then apply `Background` only — lipgloss leaves the background scoped to the rendered string. If you must pad, pad with `lipgloss.PlaceHorizontal` to a column ~2 cells past the longest content cell of that row, not to terminal width.

### Diff-stat acquisition

`+N/-N` per file is `git diff --numstat` (and `--numstat --cached` for staged). Cache per render tick; this is the cheapest git plumbing call.

### Compatibility

Truecolor is assumed. On 256-color terminals, lipgloss will degrade automatically — verify the palette still reads. Ghost (`#34322b`) collapses dangerously close to `term.bg` on 256-color; consider raising it one step in the degraded path.

---

## What's Out of Scope

- No rebase, no merge UI, no log graph drawing.
- No mouse support.
- No new screens beyond inline diff. Stash lives in the overview.
- No theming system. One palette, hardcoded.

---

## Files in This Bundle

- `gt - TUI design.html` — the deck (open in a browser, use ←/→).
- `styles.css` — the terminal cell renderer + slide chrome. Reference for cell math only; don't translate this to Go.
- `README.md` — this file.

---

## Open Questions for the Implementer

1. Is `git diff --numstat` already cached anywhere in `internal/git/`? If yes, reuse.
2. The `;` tag-prefix grammar in `PLAN.md` — has it landed yet, or is it still a plan? Stash actions assume it.
3. Is there an existing config layer for things like the diff-fallback threshold and the optional changed-row tint?
