package models

// SearchRequest defines parameters for web search + scrape.
type SearchRequest struct {
	Query           string   `json:"query"`
	Limit           int      `json:"limit"`
	Formats         []string `json:"formats"`
	OnlyMainContent bool     `json:"onlyMainContent"`
	Lang            string   `json:"lang"`
	Country         string   `json:"country"`
}
