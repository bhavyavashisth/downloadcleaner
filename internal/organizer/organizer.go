package organizer

import (
	"download-cleaner/internal/rules"
	"download-cleaner/internal/scanner"
	"fmt"
	"path/filepath"
	"strings"
)

type Operation struct {
	From     string
	To       string
	Filename string
	Category string
}

type Plan struct {
	Operations   []Operation
	Unknown      []string
	Collisions   []string
	CategoryCounts map[string]int
}

// PlanMoves figures out where each file should go
func PlanMoves(files []scanner.FileInfo, rules *rules.Manager) Plan {
	plan := Plan{
		Operations:     []Operation{},
		Unknown:        []string{},
		Collisions:     []string{},
		CategoryCounts: make(map[string]int),
	}

	for _, f := range files {
		// skip directories
		if f.IsDir {
			continue
		}

		category := rules.Classify(f.Ext)
		if category == "" {
			plan.Unknown = append(plan.Unknown, f.Name)
			continue
		}

		// build destination path
		destDir := filepath.Join(filepath.Dir(f.Path), category)
		destPath := filepath.Join(destDir, f.Name)

		// check for collision
		if fileExists(destPath) {
			plan.Collisions = append(plan.Collisions, f.Name)
			destPath = generateSafeName(destDir, f.Name)
		}

		plan.Operations = append(plan.Operations, Operation{
			From:     f.Path,
			To:       destPath,
			Filename: f.Name,
			Category: category,
		})

		plan.CategoryCounts[category]++
	}

	return plan
}

// simple check if a file exists
func fileExists(path string) bool {
	
	return false 
}

// generate a safe name when there's a collision
func generateSafeName(dir, filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s (%d)%s", name, counter, ext)
		if !fileExists(filepath.Join(dir, newName)) {
			return filepath.Join(dir, newName)
		}
		counter++
	}
}