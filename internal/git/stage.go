package git

import "os/exec"

func Stage(repoRoot string, paths ...string) error {
	args := append([]string{"add", "--"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: string(out), Err: err}
	}
	return nil
}

func Unstage(repoRoot string, paths ...string) error {
	args := append([]string{"restore", "--staged", "--"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: string(out), Err: err}
	}
	return nil
}

// RmCached untracks a file but keeps it on disk (git rm --cached).
func RmCached(repoRoot, path string) error {
	cmd := exec.Command("git", "rm", "--cached", "--", path)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: string(out), Err: err}
	}
	return nil
}

// RmFile untracks and deletes a file from disk (git rm).
func RmFile(repoRoot, path string) error {
	cmd := exec.Command("git", "rm", "--", path)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Out: string(out), Err: err}
	}
	return nil
}

type GitError struct {
	Out string
	Err error
}

func (e *GitError) Error() string {
	if e.Out != "" {
		return e.Out
	}
	return e.Err.Error()
}
