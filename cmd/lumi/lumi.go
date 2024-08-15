package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Lumi struct {
	Project  string   `json:"project"`
	Database bool     `json:"database"`
	Tag      []string `json:"tag"`
	And      []string `json:"and"`
	Ignore   []string `json:"ignore"`
	Pages    int      `json:"pages"`
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
	return &config, nil
}

func DecodeConfig(content string) (*Lumi, error) {
	var config Lumi
	decoder := json.NewDecoder(strings.NewReader(content))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	replaceSpacesWithUnderscore(&config)
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

func (l *Lumi) OutputDir() string {
	return filepath.Join(PROJECT_DIR, l.Project)
}

func (l *Lumi) PageRange() []int {
	pages := make([]int, l.Pages)
	for i := range pages {
		pages[i] = i + 1
	}
	return pages
}
