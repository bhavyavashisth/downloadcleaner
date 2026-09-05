package organizer

import (
	"downloadcleaner/internal/rules"
	"downloadcleaner/internal/scanner"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Operation struct {
	From     string
	To       string
	Filename string
	Category string
	Success  bool
	Error    string
}

type Plan struct {
	Operations    []Operation
	Unknown       []string
	Collisions    []string
	CategoryCounts map[string]int
	TotalMoved    int
	TotalErrors   int
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

		// skip hidden files (optional, can be removed)
		if strings.HasPrefix(f.Name, ".") {
			continue
		}

		category := rules.Classify(f.Ext)
		if category == "" {
			plan.Unknown = append(plan.Unknown, f.Name)
			continue
		}

		// build destination
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

// Execute moves files according to the plan
func Execute(plan *Plan, dryRun bool) ([]Operation, error) {
	if dryRun {
		// just mark everything as "would move"
		for i := range plan.Operations {
			plan.Operations[i].Success = true
		}
		return plan.Operations, nil
	}

	var results []Operation
	successCount := 0
	errorCount := 0

	for _, op := range plan.Operations {
		// create destination directory if needed
		destDir := filepath.Dir(op.To)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			op.Success = false
			op.Error = fmt.Sprintf("can't create directory: %v", err)
			errorCount++
			results = append(results, op)
			continue
		}

		// move the file
		if err := os.Rename(op.From, op.To); err != nil {
			
			if err := copyFile(op.From, op.To); err != nil {
				op.Success = false
				op.Error = fmt.Sprintf("move failed: %v", err)
				errorCount++
				results = append(results, op)
				continue
			}
			// copy succeeded, delete original
			if err := os.Remove(op.From); err != nil {
				// log but don't fail - file was copied
				op.Error = "file copied but original couldn't be removed"
			}
		}

		op.Success = true
		successCount++
		results = append(results, op)
	}

	plan.TotalMoved = successCount
	plan.TotalErrors = errorCount

	return results, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// generate a safe name when there's a collision
func generateSafeName(dir, filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s (%d)%s", name, counter, ext)
		newPath := filepath.Join(dir, newName)
		if !fileExists(newPath) {
			return newPath
		}
		counter++
	}
}

// copyFile copies a file (fallback when rename fails)
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		return err
	}

	// preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// GenerateReport creates a summary string
func GenerateReport(plan Plan, dryRun bool, startTime time.Time) string {
	var b strings.Builder

	if dryRun {
		b.WriteString("DRY RUN — NO FILES WERE MOVED\n\n")
	} else {
		b.WriteString("CLEANUP COMPLETE\n\n")
	}

	b.WriteString(fmt.Sprintf("⏱ %s\n\n", time.Since(startTime).Round(time.Millisecond)))

	// category breakdown
	for cat, count := range plan.CategoryCounts {
		b.WriteString(fmt.Sprintf("  %-12s %d\n", cat, count))
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-12s %d\n", "Total moved", len(plan.Operations)))

	if len(plan.Unknown) > 0 {
		b.WriteString(fmt.Sprintf("\n⚠  %d files left untouched (unknown types)\n", len(plan.Unknown)))
	}

	if len(plan.Collisions) > 0 {
		b.WriteString(fmt.Sprintf("⚠ %d filename conflicts resolved\n", len(plan.Collisions)))
	}

	if plan.TotalErrors > 0 {
		b.WriteString(fmt.Sprintf("\n╳ %d errors occurred\n", plan.TotalErrors))
	}

	if dryRun {
		b.WriteString("\nNo changes were made.")
	}

	return b.String()
}