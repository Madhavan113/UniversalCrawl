package llm

import "context"

// Provider defines the interface for LLM-powered extraction.
type Provider interface {
	// Name returns the provider identifier.
	Name() string
	// Complete sends a prompt to the LLM and returns the response text.
	Complete(ctx context.Context, req CompletionRequest) (string, error)
}

// CompletionRequest holds parameters for an LLM completion.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	JSONMode     bool
	MaxTokens    int
}
