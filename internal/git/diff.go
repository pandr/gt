package git

import "os/exec"

type Section int

const (
	SectionUntracked Section = iota
	SectionUnstaged
	SectionStaged
	SectionLog
)

// DiffCmd returns an exec.Cmd for the given diff target.
// Returns nil if there is nothing to diff.
func DiffCmd(repoRoot string, section Section, path string) *exec.Cmd {
	var args []string
	switch section {
	case SectionUntracked:
		if path != "" {
			args = []string{"diff", "--color=always", "--no-index", "/dev/null", path}
		} else {
			return nil
		}
	case SectionUnstaged:
		if path != "" {
			args = []string{"diff", "--color=always", "--", path}
		} else {
			args = []string{"diff", "--color=always"}
		}
	case SectionStaged:
		if path != "" {
			args = []string{"diff", "--color=always", "--staged", "--", path}
		} else {
			args = []string{"diff", "--color=always", "--staged"}
		}
	default:
		return nil
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	return cmd
}

// ShowCmd returns an exec.Cmd for `git show <sha>`.
func ShowCmd(repoRoot, sha string) *exec.Cmd {
	cmd := exec.Command("git", "show", "--color=always", sha)
	cmd.Dir = repoRoot
	return cmd
}
