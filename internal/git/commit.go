package git

import (
	"os/exec"
	"strings"
)

func Commit(repoRoot, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: strings.TrimSpace(string(out)), Err: err}
	}
	return nil
}

func CommitFile(repoRoot, filePath string) error {
	cmd := exec.Command("git", "commit", "-F", filePath)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: strings.TrimSpace(string(out)), Err: err}
	}
	return nil
}
