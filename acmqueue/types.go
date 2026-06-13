package acmqueue

import (
	"encoding/xml"
)

// Article is the record emitted for ACM Queue articles.
// Author, Topic, and Summary are empty for RSS-sourced records because
// the feed does not carry them. The fields are present for forward
// compatibility when richer data becomes available.
type Article struct {
	Rank      int    `json:"rank"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Topic     string `json:"topic"`
	Published string `json:"published"`
	Summary   string `json:"summary"`
	URL       string `json:"url"`
}

// Topic is the record emitted for ACM Queue topic taxonomy entries.
type Topic struct {
	Rank int    `json:"rank"`
	Name string `json:"name"`
}

// Topics returns the embedded ACM Queue editorial topic taxonomy.
// The list reflects the site's known categories as of June 2026.
// No HTTP call is made.
func Topics() []Topic {
	names := []string{
		"Artificial Intelligence",
		"DevOps",
		"Distributed Systems",
		"Languages & Compilers",
		"Networks",
		"Performance",
		"Reliability & Safety",
		"Security",
		"Software Architecture",
		"Software Engineering",
		"Storage",
		"Theory",
		"Tools",
		"Web",
	}
	out := make([]Topic, len(names))
	for i, n := range names {
		out[i] = Topic{Rank: i + 1, Name: n}
	}
	return out
}

// ─── wire types ──────────────────────────────────────────────────────────────

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

// rssItem holds the raw fields from one RSS <item> element.
// The <link> element in RSS 2.0 is a text node handled by encoding/xml.
type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	GUID    string `xml:"guid"`
}

