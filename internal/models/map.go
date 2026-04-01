package models

// MapRequest defines parameters for URL discovery on a site.
type MapRequest struct {
	URL               string `json:"url"`
	Search            string `json:"search"`
	IncludeSubdomains bool   `json:"includeSubdomains"`
	Limit             int    `json:"limit"`
	SitemapOnly       bool   `json:"sitemapOnly"`
}
