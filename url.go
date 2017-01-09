package piazza

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/mvdan/xurls"
	"github.com/pkg/errors"
)

// HTMLWrapper returns a new HTMLWrapper
func (c *Client) HTMLWrapper() *HTMLWrapper {
	return &HTMLWrapper{
		c:        c,
		networks: map[string]Network{},
	}
}

// HTMLWrapper is wrapper on top of Client that wraps all results with HTML.
type HTMLWrapper struct {
	c        *Client
	networks map[string]Network
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
		for _, network := range status.Result.Networks {
			classID := network.ID
			if len(classID) == 0 {
				continue
			}
			w.networks[classID] = network
			url := fmt.Sprintf("%s://%s", PiazzaScheme, classID)
			classes = append(classes, url)
		}
		sort.Strings(classes)
		return urlsToHTML(classes), nil
	}

	if u.Host != "" && len(u.Path) <= 1 {
		network, ok := w.networks[u.Host]
		if !ok {
			return "", errors.New("need to fetch piazza:// before this")
		}
		req, err := http.NewRequest("GET", network.ResourceURL(), nil)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		resources, _ := ioutil.ReadAll(resp.Body)
		feed, err := w.c.Feed(u.Host)
		if err != nil {
			return "", err
		}
		links := xurls.Strict.FindAllString(string(resources), -1)
		// This matches urls in the form "\nhttp://...." and we need to strip the n.
		for i, link := range links {
			if strings.HasPrefix(link, "nhttp") {
				links[i] = link[1:]
			}
		}
		for _, post := range feed.Result.Feed {
			if len(post.ID) == 0 {
				continue
			}
			url := fmt.Sprintf("%s://%s/%s", PiazzaScheme, u.Host, post.ID)
			links = append(links, url)
		}
		return string(resources) + urlsToHTML(links), nil
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
