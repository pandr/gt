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
