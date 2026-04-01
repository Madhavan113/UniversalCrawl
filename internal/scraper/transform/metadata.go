package transform

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/madhavanp/universalcrawl/internal/models"
)

// ExtractMetadata parses HTML to extract page title, description, OG tags, and language.
func ExtractMetadata(html string, statusCode int) models.PageMetadata {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return models.PageMetadata{StatusCode: statusCode}
	}

	meta := models.PageMetadata{
		StatusCode: statusCode,
	}

	meta.Title = doc.Find("title").First().Text()

	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		switch {
		case name == "description":
			meta.Description = content
		case property == "og:image":
			meta.OGImage = content
		case property == "og:title" && meta.Title == "":
			meta.Title = content
		case property == "og:description" && meta.Description == "":
			meta.Description = content
		}
	})

	if lang, exists := doc.Find("html").Attr("lang"); exists {
		meta.Language = lang
	}

	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		if rel, _ := s.Attr("rel"); rel == "canonical" {
			meta.Canonical, _ = s.Attr("href")
		}
	})

	return meta
}
