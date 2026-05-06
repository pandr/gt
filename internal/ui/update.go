package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

type clearKeyHintMsg struct{ token int }

// setKeyHint sets a brief key label and returns a timer cmd that clears it.
// Only active when GT_DEMO_KEYS=1; no-ops otherwise.
func setKeyHint(m Model, key string) (Model, tea.Cmd) {
	if os.Getenv("GT_DEMO_KEYS") == "" {
		return m, nil
	}
	m.keyHintToken++
	m.keyHint = key
	token := m.keyHintToken
	return m, tea.Tick(700*time.Millisecond, func(time.Time) tea.Msg {
		return clearKeyHintMsg{token: token}
	})
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
		} else {
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

	case commitFilesMsg:
		if msg.err == nil {
			m.openCommits[msg.sha] = msg.files
			m.commitBodies[msg.sha] = msg.body
			m.buildRows()
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

	case clearKeyHintMsg:
		if msg.token == m.keyHintToken {
			m.keyHint = ""
		}
		return m, nil

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
		m, hintCmd := setKeyHint(m, "l")
		m2, cmd := m.doExpand()
		return m2, tea.Batch(hintCmd, cmd)

	case "left", "h":
		m, hintCmd := setKeyHint(m, "h")
		m2, cmd := m.doCollapse()
		return m2, tea.Batch(hintCmd, cmd)

	case "d", " ":
		m, hintCmd := setKeyHint(m, "d")
		return m, tea.Batch(hintCmd, m.doDiff(m.cursorRow(), nil))

	case "v":
		m, hintCmd := setKeyHint(m, "v")
		m2, cmd := m.doViewFile()
		return m2, tea.Batch(hintCmd, cmd)

	case "V":
		m, hintCmd := setKeyHint(m, "V")
		m2, cmd := m.doEditFile()
		return m2, tea.Batch(hintCmd, cmd)

	case "r":
		m, hintCmd := setKeyHint(m, "r")
		m2, cmd := m.doRestoreConfirm()
		return m2, tea.Batch(hintCmd, cmd)

	case "x":
		m, hintCmd := setKeyHint(m, "x")
		m2, cmd := m.doRmCached()
		return m2, tea.Batch(hintCmd, cmd)

	case "X":
		m, hintCmd := setKeyHint(m, "X")
		m2, cmd := m.doRmFileConfirm()
		return m2, tea.Batch(hintCmd, cmd)

	case "s":
		m, hintCmd := setKeyHint(m, "s")
		m2, cmd := m.doStage()
		return m2, tea.Batch(hintCmd, cmd)

	case "u":
		m, hintCmd := setKeyHint(m, "u")
		m2, cmd := m.doUnstage()
		return m2, tea.Batch(hintCmd, cmd)

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
		m, hintCmd := setKeyHint(m, "t")
		return m, hintCmd

	case ";":
		if len(m.tags) > 0 {
			m.mode = modeTagPrefix
			m, hintCmd := setKeyHint(m, ";")
			return m, hintCmd
		}

	case "T":
		m.tags = make(map[string]bool)
		m, hintCmd := setKeyHint(m, "T")
		return m, hintCmd

	case "!":
		m.mode = modeShell
		m.commitInput.Placeholder = "Shell command (Enter=run  Esc=cancel)"
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		m, hintCmd := setKeyHint(m, "!")
		return m, tea.Batch(hintCmd, textinput.Blink)

	case "c":
		m.mode = modeCommit
		m.commitInput.Placeholder = "Commit message (Enter=commit  Ctrl-g=editor  Esc=cancel)"
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		m, hintCmd := setKeyHint(m, "c")
		return m, tea.Batch(hintCmd, textinput.Blink)

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
		m, hintCmd := setKeyHint(m, "A")
		return m, tea.Batch(hintCmd, textinput.Blink)
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
		m, hintCmd := setKeyHint(m, ";d")
		return m, tea.Batch(hintCmd, m.doDiff(row{}, m.tags))
	case "s":
		m, hintCmd := setKeyHint(m, ";s")
		m2, cmd := m.doTaggedStage()
		return m2, tea.Batch(hintCmd, cmd)
	case "u":
		m, hintCmd := setKeyHint(m, ";u")
		m2, cmd := m.doTaggedUnstage()
		return m2, tea.Batch(hintCmd, cmd)
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

// doDiff builds and executes the diff for the cursor row or tagged set.
func (m Model) doDiff(r row, tags map[string]bool) tea.Cmd {
	// Tagged diff: if we have tags, diff all of them
	if len(tags) > 0 {
		return m.taggedDiff(tags)
	}

	switch r.kind {
	case rowFile:
		if r.section == git.SectionWorkingTree {
			return m.diffWTFile(r.file.Path)
		}
		cmd := git.DiffCmd(m.repoRoot, r.section, r.file.Path)
		return execDiff(cmd)
	case rowSectionHeader:
		cmd := git.DiffCmd(m.repoRoot, r.section, "")
		return execDiff(cmd)
	case rowCommit:
		cmd := git.ShowCmd(m.repoRoot, r.commit.SHA)
		return execDiff(cmd)
	case rowCommitFile:
		if r.commit != nil {
			cmd := git.ShowFileCmd(m.repoRoot, r.commit.SHA, r.dirPath)
			return execDiff(cmd)
		}
	}
	return nil
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

func (m Model) diffWTFile(path string) tea.Cmd {
	var cmds []tea.Cmd
	if m.status != nil {
		for _, f := range m.status.Unstaged {
			if f.Path == path {
				if cmd := git.DiffCmd(m.repoRoot, git.SectionUnstaged, path); cmd != nil {
					cmds = append(cmds, execDiff(cmd))
				}
				break
			}
		}
		for _, f := range m.status.Staged {
			if f.Path == path {
				if cmd := git.DiffCmd(m.repoRoot, git.SectionStaged, path); cmd != nil {
					cmds = append(cmds, execDiff(cmd))
				}
				break
			}
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

func (m Model) doViewFile() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile {
		return m, nil
	}
	return m, execViewFile(filepath.Join(m.repoRoot, r.file.Path))
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
					return m, fetchWTFiles(m.cwd)
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
	case rowCommitFile:
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
