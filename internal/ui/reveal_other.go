//go:build !windows

package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// reveal opens the file manager: a folder is opened itself; a file is
// selected on macOS and its containing folder is opened on Linux.
func reveal(path string) error {
	st, err := os.Stat(path)
	isDir := err == nil && st.IsDir()
	if runtime.GOOS == "darwin" {
		if isDir {
			return exec.Command("open", path).Start()
		}
		return exec.Command("open", "-R", path).Start()
	}
	dir := path
	if !isDir {
		dir = filepath.Dir(path)
	}
	return exec.Command("xdg-open", dir).Start()
}
