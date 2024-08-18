package core

import (
	"fmt"
	"github.com/rxxuzi/lumi/pkg/raven"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL    = "https://danbooru.donmai.us/posts"
	maxPerPage = 20
	minDelay   = 3 * time.Second
	maxDelay   = 10 * time.Second
	maxRetries = 3
)

type Progress struct {
	TotalPages        int32
	CompletedPages    int32
	TotalMedia        int32
	DownloadedImages  int32
	SkippedImages     int32
	CurrentFileNumber int32
	RequestedMedia    int32
	Terminated        bool
}

func Launch(config *Lumi) *Progress {
	if len(config.Tag) > 2 {
		fmt.Println("Please use 2 or fewer tags")
		return nil
	}

	fmt.Printf("Output directory: %s\n", config.OutputDir())
	fmt.Printf("Tag: %s\n", config.Tag)
	fmt.Printf("Ignoring tags: %v\n", config.Ignore)
	fmt.Printf("AND tags: %v\n", config.And)
	fmt.Printf("Request Count: %v\n", config.MediaCount)

	if err := os.MkdirAll(config.OutputDir(), os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return nil
	}

	progress := &Progress{
		TotalMedia:     int32(config.MediaCount),
		RequestedMedia: int32(config.MediaCount),
	}

	fullPages := config.MediaCount / maxPerPage

	if fullPages == 0 {
		// If media count is less than maxPerPage, do sequential processing
		crawlPageSequential(config, baseURL, 1, progress, config.MediaCount)
	} else {
		// Parallel crawling for full pages
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 10)

		for page := 1; page <= fullPages; page++ {
			wg.Add(1)
			go func(page int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				crawlPage(config, baseURL, page, progress)
			}(page)
		}
		wg.Wait()

		// Check if we need to continue crawling
		if int(progress.DownloadedImages) < config.MediaCount {
			continueSequentialCrawl(config, baseURL, fullPages+1, progress)
		}
	}

	if progress.Terminated {
		fmt.Println("Crawling process terminated due to no more content")
	}

	fmt.Printf("Downloads completed. Downloaded: %d, Skipped: %d\n", progress.DownloadedImages, progress.SkippedImages)
	return progress
}

func continueSequentialCrawl(config *Lumi, baseURL string, startPage int, progress *Progress) {
	currentPage := startPage
	for int(progress.DownloadedImages) < config.MediaCount && !progress.Terminated {
		remainingMedia := config.MediaCount - int(progress.DownloadedImages)
		crawlPageSequential(config, baseURL, currentPage, progress, remainingMedia)
		currentPage++
	}
}

func crawlPageSequential(config *Lumi, baseURL string, page int, progress *Progress, limit int) {
	url := fmt.Sprintf("%s?page=%d&tags=%s", baseURL, page, strings.Join(config.Tag, "+"))
	fmt.Printf("Crawling page %d sequentially: %s\n", page, url)

	r, err := fetchWithRetry(url)
	if err != nil {
		fmt.Printf("Error creating Raven for page %d after retries: %v\n", page, err)
		return
	}

	links := r.GetURLs("post-preview-link", raven.CLASS)
	if len(links) == 0 {
		fmt.Printf("No links found on page %d. Terminating crawl.\n", page)
		progress.Terminated = true
		return
	}

	atomic.AddInt32(&progress.TotalPages, 1)

	for _, link := range links {
		if int(progress.DownloadedImages) >= config.MediaCount {
			return
		}
		if !config.ShouldContinue(int(progress.DownloadedImages)) {
			return
		}
		processLink(config, link, progress)
		if int(progress.DownloadedImages) >= int(progress.TotalPages)*maxPerPage {
			return
		}
	}

	atomic.AddInt32(&progress.CompletedPages, 1)
}

func crawlPage(config *Lumi, baseURL string, page int, progress *Progress) {
	url := fmt.Sprintf("%s?page=%d&tags=%s", baseURL, page, strings.Join(config.Tag, "+"))
	fmt.Printf("Crawling page %d: %s\n", page, url)

	r, err := fetchWithRetry(url)
	if err != nil {
		fmt.Printf("Error creating Raven for page %d after retries: %v\n", page, err)
		return
	}

	links := r.GetURLs("post-preview-link", raven.CLASS)
	if len(links) == 0 {
		fmt.Printf("No links found on page %d. Terminating crawl.\n", page)
		progress.Terminated = true
		return
	}

	atomic.AddInt32(&progress.TotalPages, 1)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)
	for _, link := range links {
		if !config.ShouldContinue(int(progress.DownloadedImages)) {
			return
		}
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			processLink(config, link, progress)
		}(link)
	}
	wg.Wait()

	atomic.AddInt32(&progress.CompletedPages, 1)
}

func fetchWithRetry(url string) (*raven.Raven, error) {
	var r *raven.Raven
	var err error

	for i := 0; i < maxRetries; i++ {
		r, err = raven.NewRaven(url)
		if err == nil && r.StatusCode != 429 {
			return r, nil
		}

		if r != nil && r.StatusCode == 429 {
			fmt.Printf("Rate limit exceeded for %s. Retrying after delay...\n", url)
		} else {
			fmt.Printf("Error fetching %s: %v. Retrying...\n", url, err)
		}

		time.Sleep(getRandomDelay())
	}

	return nil, fmt.Errorf("failed to fetch %s after %d retries", url, maxRetries)
}

func getRandomDelay() time.Duration {
	return minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
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
		fmt.Printf("Full HTML:\n%s\n", r.HTML)
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
