package transform

import (
	"strings"
	"testing"
)

func TestClean_RemovesScriptAndStyle(t *testing.T) {
	html := `<html><body><script>alert('x')</script><style>.x{}</style><p>Hello</p></body></html>`
	got, err := Clean(html, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "script") {
		t.Error("script tag not removed")
	}
	if strings.Contains(got, "style") {
		t.Error("style tag not removed")
	}
	if !strings.Contains(got, "Hello") {
		t.Error("content was removed")
	}
}

func TestClean_RemovesNav(t *testing.T) {
	html := `<html><body><nav><a href="/">Home</a></nav><main><p>Content</p></main></body></html>`
	got, err := Clean(html, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "nav") {
		t.Error("nav not removed")
	}
	if !strings.Contains(got, "Content") {
		t.Error("main content removed")
	}
}

func TestClean_ExcludeTags(t *testing.T) {
	html := `<html><body><aside>Sidebar</aside><p>Main</p></body></html>`
	got, err := Clean(html, []string{"aside"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Sidebar") {
		t.Error("excluded tag not removed")
	}
	if !strings.Contains(got, "Main") {
		t.Error("main content removed")
	}
}

func TestClean_IncludeTags(t *testing.T) {
	html := `<html><body><article>Keep this</article><div>Remove this</div></body></html>`
	got, err := Clean(html, nil, []string{"article"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Keep this") {
		t.Error("included content removed")
	}
	if strings.Contains(got, "Remove this") {
		t.Error("non-included content not removed")
	}
}
