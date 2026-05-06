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

func CommitAmend(repoRoot, message string) error {
	cmd := exec.Command("git", "commit", "--amend", "-m", message)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: strings.TrimSpace(string(out)), Err: err}
	}
	return nil
}

func CommitAmendFile(repoRoot, filePath string) error {
	cmd := exec.Command("git", "commit", "--amend", "-F", filePath)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: strings.TrimSpace(string(out)), Err: err}
	}
	return nil
}
