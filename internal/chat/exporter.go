package chat

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExportArchive bundles the sessions and projects directories from srcDir into a tar.gz archive.
func ExportArchive(srcDir, destArchive string) error {
	f, err := os.Create(destArchive)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Only include chat-relevant directories: sessions and projects
	includeDirs := []string{"sessions", "projects", "plans"}

	for _, sub := range includeDirs {
		subPath := filepath.Join(srcDir, sub)
		info, err := os.Stat(subPath)
		if err != nil || !info.IsDir() {
			continue
		}

		err = filepath.Walk(subPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Calculate relative path inside archive
			rel, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(rel)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(tw, file)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error archiving %s: %w", sub, err)
		}
	}

	return nil
}

// SessionSummary represents high-level info about a detected conversation session.
type SessionSummary struct {
	ID        string
	Project   string
	FilePath  string
	SizeBytes int64
	ModTime   string
}

// FindSessions discovers all session .jsonl files in the given directory.
func FindSessions(baseDir string) ([]SessionSummary, error) {
	var sessions []SessionSummary
	projectsDir := filepath.Join(baseDir, "projects")

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return sessions, nil
	}

	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") {
			rel, _ := filepath.Rel(projectsDir, path)
			parts := strings.Split(rel, string(filepath.Separator))
			projName := "unknown"
			if len(parts) > 1 {
				projName = parts[0]
			}
			sessionID := strings.TrimSuffix(info.Name(), ".jsonl")

			sessions = append(sessions, SessionSummary{
				ID:        sessionID,
				Project:   projName,
				FilePath:  path,
				SizeBytes: info.Size(),
				ModTime:   info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
		return nil
	})

	return sessions, err
}
