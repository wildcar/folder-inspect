//go:build !windows

package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// reveal opens the file manager at the path (macOS selects the item; on
// Linux the containing folder is opened).
func reveal(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}
	dir := path
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		dir = filepath.Dir(path)
	}
	return exec.Command("xdg-open", dir).Start()
}
