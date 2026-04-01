package transform

import "testing"

func TestExtractMetadata_Title(t *testing.T) {
	html := `<html><head><title>My Page</title></head><body></body></html>`
	meta := ExtractMetadata(html, 200)
	if meta.Title != "My Page" {
		t.Errorf("expected title 'My Page', got %q", meta.Title)
	}
	if meta.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", meta.StatusCode)
	}
}

func TestExtractMetadata_OGTags(t *testing.T) {
	html := `<html><head>
		<title>Title</title>
		<meta name="description" content="A description">
		<meta property="og:image" content="https://example.com/img.png">
	</head><body></body></html>`
	meta := ExtractMetadata(html, 200)
	if meta.Description != "A description" {
		t.Errorf("expected description, got %q", meta.Description)
	}
	if meta.OGImage != "https://example.com/img.png" {
		t.Errorf("expected og:image, got %q", meta.OGImage)
	}
}

func TestExtractMetadata_Language(t *testing.T) {
	html := `<html lang="en"><head><title>T</title></head><body></body></html>`
	meta := ExtractMetadata(html, 200)
	if meta.Language != "en" {
		t.Errorf("expected language 'en', got %q", meta.Language)
	}
}

func TestExtractMetadata_Canonical(t *testing.T) {
	html := `<html><head><link rel="canonical" href="https://example.com/page"></head><body></body></html>`
	meta := ExtractMetadata(html, 200)
	if meta.Canonical != "https://example.com/page" {
		t.Errorf("expected canonical, got %q", meta.Canonical)
	}
}
