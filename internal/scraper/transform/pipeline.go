package transform

import (
	"encoding/base64"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper/engines"
)

// Options controls which transforms to apply.
type Options struct {
	Formats         []string
	OnlyMainContent bool
	ExcludeTags     []string
	IncludeTags     []string
}

// Run executes the transform pipeline on raw scrape output and returns a ScrapeResult.
func Run(raw *engines.RawResult, opts Options) (*models.ScrapeResult, error) {
	result := &models.ScrapeResult{
		URL: raw.URL,
	}

	formats := make(map[string]bool, len(opts.Formats))
	for _, f := range opts.Formats {
		formats[f] = true
	}

	// Always extract metadata
	result.Metadata = ExtractMetadata(raw.HTML, raw.StatusCode)

	// Clean HTML
	cleaned, err := Clean(raw.HTML, opts.ExcludeTags, opts.IncludeTags)
	if err != nil {
		return nil, err
	}

	// Optionally extract main content
	contentHTML := cleaned
	if opts.OnlyMainContent {
		main, err := ExtractMainContent(raw.HTML, raw.URL)
		if err == nil && main != "" {
			contentHTML = main
		}
	}

	// Raw HTML format
	if formats["rawHtml"] {
		result.RawHTML = raw.HTML
	}

	// Cleaned HTML format
	if formats["html"] {
		result.HTML = contentHTML
	}

	// Markdown format
	if formats["markdown"] {
		md, err := ToMarkdown(contentHTML)
		if err != nil {
			return nil, err
		}
		result.Markdown = md
	}

	// Links format
	if formats["links"] {
		links, err := ExtractLinks(raw.HTML, raw.URL)
		if err != nil {
			return nil, err
		}
		result.Links = links
	}

	// Screenshot format
	if formats["screenshot"] && len(raw.Screenshot) > 0 {
		result.Screenshot = "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw.Screenshot)
	}

	return result, nil
}
