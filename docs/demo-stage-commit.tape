Output docs/demo-stage-commit.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "SeaShells"
Set Padding 24
Set TypingSpeed 70ms

Hide
Type "cd /Users/petera/git/gt && go build -o /tmp/gt-demo-bin ./cmd/gt/ 2>/dev/null && bash docs/demo-setup.sh && mkdir -p /tmp/gt-demo-path && cp /tmp/gt-demo-bin /tmp/gt-demo-path/gt && cd /tmp/gt-demo && export PATH=/tmp/gt-demo-path:$PATH && export PS1='$ ' && export GT_DEMO_KEYS=1"
Enter
Sleep 12s
Type "clear"
Enter
Sleep 500ms
Show

# Launch gt
Type "gt"
Sleep 500ms
Enter
Sleep 2s

# Navigate to README.md in Unstaged (j j j j from Untracked header)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Stage README.md — it moves to Staged; cursor lands on src/main.go
Sleep 500ms
Type "s"
Sleep 1400ms

# Stage src/main.go
Type "s"
Sleep 1400ms

# Open commit prompt and type message
Sleep 400ms
Type "c"
Sleep 900ms
Type "add usage section and update main"
Sleep 700ms
Enter
Sleep 2500ms
