package transform

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// defaultRemoveTags are HTML elements removed during cleaning.
var defaultRemoveTags = []string{
	"script", "style", "noscript", "iframe", "svg",
	"nav", "footer", "header",
}

// Clean removes boilerplate HTML elements and returns sanitized HTML.
func Clean(html string, excludeTags []string, includeTags []string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove default boilerplate tags
	for _, tag := range defaultRemoveTags {
		doc.Find(tag).Remove()
	}

	// Remove user-specified exclude tags
	for _, tag := range excludeTags {
		doc.Find(tag).Remove()
	}

	// If includeTags are specified, keep only matching elements
	if len(includeTags) > 0 {
		selector := strings.Join(includeTags, ", ")
		kept := doc.Find(selector)
		body := doc.Find("body")
		body.Empty()
		kept.Each(func(_ int, s *goquery.Selection) {
			body.AppendSelection(s)
		})
	}

	// Remove comments and empty attributes
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		// Remove common tracking/ad attributes
		s.RemoveAttr("onclick")
		s.RemoveAttr("onload")
		s.RemoveAttr("data-ad")
	})

	result, err := doc.Find("body").Html()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}
