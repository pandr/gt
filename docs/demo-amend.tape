Output docs/demo-amend.gif

Set Shell "bash"
Set FontSize 20
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

# Jump to bottom (initial commit), then navigate up to most recent commit
Type "G"
Sleep 700ms
Type "k"
Sleep 300ms
Type "k"
Sleep 500ms

# Amend — inline prompt opens with current message prefilled
Type "A"
Sleep 900ms

# Clear the prefilled message and type a corrected one
Sleep 300ms
Ctrl+u
Sleep 400ms
Type "release v1.0.0"
Sleep 700ms
Enter
Sleep 2500ms

Type "q"
Sleep 300ms
