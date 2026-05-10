package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/petera/gt/internal/git"
)

// refreshMsg carries fresh status+log data after a background reload.
type refreshMsg struct {
	status *git.Status
	log    []git.LogEntry
	err    error
}

type dirContentsMsg struct {
	dirPath string
	files   []git.FileEntry
	err     error
}

type wtFilesMsg struct {
	files []git.FileEntry
	err   error
}

type commitFilesMsg struct {
	sha   string
	files []git.FileEntry
	body  []string
	err   error
}

type fileLogMsg struct {
	path    string
	entries []git.LogEntry
	err     error
}

type fileLogCommitFilesMsg struct {
	sha   string
	files []git.FileEntry
	body  []string
	err   error
}


func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("gt "+m.displayPath),
		refresh(m.repoRoot),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
			return m, nil
		}
		m.status = msg.status
		m.log = msg.log
		m.buildRows()
		m.pruneTags()
		if m.cursorTargetPath != "" {
			found := false
			for i, r := range m.rows {
				if r.file != nil && r.file.Path == m.cursorTargetPath {
					m.cursor = i
					found = true
					break
				}
			}
			m.cursorTargetPath = ""
			if !found {
				m.clampCursor()
				m.skipSeparators(+1)
			}
		} else {
			m.clampCursor()
			m.skipSeparators(+1)
		}
		// Re-fetch contents of any open untracked dirs so new files appear after R.
		var refetchCmds []tea.Cmd
		for _, f := range m.status.Untracked {
			if f.IsDir && m.openDirs[f.Path] {
				refetchCmds = append(refetchCmds, expandUntrackedDir(m.repoRoot, f.Path))
			}
		}
		if len(refetchCmds) > 0 {
			return m, tea.Batch(refetchCmds...)
		}
		return m, nil

	case dirContentsMsg:
		if msg.err == nil {
			m.dirContents[msg.dirPath] = msg.files
			m.buildRows()
		}
		return m, nil

	case wtFilesMsg:
		if msg.err == nil {
			m.wtFiles = msg.files
			m.buildRows()
		}
		return m, nil

	case fileViewMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
			return m, nil
		}
		m.fileCtx = msg.ctx
		m.filePath = msg.path
		m.fileSection = msg.section
		m.fileLines = msg.lines
		m.fileCursor = 0
		m.fileSearch = ""
		m.fileMatches = nil
		m.fileMatchIdx = -1
		m.fileSearching = false
		m.mode = modeFile
		return m, nil

	case commitFilesMsg:
		if msg.err == nil {
			m.openCommits[msg.sha] = msg.files
			m.commitBodies[msg.sha] = msg.body
			m.buildRows()
		}
		return m, nil

	case fileLogMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
			return m, nil
		}
		m.fileLogPath = msg.path
		m.fileLogEntries = msg.entries
		m.fileLogCursor = 0
		m.fileLogOpen = make(map[string][]git.FileEntry)
		m.fileLogBodies = make(map[string][]string)
		m.mode = modeFileLog
		return m, nil

	case fileLogCommitFilesMsg:
		if msg.err == nil {
			m.fileLogOpen[msg.sha] = msg.files
			m.fileLogBodies[msg.sha] = msg.body
			rows := m.fileLogRows()
			for i, r := range rows {
				if r.kind == rowCommitFile && r.commit != nil && r.commit.SHA == msg.sha && r.dirPath == m.fileLogPath {
					m.fileLogCursor = i
					break
				}
			}
		}
		return m, nil

	case execDoneMsg:
		// Ignore non-zero exit from diff commands: git diff --no-index exits 1 when
		// differences are found, and the pager may exit non-zero on user interrupt.
		return m, nil

	case shellDoneMsg:
		return m, refresh(m.repoRoot)

	case editorDoneMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
			return m, nil
		}
		var err error
		if msg.amend {
			err = git.CommitAmendFile(m.repoRoot, msg.filePath)
		} else {
			err = git.CommitFile(m.repoRoot, msg.filePath)
		}
		os.Remove(msg.filePath)
		m.mode = modeNormal
		m.amendMode = false
		m.commitInput.Blur()
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		return m, refresh(m.repoRoot)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear toast on any keypress
	if m.toast != "" {
		m.toast = ""
	}

	// Help overlay
	if m.mode == modeHelp {
		m.mode = modeNormal
		return m, nil
	}

	// Commit input mode
	if m.mode == modeCommit {
		return m.handleCommitKey(msg)
	}

	// Shell command mode
	if m.mode == modeShell {
		return m.handleShellKey(msg)
	}

	// Tag prefix mode (waiting for second key after ;)
	if m.mode == modeTagPrefix {
		return m.handleTagPrefixKey(msg)
	}

	// Confirm mode (y/n for destructive actions)
	if m.mode == modeConfirm {
		return m.handleConfirmKey(msg)
	}

	// Inline diff view
	if m.mode == modeDiff {
		return m.handleDiffKey(msg)
	}

	// Inline file view
	if m.mode == modeFile {
		return m.handleFileKey(msg)
	}

	// File history view
	if m.mode == modeFileLog {
		return m.handleFileLogKey(msg)
	}

	// Normal mode
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.mode = modeHelp
		return m, nil

	case "R":
		return m, refresh(m.repoRoot)

	case "j", "down":
		m.cursor++
		m.skipSeparators(+1)

	case "k", "up":
		m.cursor--
		m.skipSeparators(-1)

	case "g":
		m.cursor = 0
		m.skipSeparators(+1)

	case "G":
		m.cursor = len(m.rows) - 1
		m.skipSeparators(-1)

	case "ctrl+d":
		m.cursor += m.height / 2
		m.skipSeparators(+1)

	case "ctrl+u":
		m.cursor -= m.height / 2
		m.skipSeparators(-1)

	case "right", "l":
		return m.doExpand()

	case "left", "h":
		return m.doCollapse()

	case "enter":
		r := m.cursorRow()
		if r.kind == rowDir || r.kind == rowCommit {
			return m.doExpand()
		}
		if r.kind == rowFile && r.section == git.SectionWorkingTree && m.statusForPath(r.file.Path) == nil {
			return m.doViewFile()
		}
		return m.doDiff(r, nil)

	case "d", " ":
		return m.doDiff(m.cursorRow(), nil)

	case "v":
		return m.doViewFile()

	case "H":
		return m.doFileLog()

	case "e":
		return m.doEditFile()

	case "r":
		return m.doRestoreConfirm()

	case "x":
		return m.doRmCached()

	case "X":
		return m.doRmFileConfirm()

	case "s":
		return m.doStage()

	case "u":
		return m.doUnstage()

	case "t":
		r := m.cursorRow()
		k := tagKey(r)
		if k != "" {
			m.tags[k] = !m.tags[k]
			if !m.tags[k] {
				delete(m.tags, k)
			}
			m.cursor++
			m.skipSeparators(+1)
			m.clampCursor()
		}
		return m, nil

	case ";":
		if len(m.tags) > 0 {
			m.mode = modeTagPrefix
		}

	case "T":
		m.tags = make(map[string]bool)

	case "!":
		m.mode = modeShell
		m.commitInput.Placeholder = "Shell command (Enter=run  Esc=cancel)"
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textinput.Blink

	case "c":
		m.mode = modeCommit
		m.commitInput.Placeholder = "Commit message (Enter=commit  Ctrl-g=editor  Esc=cancel)"
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textinput.Blink

	case "A":
		if len(m.log) == 0 {
			m.toast = "no commits to amend"
			return m, nil
		}
		if m.status != nil && m.status.Upstream != "" && m.status.Ahead == 0 {
			m.toast = "cannot amend: commit already pushed"
			return m, nil
		}
		m.amendMode = true
		m.mode = modeCommit
		m.commitInput.SetValue(m.log[0].Title)
		m.commitInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

func (m Model) handleCommitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.amendMode = false
		m.commitInput.Blur()
		return m, nil

	case "enter":
		text := m.commitInput.Value()
		if text == "" {
			m.toast = "commit message is empty"
			m.mode = modeNormal
			m.amendMode = false
			m.commitInput.Blur()
			return m, nil
		}
		m.mode = modeNormal
		amend := m.amendMode
		m.amendMode = false
		m.commitInput.Blur()
		var err error
		if amend {
			err = git.CommitAmend(m.repoRoot, text)
		} else {
			err = git.Commit(m.repoRoot, text)
		}
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		return m, refresh(m.repoRoot)

	case "ctrl+g":
		f, err := os.CreateTemp("", "gt-commit-*.txt")
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		f.WriteString(m.commitInput.Value())
		f.Close()
		return m, execEditor(f.Name(), m.amendMode)
	}

	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m Model) handleShellKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.commitInput.Blur()
		return m, nil
	case "enter":
		cmd := m.commitInput.Value()
		m.mode = modeNormal
		m.commitInput.Blur()
		if cmd == "" {
			return m, nil
		}
		return m, execShell(cmd)
	}
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m Model) handleTagPrefixKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	switch msg.String() {
	case "d":
		return m.doDiff(row{}, m.tags)
	case "s":
		return m.doTaggedStage()
	case "u":
		return m.doTaggedUnstage()
	case "esc":
		return m, nil
	}
	m.toast = fmt.Sprintf("unknown tag command: %s", msg.String())
	return m, nil
}

func (m Model) cursorRow() row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return row{}
	}
	return m.rows[m.cursor]
}

// inlineDiffThreshold is the maximum number of diff content lines before
// gt falls back to the user's pager instead of rendering inline.
const inlineDiffThreshold = 1000

// doDiff opens a diff for the given row or tagged set.
// Per-file diffs on staged/unstaged/untracked rows open inline; everything
// else (section headers, commits, tagged sets, working-tree files) uses the pager.
func (m Model) doDiff(r row, tags map[string]bool) (tea.Model, tea.Cmd) {
	if len(tags) > 0 {
		return m, m.taggedDiff(tags)
	}

	switch r.kind {
	case rowFile:
		if r.section == git.SectionWorkingTree {
			return m.doInlineWTFile(r.file.Path)
		}
		return m.doInlineDiff(r.file.Path, r.section)
	case rowSectionHeader:
		switch r.section {
		case git.SectionUnstaged, git.SectionStaged:
			return m.doInlineDiff("", r.section)
		default:
			return m, execDiff(git.DiffCmd(m.repoRoot, r.section, ""))
		}
	case rowCommit:
		return m.doInlineCommitDiff(r.commit.SHA, "")
	case rowCommitFile:
		if r.commit != nil {
			return m.doInlineCommitDiff(r.commit.SHA, r.dirPath)
		}
	}
	return m, nil
}

// doInlineDiff parses the diff for path/section and enters modeDiff, unless
// the diff is too large in which case it falls back to the pager.
// path may be empty to diff all files in the section.
func (m Model) doInlineDiff(path string, section git.Section) (tea.Model, tea.Cmd) {
	d, err := git.ParseDiff(m.repoRoot, section, path)
	if err != nil {
		m.toast = err.Error()
		return m, nil
	}
	if d == nil || len(d.Hunks) == 0 {
		m.toast = "no diff"
		return m, nil
	}
	if d.TotalLines() > inlineDiffThreshold {
		return m, execDiff(git.DiffCmd(m.repoRoot, section, path))
	}
	// For whole-section diffs the path is empty; use the section name as display label.
	if d.Path == "" {
		switch section {
		case git.SectionUnstaged:
			d.Path = "unstaged"
		case git.SectionStaged:
			d.Path = "staged"
		}
	}
	m.diff = d
	m.diffFlat = flatDiffLines(d)
	m.diffCursor = 0
	m.diffSearch = ""
	m.diffMatches = nil
	m.diffMatchIdx = -1
	m.mode = modeDiff
	return m, nil
}

// doInlineCommitDiff opens a git show diff inline. filePath may be empty for whole-commit diffs.
func (m Model) doInlineCommitDiff(sha, filePath string) (tea.Model, tea.Cmd) {
	d, err := git.ParseCommitDiff(m.repoRoot, sha, filePath)
	if err != nil {
		m.toast = err.Error()
		return m, nil
	}
	if d == nil || len(d.Hunks) == 0 {
		m.toast = "no diff"
		return m, nil
	}
	if d.TotalLines() > inlineDiffThreshold {
		if filePath != "" {
			return m, execDiff(git.ShowFileCmd(m.repoRoot, sha, filePath))
		}
		return m, execDiff(git.ShowCmd(m.repoRoot, sha))
	}
	m.diff = d
	m.diffFlat = flatDiffLines(d)
	m.diffCursor = 0
	m.diffSearch = ""
	m.diffMatches = nil
	m.diffMatchIdx = -1
	m.mode = modeDiff
	return m, nil
}

func (m Model) taggedDiff(tags map[string]bool) tea.Cmd {
	// Collect tagged file rows grouped by section
	var cmds []tea.Cmd
	for k := range tags {
		// Find the row for this key
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile {
				cmd := git.DiffCmd(m.repoRoot, r.section, r.file.Path)
				if cmd != nil {
					cmds = append(cmds, execDiff(cmd))
				}
			} else if tagKey(r) == k && r.kind == rowCommit {
				cmd := git.ShowCmd(m.repoRoot, r.commit.SHA)
				cmds = append(cmds, execDiff(cmd))
			}
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

// doInlineWTFile opens the inline diff for a working-tree file.
// Unstaged changes take priority; falls back to staged if only staged exists.
func (m Model) doInlineWTFile(path string) (tea.Model, tea.Cmd) {
	if m.status != nil {
		for _, f := range m.status.Unstaged {
			if f.Path == path {
				return m.doInlineDiff(path, git.SectionUnstaged)
			}
		}
		for _, f := range m.status.Staged {
			if f.Path == path {
				return m.doInlineDiff(path, git.SectionStaged)
			}
		}
	}
	return m, nil
}

func (m Model) doViewFile() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowFile:
		return m, fetchFileLines(filepath.Join(m.repoRoot, r.file.Path), m.branchCtx(), r.file.Path, r.section)
	case rowCommitFile:
		sha7 := r.commit.SHA
		if len(sha7) > 7 {
			sha7 = sha7[:7]
		}
		return m, fetchFileAtRevLines(m.repoRoot, r.commit.SHA, r.dirPath, sha7)
	}
	return m, nil
}

func (m Model) doEditFile() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile {
		return m, nil
	}
	return m, execEditFile(filepath.Join(m.repoRoot, r.file.Path))
}

func (m Model) doRestoreConfirm() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile {
		return m, nil
	}
	var path string
	switch r.section {
	case git.SectionUnstaged:
		path = r.file.Path
	case git.SectionWorkingTree:
		if sf := m.statusForPath(r.file.Path); sf != nil && sf.XY[1] != '.' {
			path = r.file.Path
		}
	}
	if path == "" {
		return m, nil
	}
	m.confirmKind = confirmRestore
	m.confirmPath = path
	m.confirmPrompt = fmt.Sprintf("Discard changes to %s? [y/N]", path)
	m.mode = modeConfirm
	return m, nil
}

func (m Model) doStage() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowFile:
		if r.section == git.SectionUntracked || r.section == git.SectionUnstaged {
			if err := git.Stage(m.repoRoot, r.file.Path); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	case rowSectionHeader:
		if r.section == git.SectionUntracked {
			paths := make([]string, len(m.status.Untracked))
			for i, f := range m.status.Untracked {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Stage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		} else if r.section == git.SectionUnstaged {
			paths := make([]string, len(m.status.Unstaged))
			for i, f := range m.status.Unstaged {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Stage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		}
	}
	return m, nil
}

func (m Model) doUnstage() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowFile:
		if r.section == git.SectionStaged {
			// find next staged file to restore cursor after refresh (insertion in Unstaged
			// above would otherwise shift cursor off the intended next item)
			for i := m.cursor + 1; i < len(m.rows); i++ {
				if m.rows[i].kind == rowFile && m.rows[i].section == git.SectionStaged {
					m.cursorTargetPath = m.rows[i].file.Path
					break
				}
			}
			if err := git.Unstage(m.repoRoot, r.file.Path); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	case rowSectionHeader:
		if r.section == git.SectionStaged {
			paths := make([]string, len(m.status.Staged))
			for i, f := range m.status.Staged {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Unstage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		}
	}
	return m, nil
}

func (m Model) doTaggedStage() (Model, tea.Cmd) {
	var paths []string
	for k := range m.tags {
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile &&
				(r.section == git.SectionUntracked || r.section == git.SectionUnstaged) {
				paths = append(paths, r.file.Path)
			}
		}
	}
	if len(paths) == 0 {
		m.toast = "no stageable tagged files"
		return m, nil
	}
	if err := git.Stage(m.repoRoot, paths...); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	return m, refresh(m.repoRoot)
}

func (m Model) doTaggedUnstage() (Model, tea.Cmd) {
	var paths []string
	for k := range m.tags {
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile && r.section == git.SectionStaged {
				paths = append(paths, r.file.Path)
			}
		}
	}
	if len(paths) == 0 {
		m.toast = "no unstageable tagged files"
		return m, nil
	}
	if err := git.Unstage(m.repoRoot, paths...); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	return m, refresh(m.repoRoot)
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	m.mode = modeNormal
	m.confirmPrompt = ""
	switch msg.String() {
	case "y", "Y":
		switch m.confirmKind {
		case confirmRmFile:
			if err := git.RmFile(m.repoRoot, m.confirmPath); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		case confirmRestore:
			if err := git.Restore(m.repoRoot, m.confirmPath); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	}
	m.confirmKind = confirmNone
	m.confirmPath = ""
	return m, nil
}

func (m Model) doExpand() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowDir:
		if r.section == git.SectionWorkingTree && r.dirPath == "./" {
			if !m.wtOpen {
				m.wtOpen = true
				if len(m.wtFiles) == 0 {
					return m, fetchWTFiles(m.repoRoot)
				}
				m.buildRows()
			}
			return m, nil
		}
		if m.openDirs[r.dirPath] {
			return m, nil // already open
		}
		m.openDirs[r.dirPath] = true
		if r.section == git.SectionUntracked {
			return m, expandUntrackedDir(m.repoRoot, r.dirPath)
		}
		// Working tree dir: files already in wtFiles, just rebuild
		m.buildRows()
		return m, nil
	case rowCommit:
		if r.commit == nil {
			return m, nil
		}
		if _, ok := m.openCommits[r.commit.SHA]; ok {
			return m, nil // already expanded
		}
		return m, fetchCommitFiles(m.repoRoot, r.commit.SHA)
	}
	return m, nil
}

func (m Model) doCollapse() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowDir:
		if r.section == git.SectionWorkingTree && r.dirPath == "./" {
			if m.wtOpen {
				m.wtOpen = false
				m.buildRows()
			}
			return m, nil
		}
		if m.openDirs[r.dirPath] {
			m.openDirs[r.dirPath] = false
			m.buildRows()
		} else {
			// already closed: navigate up to the section header
			for i := m.cursor - 1; i >= 0; i-- {
				if m.rows[i].kind == rowSectionHeader {
					m.cursor = i
					break
				}
			}
		}
	case rowFile:
		// If this file is a child (depth>0), collapse its parent dir
		if r.depth > 0 {
			// find parent rowDir above cursor
			for i := m.cursor - 1; i >= 0; i-- {
				if m.rows[i].kind == rowDir {
					if m.rows[i].section == git.SectionWorkingTree && m.rows[i].dirPath == "./" {
						m.wtOpen = false
					} else {
						m.openDirs[m.rows[i].dirPath] = false
					}
					m.cursor = i
					m.buildRows()
					break
				}
			}
		}
	case rowCommit:
		if r.commit != nil {
			if _, ok := m.openCommits[r.commit.SHA]; ok {
				delete(m.openCommits, r.commit.SHA)
				m.buildRows()
			}
		}
	case rowCommitBody, rowCommitFile:
		if r.commit != nil {
			delete(m.openCommits, r.commit.SHA)
			// move cursor up to the parent commit row
			for i := m.cursor - 1; i >= 0; i-- {
				if m.rows[i].kind == rowCommit && m.rows[i].commit != nil && m.rows[i].commit.SHA == r.commit.SHA {
					m.cursor = i
					break
				}
			}
			m.buildRows()
		}
	}
	return m, nil
}

func (m Model) doRmCached() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile || r.section != git.SectionWorkingTree {
		return m, nil
	}
	if err := git.RmCached(m.repoRoot, r.file.Path); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	// remove from wtFiles so the row disappears without a full refresh
	newFiles := m.wtFiles[:0]
	for _, f := range m.wtFiles {
		if f.Path != r.file.Path {
			newFiles = append(newFiles, f)
		}
	}
	m.wtFiles = newFiles
	m.buildRows()
	return m, refresh(m.repoRoot)
}

func (m Model) doRmFileConfirm() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile || r.section != git.SectionWorkingTree {
		return m, nil
	}
	m.confirmKind = confirmRmFile
	m.confirmPath = r.file.Path
	m.confirmPrompt = fmt.Sprintf("Delete %s from disk? [y/N]", r.file.Path)
	m.mode = modeConfirm
	return m, nil
}

func expandUntrackedDir(repoRoot, dirPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.ListUntrackedInDir(repoRoot, dirPath)
		return dirContentsMsg{dirPath: dirPath, files: files, err: err}
	}
}

func fetchWTFiles(cwd string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.ListTrackedUnder(cwd)
		return wtFilesMsg{files: files, err: err}
	}
}

func fetchCommitFiles(repoRoot, sha string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetCommitFiles(repoRoot, sha)
		if err != nil {
			return commitFilesMsg{sha: sha, err: err}
		}
		body, _ := git.GetCommitBody(repoRoot, sha)
		return commitFilesMsg{sha: sha, files: files, body: body}
	}
}

func refresh(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		status, err := git.GetStatus(repoRoot)
		if err != nil {
			return refreshMsg{err: err}
		}
		log, err := git.GetLog(repoRoot)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{status: status, log: log}
	}
}

// handleDiffKey handles keypresses in modeDiff.
//
// KEEP IN SYNC: navigation and search here duplicate handleFileKey in file_view.go.
// If you change one (add keys, fix page size, etc.) update the other, or unify them.
func (m Model) handleDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.diffSearching {
		return m.handleDiffSearchKey(msg)
	}
	switch msg.String() {
	case "q", "esc":
		m.mode = m.prevMode
		m.prevMode = modeNormal
		m.diff = nil
		m.diffFlat = nil
		m.diffSearch = ""
		m.diffMatches = nil
		m.diffMatchIdx = -1
	case "j", "down":
		if m.diffCursor < len(m.diffFlat)-1 {
			m.diffCursor++
		}
	case "k", "up":
		if m.diffCursor > 0 {
			m.diffCursor--
		}
	case " ", "ctrl+d":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.diffCursor += pageSize
		if m.diffCursor >= len(m.diffFlat) {
			m.diffCursor = len(m.diffFlat) - 1
		}
	case "ctrl+u":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.diffCursor -= pageSize
		if m.diffCursor < 0 {
			m.diffCursor = 0
		}
	case "g":
		m.diffCursor = 0
	case "G":
		if len(m.diffFlat) > 0 {
			m.diffCursor = len(m.diffFlat) - 1
		}
	case "]":
		m.diffCursor = m.nextHunkStart(m.diffCursor)
	case "[":
		m.diffCursor = m.prevHunkStart(m.diffCursor)
	case "/":
		m.diffSearching = true
		m.commitInput.Reset()
		m.commitInput.Placeholder = "search…"
		m.commitInput.Focus()
		return m, nil
	case "n":
		if len(m.diffMatches) > 0 {
			m.diffMatchIdx = (m.diffMatchIdx + 1) % len(m.diffMatches)
			m.diffCursor = m.diffMatches[m.diffMatchIdx]
		}
	case "N":
		if len(m.diffMatches) > 0 {
			m.diffMatchIdx = (m.diffMatchIdx - 1 + len(m.diffMatches)) % len(m.diffMatches)
			m.diffCursor = m.diffMatches[m.diffMatchIdx]
		}
	case "e":
		if m.diff != nil {
			return m, execEditFile(filepath.Join(m.repoRoot, m.diff.Path))
		}
	case "v":
		if m.diff == nil {
			return m, nil
		}
		if m.diff.SHA == "" {
			return m, fetchFileLines(filepath.Join(m.repoRoot, m.diff.Path), m.branchCtx(), m.diff.Path, m.diff.Section)
		}
		if m.diff.Path == m.diff.SHA {
			m.toast = "press v on a specific file, not a whole-commit diff"
			return m, nil
		}
		return m, fetchFileAtRevLines(m.repoRoot, m.diff.SHA, m.diff.Path, m.diff.SHA)
	case "L":
		if m.diff != nil {
			return m, m.diffFallbackCmd()
		}
	}
	return m, nil
}

func (m Model) doFileLog() (Model, tea.Cmd) {
	r := m.cursorRow()
	var path string
	switch r.kind {
	case rowFile:
		path = r.file.Path
	case rowCommitFile:
		path = r.dirPath
	default:
		return m, nil
	}
	return m, fetchFileLog(m.repoRoot, path)
}

func fetchFileLog(repoRoot, path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := git.GetFileLog(repoRoot, path)
		return fileLogMsg{path: path, entries: entries, err: err}
	}
}

func fetchFileLogCommitFiles(repoRoot, sha string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetCommitFiles(repoRoot, sha)
		if err != nil {
			return fileLogCommitFilesMsg{sha: sha, err: err}
		}
		body, _ := git.GetCommitBody(repoRoot, sha)
		return fileLogCommitFilesMsg{sha: sha, files: files, body: body}
	}
}

func (m Model) handleFileLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.fileLogRows()
	clamp := func() {
		if len(rows) == 0 {
			m.fileLogCursor = 0
			return
		}
		if m.fileLogCursor < 0 {
			m.fileLogCursor = 0
		}
		if m.fileLogCursor >= len(rows) {
			m.fileLogCursor = len(rows) - 1
		}
	}

	switch msg.String() {
	case "q", "esc":
		m.mode = modeNormal
		m.fileLogEntries = nil
		m.fileLogPath = ""
		m.fileLogOpen = make(map[string][]git.FileEntry)
		m.fileLogBodies = make(map[string][]string)

	case "j", "down":
		m.fileLogCursor++
		clamp()
	case "k", "up":
		m.fileLogCursor--
		clamp()
	case "g":
		m.fileLogCursor = 0
	case "G":
		if len(rows) > 0 {
			m.fileLogCursor = len(rows) - 1
		}
	case " ", "ctrl+d":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.fileLogCursor += pageSize
		clamp()
	case "ctrl+u":
		pageSize := m.height / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.fileLogCursor -= pageSize
		clamp()

	case "l", "right", "enter":
		if m.fileLogCursor >= len(rows) {
			break
		}
		r := rows[m.fileLogCursor]
		if r.kind == rowCommit && r.commit != nil {
			if _, ok := m.fileLogOpen[r.commit.SHA]; !ok {
				return m, fetchFileLogCommitFiles(m.repoRoot, r.commit.SHA)
			}
		}
		if r.kind == rowCommitFile && r.commit != nil {
			m.prevMode = modeFileLog
			return m.doInlineCommitDiff(r.commit.SHA, r.dirPath)
		}

	case "h", "left":
		if m.fileLogCursor >= len(rows) {
			break
		}
		r := rows[m.fileLogCursor]
		if r.kind == rowCommit && r.commit != nil {
			if _, ok := m.fileLogOpen[r.commit.SHA]; ok {
				delete(m.fileLogOpen, r.commit.SHA)
				delete(m.fileLogBodies, r.commit.SHA)
			}
		} else if (r.kind == rowCommitFile || r.kind == rowCommitBody) && r.commit != nil {
			delete(m.fileLogOpen, r.commit.SHA)
			delete(m.fileLogBodies, r.commit.SHA)
			for i := m.fileLogCursor - 1; i >= 0; i-- {
				if rows[i].kind == rowCommit && rows[i].commit != nil && rows[i].commit.SHA == r.commit.SHA {
					m.fileLogCursor = i
					break
				}
			}
		}

	case "d":
		if m.fileLogCursor >= len(rows) {
			break
		}
		r := rows[m.fileLogCursor]
		m.prevMode = modeFileLog
		if r.kind == rowCommit && r.commit != nil {
			return m.doInlineCommitDiff(r.commit.SHA, m.fileLogPath)
		}
		if r.kind == rowCommitFile && r.commit != nil {
			return m.doInlineCommitDiff(r.commit.SHA, r.dirPath)
		}
		m.prevMode = modeNormal

	case "v":
		if m.fileLogCursor >= len(rows) {
			break
		}
		r := rows[m.fileLogCursor]
		if r.kind == rowCommitFile && r.commit != nil {
			sha7 := r.commit.SHA
			if len(sha7) > 7 {
				sha7 = sha7[:7]
			}
			m.prevMode = modeFileLog
			return m, fetchFileAtRevLines(m.repoRoot, r.commit.SHA, r.dirPath, sha7)
		}
	}
	return m, nil
}

// handleDiffSearchKey handles keypresses while the search input is open.
func (m Model) handleDiffSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		pattern := m.commitInput.Value()
		m.diffSearch = pattern
		m.diffSearching = false
		m.commitInput.Blur()
		m.commitInput.Reset()
		m.diffMatches = m.computeDiffMatches(pattern)
		m.diffMatchIdx = -1
		if len(m.diffMatches) > 0 {
			// Jump to first match at or after current cursor
			m.diffMatchIdx = 0
			for i, idx := range m.diffMatches {
				if idx >= m.diffCursor {
					m.diffMatchIdx = i
					break
				}
			}
			m.diffCursor = m.diffMatches[m.diffMatchIdx]
		}
		return m, nil
	case "esc":
		m.diffSearching = false
		m.commitInput.Blur()
		m.commitInput.Reset()
		return m, nil
	default:
		var cmd tea.Cmd
		m.commitInput, cmd = m.commitInput.Update(msg)
		return m, cmd
	}
}

// computeDiffMatches returns the flat indices of all lines whose content
// contains pattern (case-insensitive). Hunk/file headers are excluded.
func (m Model) computeDiffMatches(pattern string) []int {
	if pattern == "" || m.diff == nil {
		return nil
	}
	lower := strings.ToLower(pattern)
	var matches []int
	for i, vl := range m.diffFlat {
		if vl.lineIdx < 0 {
			continue
		}
		content := m.diff.Hunks[vl.hunkIdx].Lines[vl.lineIdx].Content
		if strings.Contains(strings.ToLower(content), lower) {
			matches = append(matches, i)
		}
	}
	return matches
}

// diffFallbackCmd returns a tea.Cmd that opens the current diff in the user's pager.
func (m Model) diffFallbackCmd() tea.Cmd {
	d := m.diff
	if d.SHA != "" {
		// commit diff
		if d.Path == d.SHA {
			return execDiff(git.ShowCmd(m.repoRoot, d.SHA))
		}
		return execDiff(git.ShowFileCmd(m.repoRoot, d.SHA, d.Path))
	}
	return execDiff(git.DiffCmd(m.repoRoot, d.Section, d.Path))
}

// nextHunkStart returns the flat index of the next hunk header (lineIdx == -1) after cur.
func (m Model) nextHunkStart(cur int) int {
	for i := cur + 1; i < len(m.diffFlat); i++ {
		if m.diffFlat[i].lineIdx == -1 {
			return i
		}
	}
	return cur
}

// prevHunkStart returns the flat index of the current hunk's @@ header, or the
// previous hunk's @@ header if cur is already on a hunk header or file separator.
func (m Model) prevHunkStart(cur int) int {
	if cur == 0 {
		return 0
	}
	curHunkIdx := m.diffFlat[cur].hunkIdx
	isOnHeader := m.diffFlat[cur].lineIdx == -1
	targetHunk := curHunkIdx
	if isOnHeader {
		targetHunk = curHunkIdx - 1
	}
	if targetHunk < 0 {
		return cur
	}
	for i := 0; i < cur; i++ {
		if m.diffFlat[i].hunkIdx == targetHunk && m.diffFlat[i].lineIdx == -1 {
			return i
		}
	}
	return cur
}
