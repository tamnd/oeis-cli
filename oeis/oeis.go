// Package oeis is the library behind the oeis command: the HTTP client,
// request shaping, and the typed data models for the Online Encyclopedia of
// Integer Sequences.
//
// The public JSON search endpoint at https://oeis.org/search?fmt=json is the
// only programmatic interface offered by OEIS. All three operations in this
// package (Search, GetSeq, Top) use it with different query strings.
// No authentication is required.
package oeis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to OEIS.
const DefaultUserAgent = "oeis/dev (+https://github.com/tamnd/oeis-cli)"

// ErrNotFound is returned by GetSeq when the API returns zero results.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for the OEIS client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://oeis.org",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   15 * time.Second,
	}
}

// Client talks to the OEIS search JSON API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    cfg.BaseURL,
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// Search returns sequences matching the free-text query.
// limit <= 0 defaults to 10.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Sequence, error) {
	if limit <= 0 {
		limit = 10
	}
	rawURL := c.buildURL(query, limit)
	return c.fetchSequences(ctx, rawURL)
}

// GetSeq returns the single sequence identified by the A-number integer.
// Returns ErrNotFound when the API returns no results.
func (c *Client) GetSeq(ctx context.Context, number int) (Sequence, error) {
	q := fmt.Sprintf("id:A%06d", number)
	rawURL := c.buildURL(q, 1)
	seqs, err := c.fetchSequences(ctx, rawURL)
	if err != nil {
		return Sequence{}, err
	}
	if len(seqs) == 0 {
		return Sequence{}, ErrNotFound
	}
	return seqs[0], nil
}

// Top returns popular/core sequences (keyword:nice,core).
// limit <= 0 defaults to 10.
func (c *Client) Top(ctx context.Context, limit int) ([]Sequence, error) {
	if limit <= 0 {
		limit = 10
	}
	rawURL := c.buildURL("keyword:nice,core", limit)
	return c.fetchSequences(ctx, rawURL)
}

// buildURL constructs the full search URL.
func (c *Client) buildURL(query string, n int) string {
	u, _ := url.Parse(c.baseURL + "/search")
	q := url.Values{}
	q.Set("fmt", "json")
	q.Set("q", query)
	q.Set("n", strconv.Itoa(n))
	u.RawQuery = q.Encode()
	return u.String()
}

// fetchSequences fetches and parses a search response.
func (c *Client) fetchSequences(ctx context.Context, rawURL string) ([]Sequence, error) {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	// OEIS /search?fmt=json returns a plain JSON array, not a wrapped object.
	var wire []wireSequence
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode %s: %w", rawURL, err)
	}
	out := make([]Sequence, 0, len(wire))
	for _, w := range wire {
		out = append(out, wireToSequence(w))
	}
	return out, nil
}

// get fetches rawURL with pacing and retries.
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
	req.Header.Set("Accept", "application/json")

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

// ParseANumber normalises user input to the integer sequence number.
// Accepts: "A000045", "a000045", "000045", "45".
// Returns an error when the input contains no parseable digits.
func ParseANumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == 'A' || s[0] == 'a') {
		s = s[1:]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid A-number %q: %w", s, err)
	}
	return n, nil
}
