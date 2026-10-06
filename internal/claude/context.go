package claude

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/claudeswap/claudeswap/internal/config"
)

var sharedDirectories = []string{
	"sessions",
	"projects",
	"file-history",
	"plans",
}

// SyncSharedContext links or unlinks shared session and memory directories between
// ~/.claudeswap/shared and the profile's isolated configuration directory.
// Crucially, authentication and credential files are NEVER touched or shared.
func SyncSharedContext(paths *config.Paths, profileConfigDir string, enable bool) error {
	if profileConfigDir == "" {
		return nil
	}

	sharedBase := paths.SharedDir()
	if err := os.MkdirAll(sharedBase, 0700); err != nil {
		return fmt.Errorf("failed to create shared directory: %w", err)
	}

	for _, dirName := range sharedDirectories {
		sharedTarget := filepath.Join(sharedBase, dirName)
		profileDir := filepath.Join(profileConfigDir, dirName)

		if enable {
			// Ensure shared directory exists
			if err := os.MkdirAll(sharedTarget, 0700); err != nil {
				return err
			}

			// Check current state of profileDir
			fi, err := os.Lstat(profileDir)
			if err == nil {
				// If it's already a symlink, check where it points
				if fi.Mode()&os.ModeSymlink != 0 {
					dest, err := os.Readlink(profileDir)
					if err == nil && filepath.Clean(dest) == filepath.Clean(sharedTarget) {
						continue // Already pointing to the correct shared location
					}
					_ = os.Remove(profileDir)
				} else if fi.IsDir() {
					// Merge existing files from profileDir to sharedTarget before symlinking
					_ = mergeDirectories(profileDir, sharedTarget)
					_ = os.RemoveAll(profileDir)
				}
			}

			// Create symlink from shared target to profile directory
			if err := os.Symlink(sharedTarget, profileDir); err != nil {
				// Non-fatal if symlink fails on unsupported platforms
				return fmt.Errorf("failed to symlink shared %s: %w", dirName, err)
			}
		} else {
			// If disabling, detach symlink and create isolated local directory
			fi, err := os.Lstat(profileDir)
			if err == nil && fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(profileDir)
				_ = os.MkdirAll(profileDir, 0700)
			}
		}
	}

	// Share command history (history.jsonl)
	sharedHistory := filepath.Join(sharedBase, "history.jsonl")
	profileHistory := filepath.Join(profileConfigDir, "history.jsonl")

	if enable {
		if _, err := os.Stat(sharedHistory); os.IsNotExist(err) {
			_ = os.WriteFile(sharedHistory, []byte{}, 0600)
		}
		fi, err := os.Lstat(profileHistory)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			_ = os.Remove(profileHistory)
		}
		_ = os.Symlink(sharedHistory, profileHistory)
	} else {
		fi, err := os.Lstat(profileHistory)
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(profileHistory)
		}
	}

	return nil
}

func mergeDirectories(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			_ = os.MkdirAll(dstPath, 0700)
			_ = mergeDirectories(srcPath, dstPath)
		} else {
			// Only copy if not present in dst
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				_ = copyFileSimple(srcPath, dstPath)
			}
		}
	}
	return nil
}

func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
