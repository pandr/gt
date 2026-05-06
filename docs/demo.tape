Output docs/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "SeaShells"
Set Padding 24
Set TypingSpeed 70ms

# Hidden setup: build gt and create throwaway demo repo in /tmp
Hide
Type "cd /Users/petera/git/gt && go build -o /tmp/gt-demo-bin ./cmd/gt/ 2>/dev/null && bash docs/demo-setup.sh && cd /tmp/gt-demo && export PS1='$ '"
Enter
Sleep 12s
Type "clear"
Enter
Sleep 500ms
Show

# Launch gt in the throwaway repo
Type "/tmp/gt-demo-bin"
Enter
Sleep 2s

# --- Navigate down to docs/ ---
Type "j"
Sleep 300ms

# Expand docs/ with l
Sleep 500ms
Type "l"
Sleep 1100ms

# Collapse with h
Sleep 400ms
Type "h"
Sleep 800ms

# Navigate to README.md in Unstaged (3 steps from docs/)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Diff README.md — pager opens, read the diff, then quit
Sleep 500ms
Type "d"
Sleep 2800ms
Type "q"
Sleep 900ms

# Stage README.md (cursor is still on it after pager closes)
Sleep 500ms
Type "s"
Sleep 1400ms

# Cursor lands on src/main.go — tag it
Sleep 400ms
Type "t"
Sleep 1100ms

# Navigate up to src/handler.go in Untracked (2 steps from src/main.go)
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms

# Tag src/handler.go
Sleep 400ms
Type "t"
Sleep 1100ms

# Batch stage both tagged files with ;s
Sleep 400ms
Type ";"
Sleep 400ms
Type "s"
Sleep 1600ms

# Open commit prompt and commit
Sleep 400ms
Type "c"
Sleep 900ms
Type "add handler and update main"
Sleep 700ms
Enter
Sleep 2500ms
