package core

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/rxxuzi/lumi/pkg/raven"
	"strings"
)

type Caption struct {
	Raven *raven.Raven
	Tags  []string
}

func NewCaption(rvn *raven.Raven) *Caption {
	return &Caption{
		Raven: rvn,
		Tags:  getTags(rvn),
	}
}

func getTags(rvn *raven.Raven) []string {
	var tags []string
	rvn.Get("search-tag", raven.CLASS).Each(func(i int, s *goquery.Selection) {
		tag := strings.TrimSpace(s.Text())
		tag = strings.ReplaceAll(tag, " ", "_")
		tags = append(tags, tag)
	})
	return tags
}

func (c *Caption) ContainsIgnoredTags(ignoreTags []string) bool {
	for _, tag := range c.Tags {
		for _, ignoreTag := range ignoreTags {
			if tag == ignoreTag {
				return true
			}
		}
	}
	return false
}

func (c *Caption) ContainsAllAndTags(andTags []string) bool {
	if len(andTags) == 0 {
		return true // If there are no AND tags, consider it a match
	}
	tagSet := make(map[string]bool)
	for _, tag := range c.Tags {
		tagSet[tag] = true
	}
	for _, andTag := range andTags {
		if !tagSet[andTag] {
			return false
		}
	}
	return true
}
