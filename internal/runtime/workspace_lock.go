package runtime

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

const sandboxLockDir = ".locks"

func (m *Manager) lockPath(id string) string {
	return filepath.Join(m.root, sandboxLockDir, safeID(id)+".lock")
}

func (m *Manager) acquireLock(ctx context.Context, id string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Join(m.root, sandboxLockDir), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(m.lockPath(id), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := flockExclusive(ctx, f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func flockExclusive(ctx context.Context, f *os.File) error {
	for {
		locked, err := tryFileLock(f)
		if err != nil {
			return err
		}
		if locked {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func unlockFile(f *os.File) error {
	if f == nil {
		return nil
	}
	_ = unlockFileLock(f)
	return f.Close()
}

func (m *Manager) workspaceInUse(name string) bool {
	if m == nil || name == "" || name == sandboxLockDir {
		return false
	}
	path := m.lockPath(name)
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return false
	}
	defer f.Close()
	locked, lockErr := tryFileLock(f)
	if lockErr != nil {
		return true
	}
	if !locked {
		return true
	}
	_ = unlockFileLock(f)
	return false
}
