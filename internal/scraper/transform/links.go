package transform

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractLinks collects all href links from HTML, resolved against the base URL.
func ExtractLinks(html string, baseURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		base = &url.URL{}
	}

	seen := make(map[string]struct{})
	var links []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip fragment-only and javascript links
		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		resolved, err := base.Parse(href)
		if err != nil {
			return
		}

		// Remove fragment
		resolved.Fragment = ""
		link := resolved.String()

		if _, ok := seen[link]; !ok {
			seen[link] = struct{}{}
			links = append(links, link)
		}
	})

	return links, nil
}
