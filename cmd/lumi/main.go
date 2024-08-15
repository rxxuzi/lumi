package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	PROJECT_DIR string = "lumi-project"
)

var (
	clean  string
	noText bool
)

func main() {
	flag.StringVar(&clean, "clean", "", "Clean project")
	flag.BoolVar(&noText, "no-text", false, "Not Generate Text File")
	flag.Parse()
	if clean != "" {
		dirToDelete := filepath.Join(PROJECT_DIR, clean)
		err := os.RemoveAll(dirToDelete)
		if err != nil {
			fmt.Printf("Failed to delete directory: %v\n", err)
			return
		}

		fmt.Printf("Successfully deleted directory: %s\n", dirToDelete)
		return
	}

	path := "lumi.json"
	config, err := LoadConfig(path)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	Launch(config)
}
