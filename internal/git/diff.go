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
			args = []string{"diff", "--no-index", "/dev/null", path}
		} else {
			return nil
		}
	case SectionUnstaged:
		if path != "" {
			args = []string{"diff", "--", path}
		} else {
			args = []string{"diff"}
		}
	case SectionStaged:
		if path != "" {
			args = []string{"diff", "--staged", "--", path}
		} else {
			args = []string{"diff", "--staged"}
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
	cmd := exec.Command("git", "show", sha)
	cmd.Dir = repoRoot
	return cmd
}
