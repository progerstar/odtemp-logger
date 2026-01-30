package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// escapeAppleScript экранирует строку для AppleScript
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// escapePowerShell экранирует строку для PowerShell
func escapePowerShell(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

// showSystemDialog показывает системный диалог с сообщением
func showSystemDialog(title, message string) {
	switch runtime.GOOS {
	case "darwin":
		script := `display dialog "` + escapeAppleScript(message) + `" with title "` + escapeAppleScript(title) + `" buttons {"OK"} default button "OK" with icon caution`
		exec.Command("osascript", "-e", script).Run()

	case "linux":
		if _, err := exec.LookPath("zenity"); err == nil {
			exec.Command("zenity", "--warning", "--title="+title, "--text="+message).Run()
		} else if _, err := exec.LookPath("kdialog"); err == nil {
			exec.Command("kdialog", "--sorry", message, "--title", title).Run()
		}

	case "windows":
		script := `[System.Windows.Forms.MessageBox]::Show('` + escapePowerShell(message) + `', '` + escapePowerShell(title) + `', 'OK', 'Warning')`
		exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms;"+script).Run()
	}
}
