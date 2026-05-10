package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petera/gt/internal/git"
)

// fileViewMsg carries the result of an async file load.
type fileViewMsg struct {
	ctx     string      // branch name or sha7
	path    string      // repo-relative path
	section git.Section // for the "(unstaged)" label
	lines   []string
	err     error
}

// fetchFileLines loads an on-disk file through bat.
func fetchFileLines(absPath, ctx, displayPath string, section git.Section) tea.Cmd {
	return func() tea.Msg {
		lines, err := batCaptureFile(absPath)
		return fileViewMsg{ctx: ctx, path: displayPath, section: section, lines: lines, err: err}
	}
}

// fetchFileAtRevLines loads a file at a git revision through bat.
func fetchFileAtRevLines(repoRoot, sha, path, ctx string) tea.Cmd {
	return func() tea.Msg {
		gitCmd := exec.Command("git", "show", sha+":"+path)
		gitCmd.Dir = repoRoot
		lines, err := batCaptureCmd(gitCmd, filepath.Base(path))
		return fileViewMsg{ctx: ctx, path: path, section: git.SectionCommit, lines: lines, err: err}
	}
}

// batCaptureFile runs bat on an on-disk file and returns remapped lines.
// Falls back to plain os.ReadFile if bat is not available.
func batCaptureFile(absPath string) ([]string, error) {
	if batBin == "" {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		return splitLines(string(data)), nil
	}
	cmd := exec.Command(batBin,
		"--color=always", "--paging=never", "--style=plain", "--theme=ansi", "--tabs=4",
		absPath)
	return batRun(cmd)
}

// batCaptureCmd pipes gitCmd's stdout through bat and returns remapped lines.
// Falls back to capturing gitCmd's raw output if bat is not available.
func batCaptureCmd(gitCmd *exec.Cmd, filename string) ([]string, error) {
	if batBin == "" {
		var buf bytes.Buffer
		gitCmd.Stdout = &buf
		if err := gitCmd.Run(); err != nil {
			return nil, err
		}
		return splitLines(buf.String()), nil
	}

	batCmd := exec.Command(batBin,
		"--color=always", "--paging=never", "--style=plain", "--theme=ansi", "--tabs=4",
		"--file-name="+filename, "-")

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	gitCmd.Stdout = pw
	batCmd.Stdin = pr

	var out bytes.Buffer
	batCmd.Stdout = &out

	if err := gitCmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return nil, err
	}
	if err := batCmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		gitCmd.Wait()
		return nil, err
	}
	gitCmd.Wait()
	pw.Close()
	if err := batCmd.Wait(); err != nil {
		pr.Close()
		return nil, err
	}
	pr.Close()

	return processLines(out.String()), nil
}

func batRun(cmd *exec.Cmd) ([]string, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return processLines(buf.String()), nil
}

func processLines(s string) []string {
	lines := splitLines(s)
	for i, l := range lines {
		lines[i] = strings.ReplaceAll(remapANSI(l), "\t", "    ")
	}
	return lines
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// ansiPalette maps ANSI color indices 0–15 to "R;G;B" strings that match our theme.
// Indices 0–7 are the normal colors; 8–15 are the bright variants.
var ansiPalette = [16]string{
	"20;20;26",    // 0  black      → near-background
	"200;119;102", // 1  red        → terracotta  (#c87766)
	"148;184;122", // 2  green      → moss        (#94b87a)
	"214;169;106", // 3  yellow     → amber       (#d6a96a)
	"127;184;196", // 4  blue       → iris        (#7fb8c4)
	"180;145;200", // 5  magenta    → mauve       (#b491c8)
	"77;126;138",  // 6  cyan       → iris-dim    (#4d7e8a)
	"163;158;141", // 7  white      → fg-soft     (#a39e8d)
	"91;88;73",    // 8  br.black   → faint       (#5b5849) — comments land here
	"224;144;128", // 9  br.red     → terracotta-light
	"170;212;148", // 10 br.green   → moss-light
	"232;192;128", // 11 br.yellow  → amber-light
	"160;216;232", // 12 br.blue    → iris-light
	"203;168;224", // 13 br.magenta → mauve-light
	"127;184;196", // 14 br.cyan    → iris        (#7fb8c4)
	"200;196;180", // 15 br.white   → near-white
}

// batBin is the resolved path to bat, empty if not installed.
var batBin, _ = exec.LookPath("bat")

var ansiSeqRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// remapANSI replaces 4-bit ANSI color codes in s with true-color equivalents
// drawn from ansiPalette, preserving all non-color attributes (bold, dim, etc.).
func remapANSI(s string) string {
	return ansiSeqRe.ReplaceAllStringFunc(s, func(seq string) string {
		inner := seq[2 : len(seq)-1] // strip ESC[ and m
		if inner == "" {
			return seq
		}
		parts := strings.Split(inner, ";")
		var out []string
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				out = append(out, p)
				continue
			}
			var idx int
			var ground string
			switch {
			case n >= 30 && n <= 37:
				idx, ground = n-30, "38"
			case n >= 40 && n <= 47:
				idx, ground = n-40, "48"
			case n >= 90 && n <= 97:
				idx, ground = n-90+8, "38"
			case n >= 100 && n <= 107:
				idx, ground = n-100+8, "48"
			default:
				out = append(out, p)
				continue
			}
			out = append(out, ground, "2")
			out = append(out, strings.Split(ansiPalette[idx], ";")...)
		}
		return "\x1b[" + strings.Join(out, ";") + "m"
	})
}

// ── rendering ────────────────────────────────────────────────────────────────

func (m Model) fileView() string {
	var b strings.Builder

	b.WriteString(m.fileTitleBar())
	b.WriteString("\n")
	b.WriteString(railFaint.Render("┊"))
	b.WriteString("\n")

	h := m.height
	if h < 10 {
		h = 24
	}
	contentLines := h - 4 // title + blank rail + separator + hints

	start, end := m.fileVisibleRange(contentLines)
	total := len(m.fileLines)

	numWidth := len(strconv.Itoa(total))
	if numWidth < 3 {
		numWidth = 3
	}

	for i := start; i < end; i++ {
		num := fgFaint.Render(fmt.Sprintf("%*d ", numWidth, i+1))
		content := m.fileLines[i]
		if m.fileSearch != "" {
			content = injectHighlights(content, m.fileSearch)
		}
		line := num + content
		if i == m.fileCursor {
			line = applyCursorBg(line, m.width)
		} else if m.width > 0 {
			if w := visibleLen(num) + visibleLen(m.fileLines[i]); w < m.width {
				line += strings.Repeat(" ", m.width-w)
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	rendered := end - start
	for range contentLines - rendered {
		if m.width > 0 {
			b.WriteString(strings.Repeat(" ", m.width))
		}
		b.WriteString("\n")
	}

	sepWidth := m.width
	if sepWidth <= 0 {
		sepWidth = 64
	}
	b.WriteString(fgFaint.Render(strings.Repeat("─", sepWidth)))
	b.WriteString("\n")

	if m.fileSearching {
		b.WriteString(styleStatusBar.Render("/") + " " + m.commitInput.View())
	} else {
		hints := "j/k=line  space/ctrl+d/u=page  g/G=top/bottom  /=search  e=editor  q=back"
		if m.fileSearch != "" {
			hints = "n/N=match  " + hints
		}
		b.WriteString(styleStatusBar.Render(hints))
	}

	return b.String()
}

func (m Model) fileTitleBar() string {
	// Mirror diffTitleBar: ┊ ctx · view path (section)    line/total (pct%)
	var sectionLabel string
	switch m.fileSection {
	case git.SectionUnstaged:
		sectionLabel = "unstaged"
	case git.SectionStaged:
		sectionLabel = "staged"
	case git.SectionUntracked:
		sectionLabel = "untracked"
	}

	ctxStyle := branchIris
	if m.fileSection == git.SectionCommit {
		ctxStyle = shaIris
	}

	left := railFaint.Render("┊") + " " +
		ctxStyle.Render(m.fileCtx) + " " +
		fgFaint.Render("·") + " " +
		fgSoft.Render("view") + " " +
		m.filePath
	if sectionLabel != "" {
		left += " " + fgFaint.Render("("+sectionLabel+")")
	}

	total := len(m.fileLines)
	pos := ""
	if total > 0 {
		pct := (m.fileCursor + 1) * 100 / total
		pos = fgFaint.Render(fmt.Sprintf("%d/%d (%d%%)", m.fileCursor+1, total, pct))
	}

	w := m.width
	if w <= 0 {
		w = 64
	}
	pad := w - visibleLen(left) - visibleLen(pos)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + pos
}

func (m Model) fileVisibleRange(available int) (int, int) {
	n := len(m.fileLines)
	if n == 0 || available <= 0 {
		return 0, 0
	}
	start := m.fileCursor - available/2
	if start < 0 {
		start = 0
	}
	end := start + available
	if end > n {
		end = n
		start = end - available
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// ── key handling ─────────────────────────────────────────────────────────────

// handleFileKey handles keypresses in modeFile.
//
// KEEP IN SYNC: navigation and search here duplicate handleDiffKey in update.go.
// If you change one (add keys, fix page size, etc.) update the other, or unify them.
func (m Model) handleFileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.fileSearching {
		return m.handleFileSearchKey(msg)
	}
	n := len(m.fileLines)
	switch msg.String() {
	case "q", "esc":
		m.mode = modeNormal
		m.fileLines = nil
		m.fileCtx = ""
		m.filePath = ""
		m.fileCursor = 0
		m.fileSearch = ""
		m.fileMatches = nil
		m.fileMatchIdx = -1
	case "j", "down":
		if m.fileCursor < n-1 {
			m.fileCursor++
		}
	case "k", "up":
		if m.fileCursor > 0 {
			m.fileCursor--
		}
	case " ", "ctrl+d":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.fileCursor += pageSize
		if m.fileCursor >= n {
			m.fileCursor = n - 1
		}
	case "ctrl+u":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.fileCursor -= pageSize
		if m.fileCursor < 0 {
			m.fileCursor = 0
		}
	case "g":
		m.fileCursor = 0
	case "G":
		if n > 0 {
			m.fileCursor = n - 1
		}
	case "e":
		if m.filePath != "" {
			return m, execEditFile(filepath.Join(m.repoRoot, m.filePath))
		}
	case "/":
		m.fileSearching = true
		m.commitInput.Reset()
		m.commitInput.Placeholder = "search…"
		m.commitInput.Focus()
		return m, nil
	case "n":
		if len(m.fileMatches) > 0 {
			m.fileMatchIdx = (m.fileMatchIdx + 1) % len(m.fileMatches)
			m.fileCursor = m.fileMatches[m.fileMatchIdx]
		}
	case "N":
		if len(m.fileMatches) > 0 {
			m.fileMatchIdx = (m.fileMatchIdx - 1 + len(m.fileMatches)) % len(m.fileMatches)
			m.fileCursor = m.fileMatches[m.fileMatchIdx]
		}
	}
	return m, nil
}

func (m Model) handleFileSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		pattern := m.commitInput.Value()
		m.fileSearch = pattern
		m.fileSearching = false
		m.commitInput.Blur()
		m.commitInput.Reset()
		m.fileMatches = m.computeFileMatches(pattern)
		m.fileMatchIdx = -1
		if len(m.fileMatches) > 0 {
			m.fileMatchIdx = 0
			for i, idx := range m.fileMatches {
				if idx >= m.fileCursor {
					m.fileMatchIdx = i
					break
				}
			}
			m.fileCursor = m.fileMatches[m.fileMatchIdx]
		}
		return m, nil
	case "esc":
		m.fileSearch = ""
		m.fileSearching = false
		m.commitInput.Blur()
		m.commitInput.Reset()
		m.fileMatches = nil
		m.fileMatchIdx = -1
		return m, nil
	}
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m Model) computeFileMatches(pattern string) []int {
	if pattern == "" {
		return nil
	}
	lower := strings.ToLower(pattern)
	var matches []int
	for i, line := range m.fileLines {
		if strings.Contains(strings.ToLower(stripANSI(line)), lower) {
			matches = append(matches, i)
		}
	}
	return matches
}

// injectHighlights wraps every occurrence of pattern in line with search
// highlight codes. line may contain ANSI sequences; matching uses visible text.
func injectHighlights(line, pattern string) string {
	if pattern == "" {
		return line
	}
	raw := []rune(strings.ToLower(stripANSI(line)))
	pat := []rune(strings.ToLower(pattern))
	if len(raw) < len(pat) {
		return line
	}

	type rng struct{ start, end int }
	var ranges []rng
	for i := 0; i <= len(raw)-len(pat); {
		j := 0
		for j < len(pat) && raw[i+j] == pat[j] {
			j++
		}
		if j == len(pat) {
			ranges = append(ranges, rng{i, i + len(pat)})
			i += len(pat)
		} else {
			i++
		}
	}
	if len(ranges) == 0 {
		return line
	}

	// amber bg (#d6a96a) + near-black fg (#14141a) — matches searchHlStyle in theme.go
	const hlOpen  = "\x1b[48;2;214;169;106m\x1b[38;2;20;20;26m"
	const hlClose = "\x1b[0m"

	var out strings.Builder
	vis := 0
	ri := 0
	inHL := false
	esc := false

	for _, r := range line {
		if esc {
			out.WriteRune(r)
			if r == 'm' {
				esc = false
				if inHL {
					out.WriteString(hlOpen) // re-apply after any ANSI reset
				}
			}
			continue
		}
		if r == '\x1b' {
			esc = true
			out.WriteRune(r)
			continue
		}
		if ri < len(ranges) {
			if vis == ranges[ri].start && !inHL {
				out.WriteString(hlOpen)
				inHL = true
			}
			if vis == ranges[ri].end && inHL {
				out.WriteString(hlClose)
				inHL = false
				ri++
				if ri < len(ranges) && vis == ranges[ri].start {
					out.WriteString(hlOpen)
					inHL = true
				}
			}
		}
		out.WriteRune(r)
		vis++
	}
	if inHL {
		out.WriteString(hlClose)
	}
	return out.String()
}
