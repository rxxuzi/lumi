package core

import (
	"fmt"
	"github.com/rxxuzi/lumi/pkg/raven"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const baseURL = "https://danbooru.donmai.us/posts"

func Launch(config *Lumi) {
	if len(config.Tag) > 2 {
		fmt.Println("Please use 2 or fewer tags")
		return
	}

	fmt.Printf("Output directory: %s\n", config.OutputDir())
	fmt.Printf("Ignoring tags: %v\n", config.Ignore)
	fmt.Printf("AND tags: %v\n", config.And)

	if err := os.MkdirAll(config.OutputDir(), os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	var fileNumber int32 = 1
	var skippedCount int32 = 0

	for _, page := range config.PageRange() {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			crawlPage(config, baseURL, page, &fileNumber, &skippedCount)
		}(page)
	}

	wg.Wait()

	totalDownloaded := atomic.LoadInt32(&fileNumber) - 1
	fmt.Printf("Downloads completed. Downloaded: %d, Skipped: %d\n", totalDownloaded, atomic.LoadInt32(&skippedCount))
}

func crawlPage(config *Lumi, baseURL string, page int, fileNumber, skippedCount *int32) {
	url := fmt.Sprintf("%s?page=%d&tags=%s", baseURL, page, strings.Join(config.Tag, "+"))
	fmt.Printf("Crawling page %d: %s\n", page, url)

	r, err := raven.NewRaven(url)
	if err != nil {
		fmt.Printf("Error creating Raven for page %d: %v\n", page, err)
		return
	}

	links := r.GetURLs("post-preview-link", raven.CLASS)
	for _, link := range links {
		processLink(config, link, fileNumber, skippedCount)
	}
}

func processLink(config *Lumi, link string, fileNumber, skippedCount *int32) {
	r, err := raven.NewRaven(link)
	if err != nil {
		fmt.Printf("Error creating Raven for link %s: %v\n", link, err)
		return
	}

	// Get the image-container
	container := r.Get("image-container", raven.CLASS).First()
	if container.Length() == 0 {
		fmt.Printf("No image-container found for link %s\n", link)
		return
	}

	img := container.Find("img#image").First()
	if img.Length() == 0 {
		fmt.Printf("No image found with id 'image' for link %s\n", link)
		return
	}

	href := raven.Src(r, img)

	caption := NewCaption(r)
	if caption.ContainsIgnoredTags(config.Ignore) {
		atomic.AddInt32(skippedCount, 1)
		fmt.Println("Skipped due to ignored tags")
		return
	}

	if !caption.ContainsAllAndTags(config.And) {
		atomic.AddInt32(skippedCount, 1)
		fmt.Println("Skipped due to missing AND tags")
		return
	}

	currentFileNumber := atomic.AddInt32(fileNumber, 1)

	imageFileName := fmt.Sprintf("%s_%d.png", config.Project, currentFileNumber)
	imagePath := filepath.Join(config.OutputDir(), imageFileName)

	if err := raven.Download(imagePath, href); err != nil {
		fmt.Printf("Error downloading image: %v\n", err)
		return
	}

	if !noText {
		textFileName := fmt.Sprintf("%s_%d.txt", config.Project, currentFileNumber)
		textPath := filepath.Join(config.OutputDir(), textFileName)
		if err := saveCaption(caption.Tags, textPath); err != nil {
			fmt.Printf("Error saving caption: %v\n", err)
			return
		}

		fmt.Printf("Downloaded: %s, Caption: %s\n", imageFileName, textFileName)
	} else {
		fmt.Printf("Downloaded: %s \n", imageFileName)
	}
}

func saveCaption(tags []string, filepath string) error {
	return os.WriteFile(filepath, []byte(strings.Join(tags, ", ")), 0644)
}
