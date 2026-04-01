package transform

import "testing"

func TestExtractLinks_AbsoluteURLs(t *testing.T) {
	html := `<html><body>
		<a href="https://example.com/about">About</a>
		<a href="https://example.com/contact">Contact</a>
	</body></html>`
	links, err := ExtractLinks(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestExtractLinks_RelativeURLs(t *testing.T) {
	html := `<html><body><a href="/about">About</a></body></html>`
	links, err := ExtractLinks(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0] != "https://example.com/about" {
		t.Errorf("expected resolved URL, got %q", links[0])
	}
}

func TestExtractLinks_SkipsFragmentsAndJavascript(t *testing.T) {
	html := `<html><body>
		<a href="#section">Anchor</a>
		<a href="javascript:void(0)">JS</a>
		<a href="/real">Real</a>
	</body></html>`
	links, err := ExtractLinks(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link (fragment and JS skipped), got %d", len(links))
	}
}

func TestExtractLinks_Deduplicates(t *testing.T) {
	html := `<html><body>
		<a href="/page">Link 1</a>
		<a href="/page">Link 2</a>
	</body></html>`
	links, err := ExtractLinks(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 deduplicated link, got %d", len(links))
	}
}
