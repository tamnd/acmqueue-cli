package acmqueue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const mockFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>ACM Queue - All Queue Content</title>
    <item>
      <title>Test Article One</title>
      <link>https://queue.acm.org/detail.cfm?ref=rss&amp;id=1001</link>
      <pubDate>Mon, 01 Jan 2024 12:00:00 GMT</pubDate>
      <guid isPermaLink="false">1001</guid>
    </item>
    <item>
      <title>Test Article Two</title>
      <link>https://queue.acm.org/detail.cfm?ref=rss&amp;id=1002</link>
      <pubDate>Tue, 02 Jan 2024 08:00:00 GMT</pubDate>
      <guid isPermaLink="false">1002</guid>
    </item>
  </channel>
</rss>`

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return NewClient(cfg), srv
}

func feedHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(body))
	})
}

func TestArticles_returnsAll(t *testing.T) {
	c, _ := newTestClient(t, feedHandler(mockFeed))
	arts, err := c.Articles(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("got %d articles, want 2", len(arts))
	}
	if arts[0].Title != "Test Article One" {
		t.Errorf("first title = %q", arts[0].Title)
	}
	if arts[1].Title != "Test Article Two" {
		t.Errorf("second title = %q", arts[1].Title)
	}
}

func TestArticles_withLimit(t *testing.T) {
	c, _ := newTestClient(t, feedHandler(mockFeed))
	arts, err := c.Articles(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("got %d articles, want 1", len(arts))
	}
}

func TestArticles_rank(t *testing.T) {
	c, _ := newTestClient(t, feedHandler(mockFeed))
	arts, err := c.Articles(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range arts {
		if a.Rank != i+1 {
			t.Errorf("articles[%d].Rank = %d, want %d", i, a.Rank, i+1)
		}
	}
}

func TestArticle_found(t *testing.T) {
	c, _ := newTestClient(t, feedHandler(mockFeed))
	a, err := c.Article(context.Background(), "1001")
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Test Article One" {
		t.Errorf("title = %q", a.Title)
	}
}

func TestArticle_notFound(t *testing.T) {
	c, _ := newTestClient(t, feedHandler(mockFeed))
	_, err := c.Article(context.Background(), "9999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(mockFeed))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(mockFeed))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}
