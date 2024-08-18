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

type Progress struct {
	TotalPages        int32
	CompletedPages    int32
	TotalImages       int32
	DownloadedImages  int32
	SkippedImages     int32
	CurrentFileNumber int32
	Terminated        bool
}

func Launch(config *Lumi) *Progress {
	if len(config.Tag) > 2 {
		fmt.Println("Please use 2 or fewer tags")
		return nil
	}

	fmt.Printf("Output directory: %s\n", config.OutputDir())
	fmt.Printf("Ignoring tags: %v\n", config.Ignore)
	fmt.Printf("AND tags: %v\n", config.And)

	if err := os.MkdirAll(config.OutputDir(), os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return nil
	}

	progress := &Progress{
		TotalPages: int32(config.Pages),
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for _, page := range config.PageRange() {
		if progress.Terminated {
			break
		}
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			crawlPage(config, baseURL, page, progress)
		}(page)
	}

	wg.Wait()

	if progress.Terminated {
		fmt.Println("Crawling process terminated early due to no more content")
	}

	fmt.Printf("Downloads completed. Downloaded: %d, Skipped: %d\n", progress.DownloadedImages, progress.SkippedImages)
	return progress
}

func crawlPage(config *Lumi, baseURL string, page int, progress *Progress) {
	url := fmt.Sprintf("%s?page=%d&tags=%s", baseURL, page, strings.Join(config.Tag, "+"))
	fmt.Printf("Crawling page %d: %s\n", page, url)

	r, err := raven.NewRaven(url)
	if err != nil {
		fmt.Printf("Error creating Raven for page %d: %v\n", page, err)
		return
	}

	links := r.GetURLs("post-preview-link", raven.CLASS)
	if len(links) == 0 {
		fmt.Printf("No links found on page %d. Terminating crawl.\n", page)
		progress.Terminated = true
		return
	}

	atomic.AddInt32(&progress.TotalImages, int32(len(links)))

	var wg sync.WaitGroup
	for _, link := range links {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			processLink(config, link, progress)
		}(link)
	}
	wg.Wait()

	atomic.AddInt32(&progress.CompletedPages, 1)
}

func processLink(config *Lumi, link string, progress *Progress) {
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
		atomic.AddInt32(&progress.SkippedImages, 1)
		fmt.Println("Skipped due to ignored tags")
		return
	}

	if !caption.ContainsAllAndTags(config.And) {
		atomic.AddInt32(&progress.SkippedImages, 1)
		fmt.Println("Skipped due to missing AND tags")
		return
	}

	currentFileNumber := atomic.AddInt32(&progress.CurrentFileNumber, 1)

	imageFileName := fmt.Sprintf("%s_%d.png", config.Project, currentFileNumber)
	imagePath := filepath.Join(config.OutputDir(), imageFileName)

	if err := raven.Download(imagePath, href); err != nil {
		fmt.Printf("Error downloading image: %v\n", err)
		atomic.AddInt32(&progress.SkippedImages, 1)
		return
	}

	atomic.AddInt32(&progress.DownloadedImages, 1)

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
