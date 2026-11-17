//go:build !windows

package runtime

import (
	"fmt"
	"os"
)

// clearMaterializedLink removes a previous binding symlink. A real file or
// directory at that path is a bug in the caller's layout, not something to
// overwrite.
func clearMaterializedLink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("Agent binding path is not a symlink: %s", path)
	}
	return os.Remove(path)
}

// materializeRelativeLink creates the relative symlink the agent bindings use.
func materializeRelativeLink(path, target string) error {
	return os.Symlink(target, path)
}
