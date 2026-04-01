package models

import "encoding/json"

// ExtractRequest defines parameters for LLM-powered structured extraction.
type ExtractRequest struct {
	URLs   []string         `json:"urls"`
	Prompt string           `json:"prompt"`
	Schema *json.RawMessage `json:"schema"`
}
