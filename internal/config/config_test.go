package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const good = `schema_version = 1
[services]
p2pool = "moneroid-p2pool.service"
xmrig = "moneroid-xmrig.service"
[p2pool]
data_api_dir = "/run/moneroid-p2pool-api"
[xmrig]
api_url = "http://127.0.0.1:18088"
expected_id = "moneroid-xmrig"
`

func write(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "m.toml")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadGood(t *testing.T) {
	c, err := Load(write(t, good))
	if err != nil {
		t.Fatal(err)
	}
	if c.UI.RefreshMs != 1000 || c.Services.P2Pool != "moneroid-p2pool.service" {
		t.Fatalf("unexpected config %+v", c)
	}
}

func TestLoadBad(t *testing.T) {
	cases := map[string]string{
		"unknown key":    good + "extra = 1\n",
		"duplicate key":  good + "schema_version = 1\n",
		"schema":         strings.Replace(good, "schema_version = 1", "schema_version = 2", 1),
		"same unit":      strings.Replace(good, `xmrig = "moneroid-xmrig.service"`, `xmrig = "moneroid-p2pool.service"`, 1),
		"unit glob":      strings.Replace(good, "moneroid-xmrig.service", "moneroid-*.service", 1),
		"relative path":  strings.Replace(good, "/run/moneroid-p2pool-api", "api", 1),
		"remote api":     strings.Replace(good, "127.0.0.1:18088", "10.0.0.1:18088", 1),
		"api no port":    strings.Replace(good, "127.0.0.1:18088", "127.0.0.1", 1),
		"api hostname":   strings.Replace(good, "127.0.0.1:18088", "localhost:18088", 1),
		"api path":       strings.Replace(good, "127.0.0.1:18088", "127.0.0.1:18088/x", 1),
		"empty id":       strings.Replace(good, `expected_id = "moneroid-xmrig"`, `expected_id = ""`, 1),
		"refresh low":    good + "[ui]\nrefresh_ms = 100\n",
		"refresh string": good + "[ui]\nrefresh_ms = \"1000\"\n",
	}
	for name, s := range cases {
		if _, err := Load(write(t, s)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Error("missing file: expected error")
	}
}
