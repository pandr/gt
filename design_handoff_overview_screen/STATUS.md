# Implementation Status

## Done
- **Slide 05 — Overview refined**: rail, header rhythm (`·` separator), diff-stats on headers, inline file stats, commit row layout (sha/age colors), HEAD ref dropped from log, cursor full-width highlight, truecolor theme (`internal/ui/theme.go`)
- **Slide 07 — Inline diff (unified)**: rendering, j/k/]/[ navigation, space/ctrl+d/u paging, `/` search (n/N), commit history diffs (git show, multi-file), section-header diffs, pager fallback (L key + auto at 1000 lines)

## Not done
- **Stash section (Slide 09)** — stash between Staged and Recent commits, `S` to push, `;p` to pop *(spec says: yes)*
- **Stage-by-hunk** — `s` from within the diff view *(deferred from inline diff work)*
- **Split diff (Slide 08)** — two-pane toggle with `v` *(spec says: optional)*
- **Stage-by-line** — `S` from within the diff view *(spec says: optional, complex)*
