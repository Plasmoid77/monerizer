package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Plasmoid77/monerizer/internal/status"
)

func printStatus(w io.Writer, s *status.Snapshot) {
	fmt.Fprintf(w, "Monerizer status  %s  (collected in %d ms)\n", s.CollectedAt.Format(time.RFC3339), s.CollectionDurationMs)
	fmt.Fprintf(w, "Health: %s\n\n", s.Health.Level)

	fmt.Fprintf(w, "%-28s %-20s %-9s %-7s %-9s %s\n", "SERVICE", "STATE", "ENABLED", "PID", "UPTIME", "RESTARTS")
	for _, sv := range []status.Service{s.Services.P2Pool, s.Services.XMRig} {
		state := sv.ActiveState
		if sv.SubState != "" {
			state += "/" + sv.SubState
		}
		if sv.LoadState != "loaded" && sv.LoadState != "" {
			state = sv.LoadState
		}
		fmt.Fprintf(w, "%-28s %-20s %-9s %-7s %-9s %s\n", sv.Unit, or(state, "?"), or(sv.EnabledState, "?"), i64(sv.PID), dur(sv.UptimeSeconds), i64(sv.RestartCount))
	}

	x := s.XMRig
	fmt.Fprintf(w, "\nXMRig %s  id=%s  pool=%s  connected=%s\n", or(x.Version, "?"), or(x.ID, "?"), or(x.Pool, "?"), boolStr(x.Connected))
	fmt.Fprintf(w, "  hashrate 10s/60s/15m: %s / %s / %s H/s   accepted %s  rejected %s   hugepages %s/%s (%s%%)\n",
		f0(x.Hashrate10s), f0(x.Hashrate60s), f0(x.Hashrate15m), i64(x.Accepted), i64(x.Rejected), i64(x.HugepagesAllocated), i64(x.HugepagesTotal), f0(x.HugepagesPercent))

	p := s.P2Pool
	fmt.Fprintln(w, "P2Pool")
	fmt.Fprintf(w, "  p2p:     connections %s (incoming %s)  known peers %s  zmq activity %s s ago   %s\n",
		i64(p.P2PConnections), i64(p.P2PIncomingConnections), i64(p.PeerListSize), f0(p.ZMQAgeSeconds), age(s, status.SrcP2PoolP2P))
	fmt.Fprintf(w, "  stratum: %s H/s (15m)  %s H/s (1h)  connections %s  stratum shares %s  sidechain shares %s found / %s failed   %s\n",
		f0(p.Hashrate15m), f0(p.Hashrate1h), i64(p.StratumConnections), i64(p.StratumShares), i64(p.SidechainSharesFound), i64(p.SidechainSharesFailed), age(s, status.SrcP2PoolStratum))
	fmt.Fprintf(w, "  network: height %s  difficulty %s   %s\n", i64(p.NetworkHeight), f0(p.NetworkDifficulty), age(s, status.SrcP2PoolNetwork))
	fmt.Fprintf(w, "  pool:    hashrate %s H/s  sidechain height %s  difficulty %s   %s\n", f0(p.PoolHashrate), i64(p.SidechainHeight), f0(p.SidechainDifficulty), age(s, status.SrcP2PoolPool))

	names := make([]string, 0, len(s.Sources))
	for n := range s.Sources {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		src := s.Sources[n]
		p := n + " " + src.State
		if src.ErrorCode != "" {
			p += " (" + src.ErrorCode + ")"
		}
		parts = append(parts, p)
	}
	fmt.Fprintf(w, "\nSources: %s\n", strings.Join(parts, " · "))
	if len(s.Health.Issues) > 0 {
		fmt.Fprintln(w, "Issues:")
		for _, i := range s.Health.Issues {
			fmt.Fprintf(w, "  %-7s %-26s %s\n", i.Severity, i.Code, i.Message)
		}
	}
}

func age(s *status.Snapshot, name string) string {
	src := s.Sources[name]
	if src.State != status.StateOK && src.State != status.StateStale {
		return "[" + src.State + "]"
	}
	if src.AgeSeconds == nil {
		return "[age unknown]"
	}
	return fmt.Sprintf("[%s, %s old]", src.State, dur(src.AgeSeconds))
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func i64(v *int64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprint(*v)
}

func f0(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f", *v)
}

func boolStr(b *bool) string {
	if b == nil {
		return "unknown"
	}
	if *b {
		return "yes"
	}
	return "no"
}

func dur(sec *float64) string {
	if sec == nil {
		return "—"
	}
	d := time.Duration(*sec * float64(time.Second)).Round(time.Second)
	if d >= time.Hour {
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
