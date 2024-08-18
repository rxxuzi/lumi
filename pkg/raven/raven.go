package raven

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type By int

const (
	TAG By = iota
	CLASS
	ID
	ATTR
)

type Raven struct {
	URL            *url.URL
	HTML           string
	Document       *goquery.Document
	Origin         bool
	StatusCode     int
	ResponseHeader http.Header
}

func NewRaven(urlStr string) (*Raven, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	return &Raven{
		URL:            u,
		HTML:           string(bodyBytes),
		Document:       doc,
		Origin:         true,
		StatusCode:     resp.StatusCode,
		ResponseHeader: resp.Header,
	}, nil
}

func (r *Raven) Get(query string, by By) *goquery.Selection {
	switch by {
	case TAG:
		return r.Document.Find(query)
	case CLASS:
		return r.Document.Find("." + query)
	case ID:
		return r.Document.Find("#" + query)
	case ATTR:
		return r.Document.Find("[" + query + "]")
	default:
		return nil
	}
}

func (r *Raven) Snippet(name string, by By) (*Raven, error) {
	elements := r.Get(name, by)
	html, err := elements.Html()
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	return &Raven{
		URL:      r.URL,
		HTML:     html,
		Document: doc,
		Origin:   false,
	}, nil
}

func (r *Raven) Cut(name string, by By) {
	var selection *goquery.Selection
	switch by {
	case TAG:
		selection = r.Document.Find(name)
	case CLASS:
		selection = r.Document.Find("." + name)
	case ID:
		selection = r.Document.Find("#" + name)
	case ATTR:
		selection = r.Document.Find("[" + name + "]")
	}

	selection.Remove()

	// Update HTML after cut
	html, _ := r.Document.Html()
	r.HTML = html
}

func Src(r *Raven, s *goquery.Selection) string {
	var rawSrc string

	// src > href > srcset > data-src
	if src, exists := s.Attr("src"); exists {
		rawSrc = src
	} else if href, exists := s.Attr("href"); exists {
		rawSrc = href
	} else if srcset, exists := s.Attr("srcset"); exists {
		parts := strings.Fields(srcset)
		if len(parts) > 0 {
			rawSrc = parts[0]
		}
	} else if dataSrc, exists := s.Attr("data-src"); exists {
		rawSrc = dataSrc
	}

	if rawSrc == "" {
		return ""
	}

	absURL, err := r.URL.Parse(rawSrc)
	if err != nil {
		return ""
	}

	return absURL.String()
}

func (r *Raven) GetURLs(query string, by By) []string {
	var urls []string
	r.Get(query, by).Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			fullURL, err := r.URL.Parse(href)
			if err == nil {
				urls = append(urls, fullURL.String())
			}
		}
	})
	return urls
}

func (r *Raven) DebugInfo() string {
	return fmt.Sprintf("URL: %s\nStatus Code: %d\nContent Length: %s\n",
		r.URL.String(), r.StatusCode, r.ResponseHeader.Get("Content-Length"))
}
