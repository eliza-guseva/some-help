package internal

import (
	"os/exec"
)


func PasteToClaudeApp() error {
	cmd := exec.Command("osascript", "-e", `
		tell application "System Events"
			keystroke space using {option down}
			delay 0.5
			keystroke "v" using {command down}
			delay 0.2
			keystroke return
		end tell
	`)
	return cmd.Run()
}
