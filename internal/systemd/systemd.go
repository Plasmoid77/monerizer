// Package systemd reads unit state through systemctl (ТЗ SYS-06, SYS-07).
package systemd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a command with an argument vector; no shell is involved.
type Runner func(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)

// ExecRunner runs real commands with a bounded output size.
func ExecRunner(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(cmd.Environ(), "SYSTEMD_PAGER=", "SYSTEMD_COLORS=0", "LC_ALL=C")
	var out, errb bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &out, n: 1 << 20}
	cmd.Stderr = &limitedWriter{w: &errb, n: 64 << 10}
	err := cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

type limitedWriter struct {
	w *bytes.Buffer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if len(p) > l.n {
		p = p[:l.n]
	}
	l.n -= len(p)
	l.w.Write(p)
	return len(p), nil
}

// Props holds the key=value output of `systemctl show` for one unit.
type Props map[string]string

// ShowProperties is the SYS-06 property list.
var ShowProperties = []string{
	"Id", "LoadState", "ActiveState", "SubState", "UnitFileState", "Result", "MainPID", "ExecMainStatus",
	"NRestarts", "InvocationID", "ActiveEnterTimestampMonotonic", "InactiveExitTimestampMonotonic",
	"ExecMainStartTimestamp", "ExecMainStartTimestampMonotonic",
	"After", "Wants", "Requires", "BindsTo", "PartOf", "RuntimeDirectory", "RuntimeDirectoryPreserve",
}

// Show returns properties per unit, keyed by the requested unit name.
// A missing unit is reported by systemd as LoadState=not-found, not as an error.
func Show(ctx context.Context, run Runner, units ...string) (map[string]Props, error) {
	args := append([]string{"show", "--no-pager", "--all", "--property=" + strings.Join(ShowProperties, ",")}, "--")
	args = append(args, units...)
	out, stderr, err := run(ctx, "systemctl", args...)
	if err != nil {
		return nil, fmt.Errorf("systemctl show: %w: %s", err, strings.TrimSpace(string(stderr)))
	}
	blocks := ParseShow(out)
	res := make(map[string]Props, len(units))
	for i, u := range units {
		if i < len(blocks) {
			res[u] = blocks[i]
		}
	}
	if len(res) != len(units) {
		return res, errors.New("systemctl show: fewer blocks than units")
	}
	return res, nil
}

// ParseShow splits `systemctl show` output into per-unit blocks (blank-line separated).
func ParseShow(out []byte) []Props {
	var blocks []Props
	cur := Props{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			if len(cur) > 0 {
				blocks = append(blocks, cur)
				cur = Props{}
			}
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			cur[k] = v
		}
	}
	if len(cur) > 0 {
		blocks = append(blocks, cur)
	}
	return blocks
}

// HasDependency reports whether the space-separated list property contains unit.
func (p Props) HasDependency(prop, unit string) bool {
	for _, f := range strings.Fields(p[prop]) {
		if f == unit {
			return true
		}
	}
	return false
}
