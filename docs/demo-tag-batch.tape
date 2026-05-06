Output docs/demo-tag-batch.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "SeaShells"
Set Padding 24
Set TypingSpeed 70ms

Hide
Type "cd /Users/petera/git/gt && go build -o /tmp/gt-demo-bin ./cmd/gt/ 2>/dev/null && bash docs/demo-setup.sh && mkdir -p /tmp/gt-demo-path && cp /tmp/gt-demo-bin /tmp/gt-demo-path/gt && cd /tmp/gt-demo && export PATH=/tmp/gt-demo-path:$PATH && export PS1='$ '"
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

# Navigate to src/handler.go in Untracked (j j from row 0)
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms

# Tag src/handler.go — magenta * appears
Sleep 500ms
Type "t"
Sleep 1100ms

# Navigate down to src/main.go in Unstaged (j j j from row 2)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Tag src/main.go — status bar shows [2 tagged]
Sleep 500ms
Type "t"
Sleep 1100ms

# Batch-stage both tagged files with ;s
Sleep 400ms
Type ";"
Sleep 400ms
Type "s"
Sleep 1600ms

Type "q"
Sleep 300ms
