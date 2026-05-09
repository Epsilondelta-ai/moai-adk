//go:build windows

package cli

import (
	"errors"
	"os"
)

// errLockNotSupported is returned by lockFile on Windows where flock(2) is unavailable.
// ClaimTask relies on file locking for concurrent writes; use tmux-based team workflows
// on macOS/Linux instead of running team commands on Windows.
var errLockNotSupported = errors.New("file locking not supported on Windows; use tmux-based team workflows on macOS/Linux")

// lockFile returns errLockNotSupported on Windows. ClaimTask's caller receives a
// clear error instead of silently proceeding without the lock, preventing data races.
func lockFile(_ *os.File) error {
	return errLockNotSupported
}

// unlockFile is a no-op on Windows; lockFile never succeeds so there is nothing to unlock.
func unlockFile(_ *os.File) error {
	return nil
}
