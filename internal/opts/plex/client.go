package plex

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"goduper/internal/opts"
)

// XML models (subset)
type mediaContainer struct {
	XMLName   xml.Name    `xml:"MediaContainer"`
	Size      int         `xml:"size,attr"`
	Directory []Directory `xml:"Directory"`
	Video     []Video     `xml:"Video"`
}

type Directory struct {
	Key   string `xml:"key,attr"`
	Type  string `xml:"type,attr"`  // "movie", "show"
	Title string `xml:"title,attr"` // e.g., "Movies"
}

type Video struct {
	RatingKey        string  `xml:"ratingKey,attr"`
	Key              string  `xml:"key,attr"`
	LibrarySectionID string  `xml:"librarySectionID,attr"`
	Title            string  `xml:"title,attr"`
	Year             int     `xml:"year,attr"`
	Guid             string  `xml:"guid,attr"`
	Media            []Media `xml:"Media"`
}

type Media struct {
	ID              string `xml:"id,attr"`
	Duration        int    `xml:"duration,attr"`
	VideoCodec      string `xml:"videoCodec,attr"`
	AudioCodec      string `xml:"audioCodec,attr"`
	VideoResolution string `xml:"videoResolution,attr"`
	Container       string `xml:"container,attr"`
	Bitrate         int    `xml:"bitrate,attr"`
	Width           int    `xml:"width,attr"`
	Height          int    `xml:"height,attr"`
	Part            []Part `xml:"Part"`
}

type Part struct {
	ID            string `xml:"id,attr"`
	File          string `xml:"file,attr"`
	Size          int64  `xml:"size,attr"`
	Duration      int    `xml:"duration,attr"`
	ExistsInt     int    `xml:"exists,attr"`     // with checkFiles=1
	AccessibleInt int    `xml:"accessible,attr"` // with checkFiles=1
}

// Client
type Client struct {
	base    *url.URL
	token   string
	http    *http.Client
	verbose bool
	timeout time.Duration
}

func NewClient(o opts.Options) (*Client, error) {
	u, err := url.Parse(o.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   o.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: o.InsecureTLS}, //nolint:gosec
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{
		base:  u,
		token: o.Token,
		http: &http.Client{
			Transport: tr,
			Timeout:   o.Timeout,
		},
		verbose: o.Verbose,
		timeout: o.Timeout,
	}, nil
}

func (c *Client) buildURL(path string, q url.Values) string {
	u := *c.base
	u.Path = strings.TrimRight(c.base.Path, "/") + path
	if q == nil {
		q = url.Values{}
	}
	q.Set("X-Plex-Token", c.token)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) getXML(ctx context.Context, rawURL string) (*mediaContainer, error) {
	if c.verbose {
		fmt.Fprintln(os.Stderr, "GET", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("X-Plex-Product", "goDuper")
	req.Header.Set("X-Plex-Version", "1.3")
	req.Header.Set("X-Plex-Client-Identifier", "goDuper-"+shortHost())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("plex http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var mc mediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&mc); err != nil {
		return nil, fmt.Errorf("decode xml: %w", err)
	}
	return &mc, nil
}

func shortHost() string {
	h, _ := os.Hostname()
	if len(h) > 20 {
		return h[:20]
	}
	return h
}

// Public API
func (c *Client) DiscoverSections(ctx context.Context, includeShows bool) ([]Directory, error) {
	u := c.buildURL("/library/sections", nil)
	mc, err := c.getXML(ctx, u)
	if err != nil {
		return nil, err
	}
	var out []Directory
	for _, d := range mc.Directory {
		if d.Type == "movie" || (includeShows && d.Type == "show") {
			out = append(out, d)
		}
	}
	return out, nil
}

func (c *Client) FetchDuplicatesForSection(ctx context.Context, id string) ([]Video, error) {
	q := url.Values{}
	q.Set("duplicate", "1")
	u := c.buildURL("/library/sections/"+id+"/all", q)
	mc, err := c.getXML(ctx, u)
	if err != nil {
		return nil, err
	}
	var vids []Video
	for _, v := range mc.Video {
		if len(v.Media) > 1 {
			vids = append(vids, v)
		}
	}
	return vids, nil
}

func (c *Client) DeepFetchItem(ctx context.Context, ratingKey string, verify bool) (*Video, error) {
	q := url.Values{}
	q.Set("includeChildren", "1")
	if verify {
		q.Set("checkFiles", "1")
	}
	u := c.buildURL("/library/metadata/"+ratingKey, q)
	mc, err := c.getXML(ctx, u)
	if err != nil {
		return nil, err
	}
	if len(mc.Video) == 0 {
		return nil, fmt.Errorf("no video for ratingKey %s", ratingKey)
	}
	return &mc.Video[0], nil
}
