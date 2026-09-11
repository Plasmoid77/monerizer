// Package node probes candidate Monero nodes and rewrites the node keys of the
// P2Pool params-file (ТЗ §6.1, NODE-01..05).
package node

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Plasmoid77/monerizer/internal/jsonx"
)

const (
	ProbeTimeout = 3 * time.Second
	maxBody      = 1 << 20
)

type Candidate struct {
	Host string `json:"host"`
	RPC  int    `json:"rpc_port"`
	ZMQ  int    `json:"zmq_port"`
}

// ParseList parses nodes.txt: `host rpc_port zmq_port` per line, `#` comments (NODE-01).
func ParseList(r io.Reader) ([]Candidate, error) {
	var out []Candidate
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 {
			return nil, fmt.Errorf("line %d: expected `host rpc_port zmq_port`", n)
		}
		rpc, err1 := strconv.Atoi(f[1])
		zmq, err2 := strconv.Atoi(f[2])
		if err1 != nil || err2 != nil || rpc < 1 || rpc > 65535 || zmq < 1 || zmq > 65535 || strings.ContainsAny(f[0], "/@:") {
			return nil, fmt.Errorf("line %d: invalid host or ports", n)
		}
		out = append(out, Candidate{Host: f[0], RPC: rpc, ZMQ: zmq})
	}
	return out, sc.Err()
}

// Result is the outcome of probing one candidate.
type Result struct {
	Candidate
	LatencyMs    *int64 `json:"latency_ms"`
	Synchronized *bool  `json:"synchronized"`
	Height       *int64 `json:"height"`
	ZMQOpen      *bool  `json:"zmq_open"`
	Error        string `json:"error,omitempty"`
}

// Usable reports whether the node is synchronized with an open ZMQ port.
func (r Result) Usable() bool {
	return r.Synchronized != nil && *r.Synchronized && r.ZMQOpen != nil && *r.ZMQOpen
}

// Probe performs get_info over HTTP and a TCP connect to the ZMQ port.
func Probe(ctx context.Context, client *http.Client, c Candidate) Result {
	res := Result{Candidate: c}
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	body := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"get_info"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s/json_rpc", net.JoinHostPort(c.Host, strconv.Itoa(c.RPC))), body)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		res.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	}
	lat := time.Since(start).Milliseconds()
	res.LatencyMs = &lat
	o, err := jsonx.Decode(raw)
	if err != nil {
		res.Error = "invalid JSON-RPC response"
		return res
	}
	if v, fe := o.Int("result", "height"); fe == nil {
		res.Height = &v
	}
	if r, ok := o["result"].(map[string]any); ok {
		if b, ok := r["synchronized"].(bool); ok {
			res.Synchronized = &b
		}
	}
	d := net.Dialer{Timeout: ProbeTimeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.ZMQ)))
	open := err == nil
	if open {
		conn.Close()
	}
	res.ZMQOpen = &open
	return res
}

// ProbeAll probes candidates in parallel and returns them ranked (NODE-02):
// usable nodes first by latency, then the rest in input order.
func ProbeAll(ctx context.Context, client *http.Client, cands []Candidate) []Result {
	results := make([]Result, len(cands))
	var wg sync.WaitGroup
	for i, c := range cands {
		wg.Add(1)
		go func(i int, c Candidate) {
			defer wg.Done()
			results[i] = Probe(ctx, client, c)
		}(i, c)
	}
	wg.Wait()
	sort.SliceStable(results, func(a, b int) bool {
		ua, ub := results[a].Usable(), results[b].Usable()
		if ua != ub {
			return ua
		}
		return ua && *results[a].LatencyMs < *results[b].LatencyMs
	})
	return results
}

// NewHTTPClient returns a client without proxy or redirects.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       ProbeTimeout,
		Transport:     &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirects are not followed") },
	}
}

var nodeKeys = []string{"host", "rpc-port", "zmq-port"}

// ReadParams returns the node keys of a params-file, with P2Pool defaults for absent keys.
func ReadParams(data []byte) Candidate {
	c := Candidate{Host: "127.0.0.1", RPC: 18081, ZMQ: 18083}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := splitKV(line)
		if !ok {
			continue
		}
		switch k {
		case "host":
			c.Host = v
		case "rpc-port":
			if n, err := strconv.Atoi(v); err == nil {
				c.RPC = n
			}
		case "zmq-port":
			if n, err := strconv.Atoi(v); err == nil {
				c.ZMQ = n
			}
		}
	}
	return c
}

func splitKV(line string) (string, string, bool) {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return "", "", false
	}
	k, v, ok := strings.Cut(s, "=")
	if !ok {
		return "", "", false
	}
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, '#'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return strings.TrimSpace(k), strings.Trim(v, `"`), true
}

// Rewrite replaces exactly the three node keys, keeping every other byte (NODE-03).
// Missing keys are appended at the end.
func Rewrite(data []byte, c Candidate) []byte {
	values := map[string]string{"host": c.Host, "rpc-port": strconv.Itoa(c.RPC), "zmq-port": strconv.Itoa(c.ZMQ)}
	seen := map[string]bool{}
	lines := strings.SplitAfter(string(data), "\n")
	var out strings.Builder
	for _, line := range lines {
		k, _, ok := splitKV(line)
		if ok && values[k] != "" && !seen[k] {
			seen[k] = true
			out.WriteString(k + " = " + values[k])
			if strings.HasSuffix(line, "\n") {
				out.WriteString("\n")
			}
			continue
		}
		out.WriteString(line)
	}
	s := out.String()
	for _, k := range nodeKeys {
		if !seen[k] {
			if s != "" && !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
			s += k + " = " + values[k] + "\n"
		}
	}
	return []byte(s)
}

// WriteAtomic writes data next to path and renames it over path, preserving
// the original owner and mode.
func WriteAtomic(path string, data []byte) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".p2pool.conf.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		if err := tmp.Chown(int(sys.Uid), int(sys.Gid)); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Chmod(st.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Diff lists the lines that Rewrite changes, for --dry-run output.
func Diff(old, new []byte) []string {
	var out []string
	o := strings.Split(strings.TrimRight(string(old), "\n"), "\n")
	n := strings.Split(strings.TrimRight(string(new), "\n"), "\n")
	for i := 0; i < len(o) || i < len(n); i++ {
		var a, b string
		if i < len(o) {
			a = o[i]
		}
		if i < len(n) {
			b = n[i]
		}
		if a != b {
			if a != "" {
				out = append(out, "- "+a)
			}
			if b != "" {
				out = append(out, "+ "+b)
			}
		}
	}
	return out
}
