package xmrig

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../testdata/xmrig/6.26.0/summary.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseFixture(t *testing.T) {
	s, err := Parse(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "monerizer-xmrig" || s.Version != "6.26.0" || s.Hashrate10s == nil || s.Hashrate15m != nil {
		t.Fatalf("unexpected %+v", s)
	}
	if *s.Accepted != 1 || *s.Rejected != 0 || *s.HugepagesTotal != 1172 || *s.ConnectionUptimeMs <= 0 {
		t.Fatalf("unexpected counters %+v", s)
	}
	if len(s.InvalidFields) != 0 {
		t.Fatalf("invalid fields: %v", s.InvalidFields)
	}
}

func TestParseTolerant(t *testing.T) {
	s, err := Parse([]byte(`{"id":5,"uptime":-1,"hashrate":{"total":["x",null]},"results":{"shares_good":5,"shares_total":3},"connection":{"uptime_ms":7}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := "id uptime hashrate.total[0] results.shares_total"
	if got := strings.Join(s.InvalidFields, " "); got != want {
		t.Fatalf("invalid fields %q, want %q", got, want)
	}
	if s.Hashrate60s != nil || s.Rejected != nil || *s.ConnectionUptimeMs != 7 || *s.Accepted != 5 {
		t.Fatalf("unexpected %+v", s)
	}
	for _, bad := range []string{`[]`, `{`, `null`, ``, `{"id":"x"} trailing`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestFetch(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/2/summary":
			w.Write(fixture(t))
		case "/redir/2/summary":
			http.Redirect(w, r, "/2/summary", http.StatusFound)
		case "/big/2/summary":
			w.Write(make([]byte, maxBody+1))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	c := &Client{HTTP: NewHTTPClient(time.Second), BaseURL: srv.URL, Token: "secret"}
	s, err := c.Fetch(context.Background())
	if err != nil || s.ID != "monerizer-xmrig" || auth != "Bearer secret" {
		t.Fatalf("fetch: %v %v %q", s, err, auth)
	}
	c.BaseURL = srv.URL + "/x"
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected auth error")
	} else if _, ok := err.(*AuthError); !ok {
		t.Fatalf("expected AuthError, got %v", err)
	}
	c.BaseURL = srv.URL + "/redir"
	if _, err := c.Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect error, got %v", err)
	}
	c.BaseURL = srv.URL + "/big"
	if _, err := c.Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "1 MiB") {
		t.Fatalf("expected size error, got %v", err)
	}
}
