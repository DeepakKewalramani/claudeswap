//go:build windows

package supervisor

import (
	"os"
	"syscall"
)

// detachedProcAttr starts the child in a new process group so console signals to the parent do not reach it.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// terminateProcessGroup kills the session process. Windows has no SIGTERM equivalent for console apps.
func terminateProcessGroup(pid int) {
	if process, err := os.FindProcess(pid); err == nil {
		_ = process.Kill()
	}
}

// isProcessAlive reports whether pid refers to a running process. On Windows, FindProcess
// opens a handle and fails when the process does not exist.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = process.Release()
	return true
}
