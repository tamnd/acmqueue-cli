// Package acmqueue is the library behind the acmq command: the HTTP client,
// request shaping, and the typed data models for ACM Queue.
//
// Data source: the public RSS feed at
// https://queue.acm.org/rss/feeds/queuecontent.xml — open, no auth, no
// JavaScript required. The feed carries the 20 most recent articles with
// title, URL, pubDate, and a numeric article ID. The article detail pages
// sit behind Cloudflare and are not fetched.
package acmqueue

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// FeedURL is the ACM Queue RSS feed endpoint.
const FeedURL = "https://queue.acm.org/rss/feeds/queuecontent.xml"

// DefaultUserAgent identifies the client to ACM Queue.
const DefaultUserAgent = "acmq/dev (+https://github.com/tamnd/acmqueue-cli)"

// ErrNotFound is returned when an article ID is not in the current feed.
var ErrNotFound = errors.New("not found")

// ErrBadID is returned when ParseArticleID cannot extract a numeric id.
var ErrBadID = errors.New("bad article id")

// Config holds constructor parameters for the Client.
type Config struct {
	// BaseURL is the feed URL. Override in tests to point at a mock server.
	// Defaults to FeedURL.
	BaseURL   string
	UserAgent string
	// Rate is the minimum spacing between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults for production use.
func DefaultConfig() Config {
	return Config{
		BaseURL:   FeedURL,
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the ACM Queue RSS feed.
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured according to cfg.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = FeedURL
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    cfg.BaseURL,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// Articles fetches the RSS feed and returns up to limit articles (0 = all).
// Articles are ranked 1-N in feed order (newest first).
func (c *Client) Articles(ctx context.Context, limit int) ([]Article, error) {
	items, err := c.fetchFeed(ctx)
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]Article, len(items))
	for i, it := range items {
		out[i] = rssItemToArticle(it, i+1)
	}
	return out, nil
}

// Article fetches the RSS feed and returns the one article matching id.
// id is the numeric string from the guid element, e.g. "3807964".
// Returns ErrNotFound if no article with that id is in the current feed.
func (c *Client) Article(ctx context.Context, id string) (Article, error) {
	items, err := c.fetchFeed(ctx)
	if err != nil {
		return Article{}, err
	}
	for i, it := range items {
		if it.GUID == id {
			return rssItemToArticle(it, i+1), nil
		}
	}
	return Article{}, ErrNotFound
}

// fetchFeed retrieves and parses the RSS feed.
func (c *Client) fetchFeed(ctx context.Context) ([]rssItem, error) {
	body, err := c.get(ctx, c.baseURL)
	if err != nil {
		return nil, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	return feed.Channel.Items, nil
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// ParseArticleID extracts a numeric article id from s.
// s may be a bare integer string ("3807964") or a full article URL
// ("https://queue.acm.org/detail.cfm?ref=rss&id=3807964").
// Returns ErrBadID if neither form matches.
func ParseArticleID(s string) (string, error) {
	s = strings.TrimSpace(s)
	// Full URL: extract the id query parameter.
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", ErrBadID
		}
		id := u.Query().Get("id")
		if id == "" {
			return "", ErrBadID
		}
		if !isNumeric(id) {
			return "", ErrBadID
		}
		return id, nil
	}
	// Bare integer.
	if isNumeric(s) {
		return s, nil
	}
	return "", ErrBadID
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// rssItemToArticle converts a wire item to an Article record.
func rssItemToArticle(it rssItem, rank int) Article {
	return Article{
		Rank:      rank,
		Title:     it.Title,
		Author:    "",
		Topic:     "",
		Published: parsePubDate(it.PubDate),
		Summary:   "",
		URL:       it.Link,
	}
}

// parsePubDate parses an RFC 1123 date string and returns RFC 3339.
// Falls back to the raw string on parse error.
func parsePubDate(s string) string {
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		// Try without timezone name (some feeds use +0000 instead of GMT).
		t2, err2 := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", s)
		if err2 != nil {
			return s
		}
		return t2.UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
}
