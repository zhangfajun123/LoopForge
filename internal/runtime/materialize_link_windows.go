//go:build windows

package runtime

import (
	"io"
	"os"
	"path/filepath"
)

// clearMaterializedLink removes the copy a previous materialization left. The
// Windows binding is a plain file or directory tree, so RemoveAll also covers
// a directory whose shim was copied instead of linked.
func clearMaterializedLink(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(path)
}

// materializeRelativeLink copies the target instead of creating a symlink.
// Windows only allows symlinks for elevated processes or Developer Mode, and a
// missing privilege must not fail the whole Agent materialization. The binding
// only needs the content to be readable at path.
func materializeRelativeLink(path, target string) error {
	source := target
	if !filepath.IsAbs(source) {
		source = filepath.Join(filepath.Dir(path), target)
	}
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDirectoryTree(source, path)
	}
	return copyFileContent(source, path, info.Mode().Perm())
}

func copyDirectoryTree(source, dest string) error {
	return filepath.WalkDir(source, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return copyFileContent(current, target, info.Mode().Perm())
	})
}

func copyFileContent(source, dest string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
