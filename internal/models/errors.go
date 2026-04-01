package models

import "fmt"

// ScrapeError represents a typed error from the scrape pipeline.
type ScrapeError struct {
	URL       string
	Stage     string
	Cause     error
	Retryable bool
}

func (e *ScrapeError) Error() string {
	return fmt.Sprintf("scrape error [%s] url=%s: %v", e.Stage, e.URL, e.Cause)
}

func (e *ScrapeError) Unwrap() error { return e.Cause }

// StorageError represents a typed error from the storage layer.
type StorageError struct {
	Op    string
	Key   string
	Cause error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error [%s] key=%s: %v", e.Op, e.Key, e.Cause)
}

func (e *StorageError) Unwrap() error { return e.Cause }

// ErrNotFound indicates a requested resource does not exist.
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}
