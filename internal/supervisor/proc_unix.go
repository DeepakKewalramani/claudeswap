//go:build !windows

package supervisor

import (
	"os"
	"syscall"
	"time"
)

// detachedProcAttr puts the child in its own process group so it outlives the parent.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcessGroup sends SIGTERM to the session's process group, then SIGKILL if it is still alive.
func terminateProcessGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	time.Sleep(50 * time.Millisecond)
	if isProcessAlive(pid) {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 verifies existence without killing
	return process.Signal(syscall.Signal(0)) == nil
}
