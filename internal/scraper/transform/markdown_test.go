package transform

import (
	"strings"
	"testing"
)

func TestToMarkdown_BasicHTML(t *testing.T) {
	html := `<h1>Title</h1><p>Paragraph text.</p>`
	got, err := ToMarkdown(html)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "# Title") {
		t.Errorf("expected markdown heading, got: %s", got)
	}
	if !strings.Contains(got, "Paragraph text.") {
		t.Error("paragraph content missing")
	}
}

func TestToMarkdown_Links(t *testing.T) {
	html := `<p>Visit <a href="https://example.com">Example</a></p>`
	got, err := ToMarkdown(html)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[Example](https://example.com)") {
		t.Errorf("expected markdown link, got: %s", got)
	}
}

func TestToMarkdown_Lists(t *testing.T) {
	html := `<ul><li>One</li><li>Two</li><li>Three</li></ul>`
	got, err := ToMarkdown(html)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "One") || !strings.Contains(got, "Two") {
		t.Errorf("list items missing, got: %s", got)
	}
}

func TestToMarkdown_EmptyInput(t *testing.T) {
	got, err := ToMarkdown("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty output, got: %s", got)
	}
}
