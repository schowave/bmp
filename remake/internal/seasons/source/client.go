// Package source fetches what the real seasons are built from: league
// tables, squads, European cup participants and club logos from the German
// Wikipedia, and squads from Wikidata where Wikipedia has none. Every
// response is cached on disk, so a second run works offline.
package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UserAgent identifies the fetcher, as Wikimedia asks; it carries no
// personal data.
const UserAgent = "bmp-remake-seasons/1.0 (https://github.com/schowave/bmp)"

// Client fetches pages and caches them in Dir.
type Client struct {
	Dir   string        // cache directory
	Delay time.Duration // pause between two requests to the network
	// Refresh fetches pages again instead of reading the cache, and stores
	// the new answers; for the running season, whose tables and squads
	// change.
	Refresh bool
	HTTP    *http.Client
	Log     func(format string, args ...any)

	last time.Time
}

// ErrNotFound is returned for a page that does not exist.
var ErrNotFound = errors.New("not found")

// get returns the body of url, from the cache when it is there.
func (c *Client) get(u string) ([]byte, error) {
	sum := sha256.Sum256([]byte(u))
	file := filepath.Join(c.Dir, hex.EncodeToString(sum[:16]))
	if b, err := os.ReadFile(file); err == nil && !c.Refresh {
		if string(b) == "\x00missing" {
			return nil, ErrNotFound
		}
		return b, nil
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 90 * time.Second}
	}
	var body []byte
	for try := 0; ; try++ {
		if d := time.Until(c.last.Add(c.Delay)); d > 0 {
			time.Sleep(d)
		}
		c.last = time.Now()
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		resp, err := c.HTTP.Do(req)
		if err == nil {
			body, err = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			switch {
			case err != nil:
			case resp.StatusCode == http.StatusNotFound:
				if err := c.store(file, []byte("\x00missing")); err != nil {
					return nil, err
				}
				return nil, ErrNotFound
			case resp.StatusCode == http.StatusOK:
				return body, c.store(file, body)
			default:
				err = fmt.Errorf("%s: %s", u, resp.Status)
			}
		}
		if try == 4 {
			return nil, err
		}
		if c.Log != nil {
			c.Log("retry %s: %v", u, err)
		}
		time.Sleep(time.Duration(5<<try) * time.Second)
	}
}

func (c *Client) store(file string, b []byte) error {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, b, 0o644)
}

// getJSON fetches url and decodes its JSON into v.
func (c *Client) getJSON(u string, v any) error {
	b, err := c.get(u)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

const wikiAPI = "https://de.wikipedia.org/w/api.php"

// Raw returns the wikitext of the German Wikipedia article title,
// following redirects; ErrNotFound when there is none.
func (c *Client) Raw(title string) (string, error) {
	for range 3 {
		b, err := c.get("https://de.wikipedia.org/w/index.php?action=raw&title=" + url.QueryEscape(strings.ReplaceAll(title, " ", "_")))
		if err != nil {
			return "", err
		}
		text := string(b)
		if to, ok := redirectTarget(text); ok {
			title = to
			continue
		}
		return text, nil
	}
	return "", fmt.Errorf("%s: too many redirects", title)
}

// redirectTarget reads "#WEITERLEITUNG [[X]]" / "#REDIRECT [[X]]".
func redirectTarget(text string) (string, bool) {
	t := strings.TrimSpace(text)
	up := strings.ToUpper(t)
	if !strings.HasPrefix(up, "#WEITERLEITUNG") && !strings.HasPrefix(up, "#REDIRECT") {
		return "", false
	}
	i := strings.Index(t, "[[")
	j := strings.Index(t, "]]")
	if i < 0 || j < i {
		return "", false
	}
	to, _, _ := strings.Cut(t[i+2:j], "#")
	return strings.TrimSpace(to), true
}

// Resolve maps article titles to the titles they redirect to (or
// themselves); titles without an article map to "".
func (c *Client) Resolve(titles []string) (map[string]string, error) {
	out := map[string]string{}
	for i := 0; i < len(titles); i += 50 {
		batch := titles[i:min(i+50, len(titles))]
		q := url.Values{"action": {"query"}, "format": {"json"}, "redirects": {"1"}, "titles": {strings.Join(batch, "|")}}
		var r struct {
			Query struct {
				Normalized, Redirects []struct{ From, To string }
				Pages                 map[string]struct {
					Title   string
					Missing *string
				}
			}
		}
		if err := c.getJSON(wikiAPI+"?"+q.Encode(), &r); err != nil {
			return nil, err
		}
		step := map[string]string{}
		for _, n := range r.Query.Normalized {
			step[n.From] = n.To
		}
		red := map[string]string{}
		for _, n := range r.Query.Redirects {
			red[n.From] = n.To
		}
		exists := map[string]bool{}
		for _, p := range r.Query.Pages {
			exists[p.Title] = p.Missing == nil
		}
		for _, t := range batch {
			to := t
			if n, ok := step[to]; ok {
				to = n
			}
			if n, ok := red[to]; ok {
				to = n
			}
			if !exists[to] {
				to = ""
			}
			out[t] = to
		}
	}
	return out, nil
}

// FileInfo is an image of Wikipedia or Commons.
type FileInfo struct {
	Title    string
	Thumb    string // URL of a PNG rendering of the requested width
	License  string // short licence name, e.g. "Public domain", "CC BY-SA 4.0"
	Artist   string // author, as plain text
	Page     string // description page
	Credit   string
	NonFree  bool
	Restrict string // e.g. "trademarked"
}

// File returns the description of file title ("Datei:Logo.svg") with a
// PNG thumbnail width pixels wide.
func (c *Client) File(title string, width int) (*FileInfo, error) {
	q := url.Values{"action": {"query"}, "format": {"json"}, "prop": {"imageinfo"}, "titles": {title},
		"iiprop": {"url|extmetadata"}, "iiurlwidth": {fmt.Sprint(width)}}
	var r struct {
		Query struct {
			Pages map[string]struct {
				Title     string
				ImageInfo []struct {
					ThumbURL       string `json:"thumburl"`
					DescriptionURL string `json:"descriptionurl"`
					ExtMetadata    map[string]struct{ Value any }
				} `json:"imageinfo"`
			}
		}
	}
	if err := c.getJSON(wikiAPI+"?"+q.Encode(), &r); err != nil {
		return nil, err
	}
	for _, p := range r.Query.Pages {
		if len(p.ImageInfo) == 0 {
			return nil, fmt.Errorf("%s: %w", title, ErrNotFound)
		}
		ii := p.ImageInfo[0]
		meta := func(k string) string {
			if v, ok := ii.ExtMetadata[k]; ok {
				if s, ok := v.Value.(string); ok {
					return stripTags(s)
				}
			}
			return ""
		}
		return &FileInfo{Title: p.Title, Thumb: ii.ThumbURL, License: meta("LicenseShortName"), Artist: meta("Artist"),
			Page: ii.DescriptionURL, Credit: meta("Credit"), NonFree: meta("NonFree") == "true",
			Restrict: meta("Restrictions")}, nil
	}
	return nil, fmt.Errorf("%s: %w", title, ErrNotFound)
}

// Download returns the bytes of url (a thumbnail), cached.
func (c *Client) Download(u string) ([]byte, error) { return c.get(u) }

// stripTags removes HTML tags and collapses white space.
func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
