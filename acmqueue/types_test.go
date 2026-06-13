package acmqueue

import (
	"errors"
	"testing"
)

func TestParseArticleID_bareInt(t *testing.T) {
	id, err := ParseArticleID("3807964")
	if err != nil {
		t.Fatal(err)
	}
	if id != "3807964" {
		t.Errorf("id = %q, want %q", id, "3807964")
	}
}

func TestParseArticleID_fullURL(t *testing.T) {
	id, err := ParseArticleID("https://queue.acm.org/detail.cfm?ref=rss&id=3807964")
	if err != nil {
		t.Fatal(err)
	}
	if id != "3807964" {
		t.Errorf("id = %q, want %q", id, "3807964")
	}
}

func TestParseArticleID_badInput(t *testing.T) {
	_, err := ParseArticleID("notanid")
	if !errors.Is(err, ErrBadID) {
		t.Fatalf("got %v, want ErrBadID", err)
	}
}

func TestTopics_count(t *testing.T) {
	topics := Topics()
	if len(topics) != 14 {
		t.Errorf("len(Topics()) = %d, want 14", len(topics))
	}
}

func TestTopics_ranked(t *testing.T) {
	topics := Topics()
	for i, tp := range topics {
		if tp.Rank != i+1 {
			t.Errorf("topics[%d].Rank = %d, want %d", i, tp.Rank, i+1)
		}
	}
}
