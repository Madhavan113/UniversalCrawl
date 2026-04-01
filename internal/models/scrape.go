package models

// ScrapeRequest defines parameters for a single-URL scrape operation.
type ScrapeRequest struct {
	URL             string            `json:"url"`
	Formats         []string          `json:"formats"`
	OnlyMainContent bool              `json:"onlyMainContent"`
	WaitFor         int               `json:"waitFor"`
	Timeout         int               `json:"timeout"`
	Headers         map[string]string `json:"headers"`
	Actions         []BrowserAction   `json:"actions"`
	Mobile          bool              `json:"mobile"`
	IncludeTags     []string          `json:"includeTags"`
	ExcludeTags     []string          `json:"excludeTags"`
}

// BrowserAction describes a single browser interaction step.
type BrowserAction struct {
	Type         string `json:"type"`
	Selector     string `json:"selector"`
	Text         string `json:"text"`
	Milliseconds int    `json:"milliseconds"`
	Direction    string `json:"direction"`
	Amount       int    `json:"amount"`
}

// ScrapeResult holds the output of a scrape operation.
type ScrapeResult struct {
	URL        string       `json:"url"`
	Markdown   string       `json:"markdown,omitempty"`
	HTML       string       `json:"html,omitempty"`
	RawHTML    string       `json:"rawHtml,omitempty"`
	Screenshot string       `json:"screenshot,omitempty"`
	Links      []string     `json:"links,omitempty"`
	Metadata   PageMetadata `json:"metadata"`
}

// PageMetadata holds extracted page metadata.
type PageMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Language    string `json:"language"`
	OGImage     string `json:"ogImage,omitempty"`
	Canonical   string `json:"canonical,omitempty"`
	StatusCode  int    `json:"statusCode"`
}
