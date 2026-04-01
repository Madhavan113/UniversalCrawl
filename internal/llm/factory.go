package llm

import "fmt"

// NewProviderFromRequest creates an LLM provider from per-request parameters.
func NewProviderFromRequest(providerName, apiKey, ollamaBaseURL string) (Provider, error) {
	switch providerName {
	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("llmApiKey is required for anthropic provider")
		}
		return NewAnthropicProvider(apiKey), nil
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("llmApiKey is required for openai provider")
		}
		return NewOpenAIProvider(apiKey), nil
	case "ollama":
		base := ollamaBaseURL
		if base == "" {
			base = "http://localhost:11434"
		}
		return NewOllamaProvider(base), nil
	default:
		return nil, fmt.Errorf("unsupported llmProvider %q: use anthropic, openai, or ollama", providerName)
	}
}
