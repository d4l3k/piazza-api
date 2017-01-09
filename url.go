package piazza

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/mvdan/xurls"
	"github.com/pkg/errors"
)

// HTMLWrapper returns a new HTMLWrapper
func (c *Client) HTMLWrapper() *HTMLWrapper {
	return &HTMLWrapper{c}
}

// HTMLWrapper is wrapper on top of Client that wraps all results with HTML.
type HTMLWrapper struct {
	c *Client
}

// PiazzaScheme is the fake URL scheme for Piazza. It's in the format of:
// "piazza://classID/contentID"
const PiazzaScheme = "piazza"

// Get makes a request to Piazza.
func (w *HTMLWrapper) Get(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != PiazzaScheme {
		return "", errors.Errorf("scheme is not %q", PiazzaScheme)
	}

	if u.Host == "" && len(u.Path) <= 1 {
		status, err := w.c.UserStatus()
		if err != nil {
			return "", err
		}
		var classes []string
		for classID := range status.Result.Config.EmailPrefs {
			url := fmt.Sprintf("%s://%s", PiazzaScheme, classID)
			classes = append(classes, url)
		}
		sort.Strings(classes)
		return urlsToHTML(classes), nil
	}

	if u.Host != "" && len(u.Path) <= 1 {
		feed, err := w.c.Feed(u.Host)
		if err != nil {
			return "", err
		}
		var posts []string
		for _, post := range feed.Result.Feed {
			url := fmt.Sprintf("%s://%s/%s", PiazzaScheme, u.Host, post.ID)
			posts = append(posts, url)
		}
		return urlsToHTML(posts), nil
	}

	contentID := u.Path
	if strings.HasPrefix(contentID, "/") {
		contentID = contentID[1:]
	}

	post, err := w.c.Content(u.Host, contentID)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	for _, p := range allChildrenPosts(post) {
		for _, h := range p.History {
			buf.WriteString(h.Content)
			urls := xurls.Strict.FindAllString(h.Content, -1)
			buf.WriteString(urlsToHTML(urls))
		}
	}
	return buf.String(), nil
}

func allChildrenPosts(post Post) []Post {
	posts := []Post{post}
	for _, child := range post.Children {
		posts = append(posts, allChildrenPosts(child)...)
	}
	return posts
}

func urlsToHTML(urls []string) string {
	var buf bytes.Buffer
	for _, url := range urls {
		fmt.Fprintf(&buf, "<a href=\"%s\">%s</a>\n", url, url)
	}
	return buf.String()
}
