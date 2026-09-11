package p2pool

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for src, dst := range map[string]string{"local-p2p": FileP2P, "local-stratum": FileStratum, "network-stats": FileNetwork, "pool-stats": FilePool} {
		b, err := os.ReadFile("../../testdata/p2pool/4.18/" + src + ".json")
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, dst)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, b, 0o644)
	}
	return dir
}

func TestReadDir(t *testing.T) {
	files := ReadDir(setup(t))
	for n, f := range files {
		if f.Err != nil {
			t.Errorf("%s: %v", n, f.Err)
		}
	}
	if c, fe := files[FileP2P].Object.Int("connections"); fe != nil || c != 10 {
		t.Fatalf("connections %d %v", c, fe)
	}
	if h, fe := files[FilePool].Object.Int("pool_statistics", "sidechainHeight"); fe != nil || h == 0 {
		t.Fatalf("sidechainHeight %d %v", h, fe)
	}
}

func TestReadErrors(t *testing.T) {
	dir := setup(t)
	os.WriteFile(filepath.Join(dir, FileP2P), []byte("{"), 0o644)
	if f := Read(dir, FileP2P); f.Err == nil {
		t.Error("corrupt: expected error")
	}
	os.WriteFile(filepath.Join(dir, FileP2P), nil, 0o644)
	if f := Read(dir, FileP2P); !errors.Is(f.Err, ErrEmpty) {
		t.Errorf("empty: %v", f.Err)
	}
	os.Remove(filepath.Join(dir, FileP2P))
	if f := Read(dir, FileP2P); !errors.Is(f.Err, ErrMissing) {
		t.Errorf("missing: %v", f.Err)
	}
	os.WriteFile(filepath.Join(dir, FileStratum), make([]byte, maxFile+1), 0o644)
	if f := Read(dir, FileStratum); !errors.Is(f.Err, errTooLarge) {
		t.Errorf("large: %v", f.Err)
	}
	if f := Read(dir, "local"); !errors.Is(f.Err, errNotRegular) {
		t.Errorf("dir: %v", f.Err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, FileNetwork+".fifo"), 0o600); err == nil {
		done := make(chan File, 1)
		go func() { done <- Read(dir, FileNetwork+".fifo") }()
		select {
		case f := <-done:
			if !errors.Is(f.Err, errNotRegular) {
				t.Errorf("fifo: %v", f.Err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("reading a FIFO must not block")
		}
	}
	os.Symlink(filepath.Join(dir, FilePool), filepath.Join(dir, "link"))
	if f := Read(dir, "link"); f.Err == nil {
		t.Error("symlink must be rejected")
	}
}
