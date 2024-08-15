package raven

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Download downloads a file from the given URL and saves it to the specified path.
// If the path is a directory, it extracts the filename from the URL.
// It returns an error if the download or save operation fails.
func Download(path string, url string) error {
	// Check if the path is a directory
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// Extract filename from URL
		filename := extractFilename(url)
		if filename == "" {
			return fmt.Errorf("could not extract filename from URL: %s", url)
		}
		path = filepath.Join(path, filename)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to GET from %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}
	return nil
}

// extractFilename extracts the filename from the URL
func extractFilename(url string) string {
	// Split the URL by '/'
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}

	// Get the last part
	filename := parts[len(parts)-1]
	filename = strings.Split(filename, "?")[0]
	if filename == "" {
		filename = "file"
	}

	return filename
}
