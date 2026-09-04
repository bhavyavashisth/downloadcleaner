// cmd/dc/main.go
package main

import (
	"download-cleaner/internal/config"
	"download-cleaner/internal/scanner"
	"fmt"
	"log"
)

func main() {
	// find the downloads folder
	downloads, err := config.GetDownloadsDir()
	if err != nil {
		log.Fatalf("can't find downloads: %v", err)
	}

	fmt.Printf("📂 %s\n\n", downloads)

	// scan for files
	files, err := scanner.Scan(downloads)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}

	fmt.Printf("found %d files\n", len(files))

	// show a few examples
	if len(files) > 0 {
		fmt.Println("\nfirst few files:")
		for i := 0; i < len(files) && i < 5; i++ {
			f := files[i]
			fmt.Printf("  %s (%s, %d bytes)\n", f.Name, f.Ext, f.Size)
		}
		if len(files) > 5 {
			fmt.Printf("  ... and %d more\n", len(files)-5)
		}
	}
}