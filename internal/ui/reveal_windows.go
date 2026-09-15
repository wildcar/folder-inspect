//go:build windows

package ui

import (
	"os/exec"
	"strings"
	"syscall"
)

// reveal opens Explorer with the file or folder selected. The command line
// is built by hand: Go would quote the whole "/select,C:\path with spaces"
// argument, which Explorer does not understand (it then opens Documents).
// Explorer wants exactly: explorer.exe /select,"C:\path with spaces\file".
func reveal(path string) error {
	clean := strings.ReplaceAll(path, `"`, "")
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + clean + `"`}
	return cmd.Start()
}
