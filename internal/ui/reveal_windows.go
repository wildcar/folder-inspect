//go:build windows

package ui

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// reveal opens Explorer: a folder is opened itself, a file is shown selected
// in its folder. The command line is built by hand: Go would quote the whole
// "/select,C:\path with spaces" argument, which Explorer does not understand
// (it then opens Documents). Explorer wants exactly:
// explorer.exe /select,"C:\path with spaces\file".
func reveal(path string) error {
	clean := strings.ReplaceAll(path, `"`, "")
	line := `explorer.exe /select,"` + clean + `"`
	if st, err := os.Stat(clean); err == nil && st.IsDir() {
		line = `explorer.exe "` + clean + `"`
	}
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
	return cmd.Start()
}
