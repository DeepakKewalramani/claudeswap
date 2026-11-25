package claude

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DeepakKewalramani/claudeswap/internal/config"
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
					// Merge existing files from profileDir into sharedTarget before symlinking.
					// Only delete the profile copy once every file is safely in the shared pool.
					if err := mergeDirectories(profileDir, sharedTarget, conflictSuffix(profileConfigDir)); err != nil {
						return fmt.Errorf("failed to merge %s into shared context (profile data left untouched): %w", dirName, err)
					}
					if err := os.RemoveAll(profileDir); err != nil {
						return fmt.Errorf("failed to replace %s with shared link: %w", dirName, err)
					}
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
			// Preserve the profile's own history by appending it to the shared history first.
			if err := appendFile(profileHistory, sharedHistory); err != nil {
				return fmt.Errorf("failed to merge history.jsonl into shared context (profile history left untouched): %w", err)
			}
			if err := os.Remove(profileHistory); err != nil {
				return fmt.Errorf("failed to replace history.jsonl with shared link: %w", err)
			}
		}
		if err := os.Symlink(sharedHistory, profileHistory); err != nil {
			return fmt.Errorf("failed to symlink shared history.jsonl: %w", err)
		}
	} else {
		fi, err := os.Lstat(profileHistory)
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(profileHistory)
		}
	}

	return nil
}

// conflictSuffix names the copy kept when a profile file collides with a different shared file.
func conflictSuffix(profileConfigDir string) string {
	return ".conflict-" + filepath.Base(profileConfigDir)
}

// mergeDirectories copies every file from src into dst. Files missing from dst are copied as-is;
// files that exist in dst with different content are kept alongside it with the conflict suffix,
// so nothing from src is lost. Any error aborts the merge and is returned to the caller.
func mergeDirectories(src, dst, suffix string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0700); err != nil {
				return err
			}
			if err := mergeDirectories(srcPath, dstPath, suffix); err != nil {
				return err
			}
			continue
		}

		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			if err := copyFileSimple(srcPath, dstPath); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}

		same, err := sameContent(srcPath, dstPath)
		if err != nil {
			return err
		}
		if !same {
			if err := copyFileSimple(srcPath, dstPath+suffix); err != nil {
				return err
			}
		}
	}
	return nil
}

func sameContent(a, b string) (bool, error) {
	da, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(da, db), nil
}

// appendFile appends src to dst, adding a trailing newline when src does not end with one.
func appendFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
