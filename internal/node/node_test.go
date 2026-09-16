package node

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseList(t *testing.T) {
	c, err := ParseList(strings.NewReader("# comment\n\nxmr.support 18081 18083 # trailing\n1.2.3.4 18089 18083\n"))
	if err != nil || len(c) != 2 || c[0].Host != "xmr.support" || c[1].RPC != 18089 {
		t.Fatalf("%v %v", c, err)
	}
	if c, err := ParseList(strings.NewReader("2001:db8::1 18081 18083\n")); err != nil || c[0].Host != "2001:db8::1" {
		t.Fatalf("IPv6 literal: %v %v", c, err)
	}
	for _, bad := range []string{"host 1", "host abc 18083", "http://x 18081 18083", "h 0 1", "a:b 18081 18083"} {
		if _, err := ParseList(strings.NewReader(bad)); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestProbe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "get_block_headers_range") {
			w.Write([]byte(`{"jsonrpc":"2.0","id":"0","result":{"headers":[{"height":1}]}}`))
			return
		}
		w.Write([]byte(`{"jsonrpc":"2.0","id":"0","result":{"height":1000,"synchronized":true}}`))
	}))
	defer srv.Close()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { // minimal ZMTP responder
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 10)
				io.ReadFull(c, buf)
				c.Write([]byte{0xff, 0, 0, 0, 0, 0, 0, 0, 1, 0x7f})
			}(c)
		}
	}()
	host, rpcs, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	rpc, _ := strconv.Atoi(rpcs)
	zmq := ln.Addr().(*net.TCPAddr).Port
	res := ProbeAll(context.Background(), NewHTTPClient(), []Candidate{{Host: host, RPC: rpc, ZMQ: 1}, {Host: host, RPC: rpc, ZMQ: zmq}, {Host: host, RPC: 1, ZMQ: zmq}})
	if !res[0].Usable() || res[0].ZMQ != zmq || *res[0].Height != 1000 || !*res[0].HeadersOK {
		t.Fatalf("first %+v", res[0])
	}
	if res[1].Usable() || res[2].Error == "" {
		t.Fatalf("rest %+v %+v", res[1], res[2])
	}
}

func TestRewrite(t *testing.T) {
	in := "# node\nhost = old.example   # keep comment? no, line is replaced\nwallet = 4abc\n\nrpc-port=18081\nmini = 1"
	out := string(Rewrite([]byte(in), Candidate{Host: "new.example", RPC: 18089, ZMQ: 18083}))
	want := "# node\nhost = new.example\nwallet = 4abc\n\nrpc-port = 18089\nmini = 1\nzmq-port = 18083\n"
	if out != want {
		t.Fatalf("got %q\nwant %q", out, want)
	}
	if c := ReadParams([]byte(out)); c.Host != "new.example" || c.RPC != 18089 || c.ZMQ != 18083 {
		t.Fatalf("read back %+v", c)
	}
	if c := ReadParams(nil); c.Host != "127.0.0.1" || c.RPC != 18081 {
		t.Fatalf("defaults %+v", c)
	}
	if d := Diff([]byte(in), []byte(out)); len(d) != 5 {
		t.Fatalf("diff %v", d)
	}
	dup := "host = a\nhost = b\nrpc-port = 1\nzmq-port = 2\nrpc-port = 3\n"
	if got := string(Rewrite([]byte(dup), Candidate{Host: "c", RPC: 4, ZMQ: 5})); got != "host = c\nrpc-port = 4\nzmq-port = 5\n" {
		t.Fatalf("duplicates must be dropped, got %q", got)
	}
}

func TestWriteAtomic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "p2pool.conf")
	os.WriteFile(p, []byte("a = 1\n"), 0o640)
	if err := WriteAtomic(p, []byte("a = 2\n")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	st, _ := os.Stat(p)
	if string(b) != "a = 2\n" || st.Mode().Perm() != 0o640 {
		t.Fatalf("%q %v", b, st.Mode())
	}
	if err := WriteAtomic(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("missing file must not be created")
	}
}
