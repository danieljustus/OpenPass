# OpenPass demo tape (charmbracelet/vhs)
#
# Renders docs/assets/openpass-demo.gif. Re-render with:
#   go build -o /tmp/openpass . && vhs docs/vhs/openpass-demo.tape
#
# Requires: vhs, ttyd, ffmpeg, and an `openpass` binary on PATH (here: /tmp/openpass).

Output "docs/assets/openpass-demo.gif"

Set FontSize 18
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 50ms

Env PATH "/tmp:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
Env OPENPASS_VAULT "/tmp/demo-vault"
Env OPENPASS_PASSPHRASE "demo-pass-2026"

Hide
Type "rm -rf /tmp/demo-vault"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 200ms
Show

Type "# OpenPass — password manager for terminal users and AI agents"
Enter
Sleep 1200ms

Type "openpass init /tmp/demo-vault"
Enter
Sleep 1500ms
Type "demo-pass-2026"
Enter
Sleep 2500ms

Type "openpass add github --username demo@example.com --value xK9mP2vL7nQ4"
Enter
Sleep 1800ms

Type "openpass get github"
Enter
Sleep 2500ms

Type "openpass generate --length 24 --symbols"
Enter
Sleep 2200ms

Type "openpass mcp-config claude-code"
Enter
Sleep 3500ms
