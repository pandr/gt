package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/petera/gt/internal/git"
)

func (m Model) View() string {
	if m.mode == modeHelp {
		return m.helpView()
	}
	if m.mode == modeDiff {
		return m.diffView()
	}

	var b strings.Builder

	b.WriteString(m.branchHeader())
	b.WriteString("\n")

	visibleStart, visibleEnd := m.visibleRange()
	for i := visibleStart; i < visibleEnd; i++ {
		r := m.rows[i]
		line := m.renderRow(r, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	rendered := visibleEnd - visibleStart
	headerLines := 1
	statusLines := 1
	if m.mode == modeCommit || m.mode == modeShell {
		statusLines = 2
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	remaining := h - headerLines - rendered - statusLines
	for ; remaining > 0; remaining-- {
		b.WriteString("\n")
	}

	b.WriteString(m.statusBar())

	return b.String()
}

func (m Model) branchHeader() string {
	if m.status == nil {
		return railFaint.Render("┊") + " " + branchIris.Render("Loading…")
	}

	branch := m.status.Branch
	if branch == "(detached)" || branch == "" {
		branch = "(detached HEAD)"
	}

	s := branchIris.Render(branch)

	if m.status.Upstream != "" {
		s += "  " + remoteIris.Render(m.status.Upstream)
		if m.status.Ahead > 0 {
			s += "  " + branchIris.Render("↑") + fgFaint.Render(fmt.Sprintf("%d", m.status.Ahead))
		}
		if m.status.Behind > 0 {
			s += " " + remoteIris.Render("↓") + fgFaint.Render(fmt.Sprintf("%d", m.status.Behind))
		}
	}

	rail := railFaint.Render("┊")
	line := rail + " " + s

	if m.version != "" && m.width > 0 {
		ver := fgFaint.Render(m.version)
		used := visibleLen(line)
		pad := m.width - used - visibleLen(m.version)
		if pad > 1 {
			line += strings.Repeat(" ", pad) + ver
		}
	}

	return line
}

func (m Model) visibleRange() (int, int) {
	if len(m.rows) == 0 {
		return 0, 0
	}
	headerLines := 1
	statusLines := 1
	if m.mode == modeCommit || m.mode == modeShell {
		statusLines = 2
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	available := h - headerLines - statusLines
	if available < 1 {
		available = 1
	}

	start := m.cursor - available/2
	if start < 0 {
		start = 0
	}
	end := start + available
	if end > len(m.rows) {
		end = len(m.rows)
		start = end - available
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func (m Model) renderRow(r row, isCursor bool) string {
	content := m.rowContent(r)

	tagged := tagKey(r) != "" && m.tags[tagKey(r)]
	if tagged {
		// Replace the indicator/arrow character with the * tagged marker.
		// For file rows: "indent  X path" — indicator is at depth*2+2.
		// For commit rows: "  ▶ sha..." — arrow is at position 2 (depth=0).
		// The formula depth*2+2 covers both cases.
		indicatorPos := r.depth*2 + 2
		before := visiblePrefix(content, indicatorPos)
		after := visibleSuffix(content, indicatorPos+1)
		content = before + tagStyle.Render("*") + after
	}

	rail := m.railFor(r)
	line := rail + " " + content

	if isCursor {
		return applyCursorBg(line, m.width)
	}
	return line
}

// applyCursorBg pads line to width and applies the cursor background color.
// It re-injects the background code after every SGR reset sequence, because
// inner color codes emit a reset that clears any background set by a wrapper.
const cursorBgCode = "\x1b[48;2;42;42;53m"

func applyCursorBg(line string, width int) string {
	if width > 0 {
		if w := visibleLen(line); w < width {
			line += strings.Repeat(" ", width-w)
		}
	}
	line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+cursorBgCode)
	line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+cursorBgCode)
	return cursorBgCode + line + "\x1b[0m"
}

// railFor returns the styled ┊ glyph appropriate for a given row.
func (m Model) railFor(r row) string {
	const glyph = "┊"
	switch r.kind {
	case rowSeparator:
		return railGhost.Render(glyph)
	case rowSectionHeader:
		switch r.section {
		case git.SectionLog, git.SectionWorkingTree:
			return railFaint.Render(glyph)
		default:
			if m.sectionHasWork(r.section) {
				return railActive.Render(glyph)
			}
			return railGhost.Render(glyph)
		}
	case rowFile, rowDir:
		switch r.section {
		case git.SectionLog, git.SectionWorkingTree:
			return railFaint.Render(glyph)
		default:
			return railUnder.Render(glyph)
		}
	case rowCommit, rowCommitBody, rowCommitFile:
		return railFaint.Render(glyph)
	}
	return railGhost.Render(glyph)
}

func (m Model) sectionHasWork(s git.Section) bool {
	if m.status == nil {
		return false
	}
	switch s {
	case git.SectionUntracked:
		return len(m.status.Untracked) > 0
	case git.SectionUnstaged:
		return len(m.status.Unstaged) > 0
	case git.SectionStaged:
		return len(m.status.Staged) > 0
	}
	return false
}

func (m Model) rowContent(r row) string {
	indent := strings.Repeat("  ", r.depth)
	switch r.kind {
	case rowSectionHeader:
		return m.sectionHeader(r.section)
	case rowFile:
		if r.file == nil {
			return ""
		}
		return indent + m.fileRow(r.file, r.section)
	case rowCommit:
		if r.commit == nil {
			return ""
		}
		sha := r.commit.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		arrow := fgFaint.Render("▶")
		if _, ok := m.openCommits[r.commit.SHA]; ok {
			arrow = fgFaint.Render("▼")
		}
		age := fgFaint.Render(formatCommitAge(r.commit.Time, time.Now()))
		refs := ""
		if len(r.commit.Refs) > 0 {
			refs = "  " + renderRefs(r.commit.Refs)
		}
		return "  " + arrow + " " + shaIris.Render(sha) + " " + age + " " + r.commit.Title + refs
	case rowCommitBody:
		return "                  " + fgFaint.Render(r.dirPath)
	case rowCommitFile:
		if r.commit == nil {
			return ""
		}
		return indent + "    " + fgFaint.Render("·") + " " + r.dirPath + formatStats(r.statAdded, r.statDeleted)
	case rowDir:
		return m.dirRow(r)
	case rowSeparator:
		return ""
	}
	return ""
}

func (m Model) dirRow(r row) string {
	if r.section == git.SectionWorkingTree && r.dirPath == "./" {
		arrow := "▶"
		if m.wtOpen {
			arrow = "▼"
		}
		extra := ""
		if !m.wtOpen && len(m.wtFiles) > 0 {
			extra = fgFaint.Render(fmt.Sprintf("  (%d files)", len(m.wtFiles)))
		}
		return "  " + fgFaint.Render(arrow) + " " + fgFaint.Render("./") + extra
	}
	indent := strings.Repeat("  ", r.depth)
	arrow := "▶"
	if m.openDirs[r.dirPath] {
		arrow = "▼"
	}
	name := r.dirPath
	extra := ""
	if !m.openDirs[r.dirPath] {
		if children, ok := m.dirContents[r.dirPath]; ok {
			extra = fgFaint.Render(fmt.Sprintf("  (%d files)", len(children)))
		}
	}
	var nameStyle lipgloss.Style
	switch r.section {
	case git.SectionWorkingTree:
		if m.dirHasChanges(r.dirPath) {
			nameStyle = modStyle
		} else {
			nameStyle = fgFaint
		}
	default:
		nameStyle = addStyle
	}
	return indent + "  " + fgFaint.Render(arrow) + " " + nameStyle.Render(name) + extra
}

func (m Model) sectionHeader(s git.Section) string {
	switch s {
	case git.SectionLog:
		return sectHeaderQ.Render("Recent commits")
	case git.SectionWorkingTree:
		return sectHeader.Render("Working tree")
	}

	var name string
	var count int
	var totalAdded, totalDeleted int

	switch s {
	case git.SectionUntracked:
		name = "Untracked"
		if m.status != nil {
			for _, f := range m.status.Untracked {
				if f.IsDir {
					if contents, ok := m.dirContents[f.Path]; ok {
						count += len(contents)
					} else {
						count++
					}
				} else {
					count++
				}
			}
		}
	case git.SectionUnstaged:
		name = "Unstaged"
		if m.status != nil {
			count = len(m.status.Unstaged)
			for _, f := range m.status.Unstaged {
				totalAdded += f.Added
				totalDeleted += f.Deleted
			}
		}
	case git.SectionStaged:
		name = "Staged"
		if m.status != nil {
			count = len(m.status.Staged)
			for _, f := range m.status.Staged {
				totalAdded += f.Added
				totalDeleted += f.Deleted
			}
		}
	}

	header := sectHeader.Render(name) + " " + fgFaint.Render("·") + " " + fgSoft.Render(fmt.Sprintf("%d", count))
	if totalAdded > 0 || totalDeleted > 0 {
		header += " " + formatStats(totalAdded, totalDeleted)
	}
	return header
}

func (m Model) fileRow(f *git.FileEntry, section git.Section) string {
	xy := f.XY
	var indicator string
	var indicatorStyle lipgloss.Style

	switch section {
	case git.SectionUntracked:
		indicator = "?"
		indicatorStyle = addStyle
	case git.SectionUnstaged:
		y := rune(xy[1])
		indicator, indicatorStyle = xyIndicator(y)
	case git.SectionStaged:
		x := rune(xy[0])
		indicator, indicatorStyle = xyIndicator(x)
	case git.SectionWorkingTree:
		if sf := m.statusForPath(f.Path); sf != nil {
			y := rune(sf.XY[1])
			x := rune(sf.XY[0])
			if y != '.' {
				ind, sty := xyIndicator(y)
				return "  " + sty.Render(ind) + " " + f.Path
			} else if x != '.' {
				ind, sty := xyIndicator(x)
				return "  " + sty.Render(ind) + " " + fgFaint.Render(f.Path)
			}
		}
		return "   " + fgFaint.Render(f.Path)
	}

	prefix := "  " + indicatorStyle.Render(indicator) + " "
	stats := formatStats(f.Added, f.Deleted)
	if stats == "" {
		return prefix + f.Path
	}
	return prefix + f.Path + "  " + stats
}

func xyIndicator(ch rune) (string, lipgloss.Style) {
	switch ch {
	case 'A':
		return "A", addStyle
	case 'M', 'T', 'U':
		return "M", modStyle
	case 'D':
		return "D", delStyle
	case 'R':
		return "R", modStyle
	case 'C':
		return "C", addStyle
	}
	return string(ch), fgFaint
}

func formatStats(added, deleted int) string {
	if added == 0 && deleted == 0 {
		return ""
	}
	var b strings.Builder
	if added > 0 {
		b.WriteString(addStyle.Render(fmt.Sprintf("+%d", added)))
	}
	if added > 0 && deleted > 0 {
		b.WriteString(fgFaint.Render("/"))
	}
	if deleted > 0 {
		b.WriteString(delStyle.Render(fmt.Sprintf("-%d", deleted)))
	}
	return b.String()
}

func renderRefs(refs []string) string {
	var parts []string
	for _, r := range refs {
		var rendered string
		switch {
		case strings.HasPrefix(r, "HEAD -> "):
			// Design: drop the "HEAD →" annotation; just show the branch normally.
			branch := strings.TrimPrefix(r, "HEAD -> ")
			rendered = branchIris.Render(branch)
		case strings.HasPrefix(r, "tag: "):
			rendered = refTagIris.Render(r)
		case strings.Contains(r, "/"):
			rendered = remoteIris.Render(r)
		default:
			rendered = branchIris.Render(r)
		}
		parts = append(parts, rendered)
	}
	return fgFaint.Render("(") + strings.Join(parts, fgFaint.Render(", ")) + fgFaint.Render(")")
}

func (m Model) statusBar() string {
	if m.mode == modeHelp {
		return ""
	}
	if m.mode == modeCommit {
		prefix := ""
		if m.amendMode {
			prefix = modStyle.Render("amend  ")
		}
		return "\n" + prefix + m.commitInput.View()
	}
	if m.mode == modeShell {
		return "\n" + sectHeader.Render("!") + " " + m.commitInput.View()
	}
	if m.mode == modeTagPrefix {
		return styleStatusBar.Render(";_ — waiting for command (s=stage  u=unstage  d=diff)")
	}
	if m.mode == modeConfirm {
		return styleToast.Render(m.confirmPrompt)
	}

	var parts []string
	if m.toast != "" {
		parts = append(parts, styleToast.Render(m.toast))
	} else {
		n := len(m.tags)
		if n > 0 {
			parts = append(parts, tagStyle.Render(fmt.Sprintf("[%d tagged]", n)))
		}
		parts = append(parts, styleStatusBar.Render(m.contextHints()))
	}
	return strings.Join(parts, "   ")
}

func (m Model) contextHints() string {
	var hints []string

	if m.cursor >= 0 && m.cursor < len(m.rows) {
		r := m.rows[m.cursor]
		switch r.kind {
		case rowCommit:
			if r.commit != nil {
				isFirst := len(m.log) > 0 && r.commit.SHA == m.log[0].SHA
				canAmend := isFirst && (m.status == nil || m.status.Upstream == "" || m.status.Ahead > 0)
				if _, ok := m.openCommits[r.commit.SHA]; ok {
					hints = []string{"d=diff", "h=collapse", "t=tag", "?=help", "q=quit"}
				} else {
					hints = []string{"d=diff", "l=expand", "t=tag", "?=help", "q=quit"}
				}
				if canAmend {
					hints = append([]string{"A=amend"}, hints...)
				}
			}
		case rowCommitFile:
			hints = []string{"d=diff", "h=collapse", "?=help", "q=quit"}
		case rowSectionHeader:
			switch r.section {
			case git.SectionUntracked:
				hints = []string{"s=stage", "t=tag", "c=commit", "?=help", "q=quit"}
			case git.SectionUnstaged:
				hints = []string{"d=diff", "s=stage", "r=restore", "t=tag", "c=commit", "?=help", "q=quit"}
			case git.SectionStaged:
				hints = []string{"d=diff", "u=unstage", "c=commit", "?=help", "q=quit"}
			case git.SectionLog:
				hints = []string{"R=refresh", "?=help", "q=quit"}
			case git.SectionWorkingTree:
				hints = []string{"R=refresh", "?=help", "q=quit"}
			}
		case rowFile:
			switch r.section {
			case git.SectionUntracked:
				hints = []string{"s=stage", "x=untrack", "X=delete", "t=tag", "c=commit", "?=help", "q=quit"}
			case git.SectionUnstaged:
				hints = []string{"d=diff", "s=stage", "r=restore", "x=untrack", "t=tag", "c=commit", "?=help", "q=quit"}
			case git.SectionStaged:
				hints = []string{"d=diff", "u=unstage", "c=commit", "t=tag", "?=help", "q=quit"}
			case git.SectionWorkingTree:
				hints = []string{"d=diff", "t=tag", "R=refresh", "?=help", "q=quit"}
			}
		case rowDir:
			switch r.section {
			case git.SectionUntracked:
				hints = []string{"s=stage", "l/h=expand", "?=help", "q=quit"}
			case git.SectionWorkingTree:
				hints = []string{"d=diff", "l/h=expand", "?=help", "q=quit"}
			}
		}
	}

	if len(hints) == 0 {
		hints = []string{"c=commit", "d=diff", "s=stage", "u=unstage", "t=tag", "?=help", "q=quit"}
	}
	return strings.Join(hints, "  ")
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString("Key bindings\n\n")
	for _, kb := range keyBindings {
		b.WriteString(fmt.Sprintf("  %-22s %s\n", kb.Key, kb.Desc))
	}
	b.WriteString("\nPress ? or q to close")
	return styleHelp.Render(b.String())
}

// formatCommitAge renders a commit timestamp as a fixed 4-char field:
//
//	"  3m"  under 1 hour
//	" 30h"  1-47 hours
//	" 13d"  2-13 days
//	"DD-MM" 14d up to 1 year
//	"YYYY"  older than 1 year
func formatCommitAge(t, now time.Time) string {
	if t.IsZero() {
		return "    "
	}
	d := now.Sub(t)
	switch {
	case d < time.Hour:
		m := int(d / time.Minute)
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("%3dm", m)
	case d < 48*time.Hour:
		return fmt.Sprintf("%3dh", int(d/time.Hour))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%3dd", int(d/(24*time.Hour)))
	case d < 365*24*time.Hour:
		return t.Format("02-01")
	default:
		return fmt.Sprintf("%4d", t.Year())
	}
}

// visibleLen returns the number of visible (non-ANSI) characters in s.
func visibleLen(s string) int {
	inEscape := false
	n := 0
	for _, r := range s {
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		n++
	}
	return n
}

// stripANSI returns s with all ANSI escape sequences removed.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// visiblePrefix returns the bytes of s that correspond to the first n visible characters,
// preserving any ANSI sequences encountered along the way.
func visiblePrefix(s string, n int) string {
	var b strings.Builder
	inEscape := false
	count := 0
	for _, r := range s {
		if count >= n && !inEscape {
			break
		}
		b.WriteRune(r)
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		count++
	}
	return b.String()
}

// visibleSuffix returns the bytes of s starting after the first n visible characters.
func visibleSuffix(s string, n int) string {
	inEscape := false
	count := 0
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		if inEscape {
			if runes[i] == 'm' {
				inEscape = false
			}
			i++
			continue
		}
		if runes[i] == '\x1b' {
			inEscape = true
			i++
			continue
		}
		count++
		i++
		if count >= n {
			break
		}
	}
	return string(runes[i:])
}
