package piazza

import (
	"os"
	"reflect"
	"testing"

	"github.com/mvdan/xurls"
)

func clientFromEnv(t *testing.T) *Client {
	user := os.Getenv("PIAZZAUSER")
	pass := os.Getenv("PIAZZAPASS")
	c, err := MakeClient(user, pass)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHTMLWrapper(t *testing.T) {
	c := clientFromEnv(t)
	w := c.HTMLWrapper()
	cases := []struct {
		url      string
		expected string
	}{
		{"piazza://", ""},
		{"piazza://ixe691ydpaazc", ""},
	}
	for _, c := range cases {
		out, err := w.Get(c.url)
		if err != nil {
			t.Fatal(err)
		}
		if out != c.expected {
			t.Errorf("w.Get(%q) = %q; not %q", c.url, out, c.expected)
		}
	}
}

func TestXURLs(t *testing.T) {
	html := `<a href="https://fn.lc/duck">Duck</a>`
	links := xurls.Strict.FindAllString(html, -1)
	want := []string{"https://fn.lc/duck"}
	if !reflect.DeepEqual(want, links) {
		t.Errorf("xurls.Strict.FindAllString(%q) = %+v; not %+v", html, links, want)
	}
}
