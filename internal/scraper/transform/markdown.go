package transform

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

// ToMarkdown converts HTML to clean Markdown.
func ToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(markdown), nil
}
