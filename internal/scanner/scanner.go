package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path    string
	Name    string
	Ext     string
	Size    int64
	ModTime int64
	IsDir   bool
}

// Scan reads files in a directory, non-recursive
func Scan(dir string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []FileInfo

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // skip files we can't read
		}

		// get extension 
		ext := strings.ToLower(filepath.Ext(entry.Name()))

		files = append(files, FileInfo{
			Path:    filepath.Join(dir, entry.Name()),
			Name:    entry.Name(),
			Ext:     ext,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			IsDir:   entry.IsDir(),
		})
	}

	return files, nil
}

// GetStats returns basic stats about scanned files
func GetStats(files []FileInfo) (total int, dirs int, filesCount int) {
	total = len(files)
	for _, f := range files {
		if f.IsDir {
			dirs++
		} else {
			filesCount++
		}
	}
	return
}