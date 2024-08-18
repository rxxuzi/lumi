package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PROJECT_DIR         string = "lumi-project"
	DEFAULT_MEDIA_COUNT int    = 20
	MIN_MEDIA_COUNT     int    = 1
	MAX_MEDIA_COUNT     int    = 10000
)

var (
	noText bool
)

type Lumi struct {
	Project    string   `json:"project"`
	Database   bool     `json:"database"`
	Tag        []string `json:"tag"`
	And        []string `json:"and"`
	Ignore     []string `json:"ignore"`
	MediaCount int      `json:"mediaCount"`
}

func LoadConfig(name string) (*Lumi, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	var config Lumi
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	replaceSpacesWithUnderscore(&config)
	validateAndAdjustMediaCount(&config)
	return &config, nil
}

func DecodeConfig(content string) (*Lumi, error) {
	var config Lumi
	decoder := json.NewDecoder(strings.NewReader(content))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	replaceSpacesWithUnderscore(&config)
	validateAndAdjustMediaCount(&config)
	return &config, nil
}

func replaceSpacesWithUnderscore(config *Lumi) {
	for i, tag := range config.Tag {
		config.Tag[i] = strings.ReplaceAll(tag, " ", "_")
	}
	for i, and := range config.And {
		config.And[i] = strings.ReplaceAll(and, " ", "_")
	}
	for i, ignore := range config.Ignore {
		config.Ignore[i] = strings.ReplaceAll(ignore, " ", "_")
	}
}

func validateAndAdjustMediaCount(config *Lumi) {
	if config.MediaCount == 0 {
		config.MediaCount = DEFAULT_MEDIA_COUNT
	} else if config.MediaCount < MIN_MEDIA_COUNT {
		config.MediaCount = MIN_MEDIA_COUNT
	} else if config.MediaCount > MAX_MEDIA_COUNT {
		config.MediaCount = MAX_MEDIA_COUNT
	}
}

func (l *Lumi) OutputDir() string {
	return filepath.Join(PROJECT_DIR, l.Project)
}

// InitialPage returns the starting page number (always 1)
func (l *Lumi) InitialPage() int {
	return 1
}

// ShouldContinue checks if more media should be downloaded based on the current count
func (l *Lumi) ShouldContinue(currentCount int) bool {
	return currentCount < l.MediaCount
}
