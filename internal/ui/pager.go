package ui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// execDiff runs a git command via the user's pager (ExecProcess suspends TUI).
func execDiff(gitCmd *exec.Cmd) tea.Cmd {
	if gitCmd == nil {
		return nil
	}
	return tea.ExecProcess(gitCmd, func(err error) tea.Msg {
		return execDoneMsg{err: err}
	})
}

// execEditor opens $EDITOR on a temp file and returns its path via msg.
func execEditor(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, filePath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{filePath: filePath, err: err}
	})
}

type execDoneMsg struct{ err error }
type editorDoneMsg struct {
	filePath string
	err      error
}
