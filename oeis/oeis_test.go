package oeis_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/oeis-cli/oeis"
)

const fixtureSearch = `{
  "greeting": "Greetings from The On-Line Encyclopedia of Integer Sequences!",
  "query": "fibonacci",
  "count": 2,
  "start": 0,
  "results": [
    {
      "number": 45,
      "data": "0,1,1,2,3,5,8,13,21,34",
      "name": "Fibonacci numbers: F(n) = F(n-1) + F(n-2), with F(0) = 0 and F(1) = 1.",
      "comment": ["Also called Lamé's sequence."],
      "keyword": "nonn,easy,core,nice",
      "offset": "0,3",
      "author": "_N. J. A. Sloane_",
      "xref": ["A039834", "A212804", "A000035", "A001519"]
    },
    {
      "number": 285,
      "data": "1,4,5,9,14,23,37,60,97,157",
      "name": "a(0) = 1, a(1) = 4, and for n > 1, a(n) = a(n-1) + a(n-2).",
      "keyword": "nonn,easy",
      "offset": "0,2",
      "author": "_N. J. A. Sloane_",
      "xref": []
    }
  ]
}`

const fixtureEmpty = `{
  "greeting": "Greetings from The On-Line Encyclopedia of Integer Sequences!",
  "query": "xyzzy_no_results",
  "count": 0,
  "start": 0,
  "results": null
}`

const fixtureOne = `{
  "greeting": "Greetings from The On-Line Encyclopedia of Integer Sequences!",
  "query": "id:A000045",
  "count": 1,
  "start": 0,
  "results": [
    {
      "number": 45,
      "data": "0,1,1,2,3,5,8,13,21,34",
      "name": "Fibonacci numbers: F(n) = F(n-1) + F(n-2), with F(0) = 0 and F(1) = 1.",
      "comment": ["Also called Lamé's sequence."],
      "keyword": "nonn,easy,core,nice",
      "offset": "0,3",
      "author": "_N. J. A. Sloane_",
      "xref": ["A039834", "A212804", "A000035"]
    }
  ]
}`

const fixtureTop = `{
  "greeting": "Greetings from The On-Line Encyclopedia of Integer Sequences!",
  "query": "keyword:nice,core",
  "count": 3,
  "start": 0,
  "results": [
    {
      "number": 1,
      "data": "0,1,1,1,2,1,2,1,5,2",
      "name": "Number of groups of order n.",
      "keyword": "nonn,hard,core,nice",
      "offset": "0,5",
      "author": "_N. J. A. Sloane_",
      "xref": []
    },
    {
      "number": 2,
      "data": "1,2,2,1,1,2,1,2,2,1",
      "name": "Kolakoski sequence.",
      "keyword": "nonn,easy,core,nice",
      "offset": "1,2",
      "author": "_N. J. A. Sloane_",
      "xref": []
    },
    {
      "number": 5,
      "data": "1,2,3,5,7,11,13,17,19,23",
      "name": "The prime numbers.",
      "keyword": "nonn,core,nice",
      "offset": "1,3",
      "author": "_N. J. A. Sloane_",
      "xref": []
    }
  ]
}`

func newTestClient(t *testing.T, body string) (*oeis.Client, func()) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}))
	cfg := oeis.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return oeis.NewClient(cfg), ts.Close
}

func TestParseANumber(t *testing.T) {
	cases := []struct {
		in   string
		want int
		err  bool
	}{
		{"A000045", 45, false},
		{"a000045", 45, false},
		{"45", 45, false},
		{"000045", 45, false},
		{"A000079", 79, false},
		{"xyz", 0, true},
	}
	for _, tc := range cases {
		got, err := oeis.ParseANumber(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("ParseANumber(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseANumber(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseANumber(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestSearch(t *testing.T) {
	c, close := newTestClient(t, fixtureSearch)
	defer close()

	seqs, err := c.Search(context.Background(), "fibonacci", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(seqs) != 2 {
		t.Fatalf("len(seqs) = %d, want 2", len(seqs))
	}
	if seqs[0].ANumber != "A000045" {
		t.Errorf("seqs[0].ANumber = %q, want %q", seqs[0].ANumber, "A000045")
	}
	if seqs[0].Name == "" {
		t.Error("seqs[0].Name is empty")
	}
	if seqs[0].Comment != "Also called Lamé's sequence." {
		t.Errorf("seqs[0].Comment = %q", seqs[0].Comment)
	}
	// XRefs should be capped at 3 (fixture has 4)
	if len(seqs[0].XRefs) != 3 {
		t.Errorf("seqs[0].XRefs len = %d, want 3", len(seqs[0].XRefs))
	}
	if seqs[0].URL != "https://oeis.org/A000045" {
		t.Errorf("seqs[0].URL = %q", seqs[0].URL)
	}
}

func TestSearchEmpty(t *testing.T) {
	c, close := newTestClient(t, fixtureEmpty)
	defer close()

	seqs, err := c.Search(context.Background(), "xyzzy_no_results", 10)
	if err != nil {
		t.Fatalf("Search empty: %v", err)
	}
	if len(seqs) != 0 {
		t.Errorf("len(seqs) = %d, want 0", len(seqs))
	}
}

func TestGetSeq(t *testing.T) {
	c, close := newTestClient(t, fixtureOne)
	defer close()

	seq, err := c.GetSeq(context.Background(), 45)
	if err != nil {
		t.Fatalf("GetSeq: %v", err)
	}
	if seq.ANumber != "A000045" {
		t.Errorf("ANumber = %q, want %q", seq.ANumber, "A000045")
	}
	if seq.Data == "" {
		t.Error("Data is empty")
	}
	if len(seq.Keywords) == 0 {
		t.Error("Keywords is empty")
	}
}

func TestGetSeqNotFound(t *testing.T) {
	c, close := newTestClient(t, fixtureEmpty)
	defer close()

	_, err := c.GetSeq(context.Background(), 99999999)
	if !errors.Is(err, oeis.ErrNotFound) {
		t.Errorf("GetSeq not found: got %v, want ErrNotFound", err)
	}
}

func TestTop(t *testing.T) {
	c, close := newTestClient(t, fixtureTop)
	defer close()

	seqs, err := c.Top(context.Background(), 3)
	if err != nil {
		t.Fatalf("Top: %v", err)
	}
	if len(seqs) != 3 {
		t.Fatalf("len(seqs) = %d, want 3", len(seqs))
	}
	for _, s := range seqs {
		if len(s.ANumber) == 0 || s.ANumber[0] != 'A' {
			t.Errorf("ANumber %q does not start with A", s.ANumber)
		}
	}
}
