package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petera/gt/internal/ui"
)

func main() {
	repoRoot, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gt: not inside a git repository")
		os.Exit(1)
	}

	m := ui.NewModel(repoRoot)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gt: %v\n", err)
		os.Exit(1)
	}
}

func repoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
