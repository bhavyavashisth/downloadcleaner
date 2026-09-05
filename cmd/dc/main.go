package main

import (
	"downloadcleaner/internal/config"
	"downloadcleaner/internal/organizer"
	"downloadcleaner/internal/rules"
	"downloadcleaner/internal/scanner"
	"downloadcleaner/internal/tui"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// command line flags
	dryRun := flag.Bool("dry-run", false, "show what would be moved without actually moving")
	clean := flag.Bool("clean", false, "run cleanup immediately without TUI")
	flag.Parse()

	// find downloads
	downloads, err := config.GetDownloadsDir()
	if err != nil {
		log.Fatalf("can't find downloads: %v", err)
	}

	// load rules
	ruleManager := rules.NewManager()
	if err := ruleManager.Load(); err != nil {
		fmt.Printf("warning: couldn't load rules: %v\n", err)
	}

	// scan files
	files, err := scanner.Scan(downloads)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}

	// if clean flag is set, run non-interactive
	if *clean {
		runClean(files, ruleManager, *dryRun)
		return
	}

	// otherwise start the TUI
	app := tui.NewApp(downloads, files, ruleManager)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runClean(files []scanner.FileInfo, rules *rules.Manager, dryRun bool) {
	plan := organizer.PlanMoves(files, rules)
	if len(plan.Operations) == 0 {
		fmt.Println("No files to organize.")
		return
	}

	// show plan
	if dryRun {
		fmt.Println("DRY RUN — NO FILES WILL BE MOVED\n")
	}

	for _, op := range plan.Operations {
		fmt.Printf("%s → %s/\n", op.Filename, op.Category)
	}

	fmt.Printf("\n%d files will be organized\n", len(plan.Operations))

	if dryRun {
		fmt.Println("\nNo changes were made.")
		return
	}

	// ask for confirmation
	fmt.Print("\nProceed? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return
	}

	// execute
	startTime := time.Now()
	organizer.Execute(&plan, false)
	report := organizer.GenerateReport(plan, false, startTime)
	fmt.Println("\n" + report)
}