package transform

import (
	"strings"
	"testing"
)

func TestExtractMainContent_ExtractsArticle(t *testing.T) {
	html := `<html><head><title>Test</title></head><body>
		<nav><a href="/">Home</a></nav>
		<article>
			<h1>Article Title</h1>
			<p>This is the main content of the article. It contains enough text to be recognized as the main content by the readability algorithm. The article discusses important topics that readers care about.</p>
			<p>Another paragraph with more substantial content to help the readability algorithm identify this as the main content area of the page.</p>
		</article>
		<aside>Related links</aside>
	</body></html>`

	got, err := ExtractMainContent(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "main content") {
		t.Error("main content not extracted")
	}
}

func TestExtractMainContent_FallsBackOnFailure(t *testing.T) {
	html := `<html><body><p>Simple</p></body></html>`
	got, err := ExtractMainContent(html, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Error("returned empty on simple HTML")
	}
}
