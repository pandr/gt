package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Rail — the 1-cell ┊ column; color is the "attendance light"
	railActive = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a")) // amber: section has work
	railUnder  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a6e44")) // amber.dim: file under active section
	railGhost  = lipgloss.NewStyle().Foreground(lipgloss.Color("#34322b")) // ghost: empty / separator
	railFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b5849")) // faint: log section

	// Section headers
	sectHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a")).Bold(true) // amber: section with work
	sectHeaderQ = lipgloss.NewStyle().Foreground(lipgloss.Color("#a39e8d")).Bold(true) // fg.soft: "Recent commits"

	// Branch line / refs
	branchIris = lipgloss.NewStyle().Foreground(lipgloss.Color("#7fb8c4")).Bold(true) // iris bold: local branch
	remoteIris = lipgloss.NewStyle().Foreground(lipgloss.Color("#4d7e8a"))            // iris.dim: remote ref
	shaIris    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4d7e8a"))            // iris.dim: commit SHA
	refTagIris = lipgloss.NewStyle().Foreground(lipgloss.Color("#b491c8"))            // mauve: git tags

	// File status indicators
	addStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#94b87a")) // moss: A ? +
	delStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c87766")) // terracotta: D -
	modStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#d6a96a")) // amber: M modified
	tagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#b491c8")) // mauve: * tagged row

	// Text hierarchy
	fgSoft  = lipgloss.NewStyle().Foreground(lipgloss.Color("#a39e8d")) // de-emphasized
	fgFaint = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b5849")) // hint bar, separators

	// Status bar chrome
	styleStatusBar = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b5849"))
	styleToast     = lipgloss.NewStyle().Foreground(lipgloss.Color("#c87766")).Bold(true)
	styleHelp      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)
