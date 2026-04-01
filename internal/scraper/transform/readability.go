package transform

import (
	"net/url"
	"strings"

	readability "github.com/go-shiori/go-readability"
)

// ExtractMainContent uses readability to extract the main content from HTML.
func ExtractMainContent(html string, pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		u = &url.URL{}
	}

	article, err := readability.FromReader(strings.NewReader(html), u)
	if err != nil {
		// Fall back to raw HTML if readability fails
		return html, nil
	}

	if article.Content == "" {
		return html, nil
	}

	return article.Content, nil
}
