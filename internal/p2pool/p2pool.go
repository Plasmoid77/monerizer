// Package p2pool reads the fixed set of P2Pool Data API files (contracts §3–5,
// ТЗ DATA-02..04). It never parses wallet, peers or workers.
package p2pool

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/Plasmoid77/monerizer/internal/jsonx"
)

const maxFile = 1 << 20

// Names of the files read in v1, relative to data_api_dir.
const (
	FileP2P     = "local/p2p"
	FileStratum = "local/stratum"
	FileNetwork = "network/stats"
	FilePool    = "pool/stats"
)

// File is one read attempt of a Data API file.
type File struct {
	Name    string
	ModTime time.Time
	Object  jsonx.Object
	Err     error // nil when Object is valid
}

// ErrPermission and ErrMissing classify Err.
var (
	ErrMissing = errors.New("file missing")
	ErrEmpty   = errors.New("file empty")
)

// Read reads one file with one retry after 50 ms for missing/empty/corrupt
// content (DATA-04). Permission and size errors are not retried.
func Read(dir, name string) File {
	f := readOnce(dir, name)
	if f.Err == nil || errors.Is(f.Err, fs.ErrPermission) || errors.Is(f.Err, errTooLarge) || errors.Is(f.Err, errNotRegular) {
		return f
	}
	time.Sleep(50 * time.Millisecond)
	return readOnce(dir, name)
}

var (
	errTooLarge   = errors.New("file exceeds 1 MiB")
	errNotRegular = errors.New("not a regular file")
)

func readOnce(dir, name string) File {
	f := File{Name: name}
	path := filepath.Join(dir, filepath.FromSlash(name))
	fh, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = ErrMissing
		}
		f.Err = err
		return f
	}
	defer fh.Close()
	st, err := fh.Stat()
	if err != nil {
		f.Err = err
		return f
	}
	if !st.Mode().IsRegular() {
		f.Err = errNotRegular
		return f
	}
	if st.Size() > maxFile {
		f.Err = errTooLarge
		return f
	}
	f.ModTime = st.ModTime()
	data, err := readAll(fh, st.Size())
	if err != nil {
		f.Err = err
		return f
	}
	if len(data) == 0 {
		f.Err = ErrEmpty
		return f
	}
	f.Object, f.Err = jsonx.Decode(data)
	return f
}

func readAll(fh *os.File, size int64) ([]byte, error) {
	buf := make([]byte, size+1)
	n, err := fh.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n > maxFile {
		return nil, errTooLarge
	}
	return buf[:n], nil
}

// ReadDir reads all v1 files. The result is keyed by file name.
func ReadDir(dir string) map[string]File {
	out := make(map[string]File, 4)
	for _, n := range []string{FileP2P, FileStratum, FileNetwork, FilePool} {
		out[n] = Read(dir, n)
	}
	return out
}
