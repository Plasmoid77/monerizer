// Package xmrig reads GET /2/summary from the local XMRig HTTP API
// (contracts §2). Only GET; the token never leaves the request header.
package xmrig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Plasmoid77/moneroid/internal/jsonx"
)

const maxBody = 1 << 20

// Summary is the normalized subset of /2/summary; nil means unknown.
type Summary struct {
	Version            string
	ID                 string
	UptimeSeconds      *int64
	Hashrate10s        *float64
	Hashrate60s        *float64
	Hashrate15m        *float64
	HugepagesAllocated *int64
	HugepagesTotal     *int64
	Accepted           *int64
	Rejected           *int64
	Pool               string
	ConnectionUptimeMs *int64
	// InvalidFields lists paths whose value had a wrong type or range (FIELD_INVALID).
	InvalidFields []string
}

// Client fetches the summary. HTTP must not follow redirects or use proxies (CFG-04).
type Client struct {
	HTTP    *http.Client
	BaseURL string
	Token   string
}

// NewHTTPClient returns a client that never follows redirects and ignores proxy env.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirects are not followed")
		},
	}
}

// AuthError marks HTTP 401/403 (PERMISSION_DENIED rather than unavailable).
type AuthError struct{ Status int }

func (e *AuthError) Error() string { return fmt.Sprintf("HTTP %d from XMRig API", e.Status) }

// Fetch performs one GET /2/summary.
func (c *Client) Fetch(ctx context.Context) (*Summary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/2/summary", nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from XMRig API", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("response body exceeds 1 MiB")
	}
	return Parse(body)
}

// Parse normalizes a /2/summary body.
func Parse(body []byte) (*Summary, error) {
	o, err := jsonx.Decode(body)
	if err != nil {
		return nil, err
	}
	s := &Summary{}
	note := func(fe *jsonx.FieldError) {
		if fe != nil && !fe.Missing {
			s.InvalidFields = append(s.InvalidFields, fe.Path)
		}
	}
	var fe *jsonx.FieldError
	if s.Version, fe = o.String("version"); fe != nil {
		note(fe)
	}
	if s.ID, fe = o.String("id"); fe != nil {
		note(fe)
	}
	s.UptimeSeconds = intp(o.Int("uptime"))(note)
	s.Hashrate10s = floatp(o.FloatAt(0, "hashrate", "total"))(note)
	s.Hashrate60s = floatp(o.FloatAt(1, "hashrate", "total"))(note)
	s.Hashrate15m = floatp(o.FloatAt(2, "hashrate", "total"))(note)
	s.HugepagesAllocated = intp(o.IntAt(0, "hugepages"))(note)
	s.HugepagesTotal = intp(o.IntAt(1, "hugepages"))(note)
	good := intp(o.Int("results", "shares_good"))(note)
	total := intp(o.Int("results", "shares_total"))(note)
	s.Accepted = good
	if good != nil && total != nil {
		if *total >= *good {
			r := *total - *good
			s.Rejected = &r
		} else {
			note(&jsonx.FieldError{Path: "results.shares_total"})
		}
	}
	if s.Pool, fe = o.String("connection", "pool"); fe != nil {
		note(fe)
	}
	s.ConnectionUptimeMs = intp(o.Int("connection", "uptime_ms"))(note)
	return s, nil
}

func intp(v int64, fe *jsonx.FieldError) func(func(*jsonx.FieldError)) *int64 {
	return func(note func(*jsonx.FieldError)) *int64 {
		if fe != nil {
			note(fe)
			return nil
		}
		return &v
	}
}

func floatp(v float64, fe *jsonx.FieldError) func(func(*jsonx.FieldError)) *float64 {
	return func(note func(*jsonx.FieldError)) *float64 {
		if fe != nil {
			note(fe)
			return nil
		}
		return &v
	}
}
