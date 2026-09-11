package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Plasmoid77/monerizer/internal/status"
)

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "monerizer"
	return v
}

func (m Model) render() string {
	if m.width > 0 && (m.width < minWidth || m.height < minHeight) {
		return m.compact()
	}
	var b strings.Builder
	s := m.snap
	if s == nil {
		b.WriteString("monerizer  collecting…\n\n  q quit  r refresh  s services  l logs  ? help\n")
		return b.String()
	}
	fmt.Fprintf(&b, "monerizer  %s  health: %s  (collected %s ago)\n\n", s.CollectedAt.Format("15:04:05 UTC"), strings.ToUpper(s.Health.Level), ago(s.CollectedAt))
	fmt.Fprintf(&b, "  %-28s %-18s %-9s %-8s %s\n", "SERVICE", "STATE", "ENABLED", "UPTIME", "RESTARTS")
	for _, sv := range []status.Service{s.Services.P2Pool, s.Services.XMRig} {
		state := sv.ActiveState + "/" + sv.SubState
		if sv.LoadState != "loaded" && sv.LoadState != "" {
			state = sv.LoadState
		}
		fmt.Fprintf(&b, "  %-28s %-18s %-9s %-8s %s\n", sv.Unit, state, sv.EnabledState, dur(sv.UptimeSeconds), i64(sv.RestartCount))
	}
	x := s.XMRig
	fmt.Fprintf(&b, "\n  XMRig %s  connected=%s  pool=%s\n", x.Version, boolStr(x.Connected), x.Pool)
	fmt.Fprintf(&b, "  hashrate 10s %s  60s %s  15m %s H/s   accepted %s  rejected %s   hugepages %s%%\n",
		f0(x.Hashrate10s), f0(x.Hashrate60s), f0(x.Hashrate15m), i64(x.Accepted), i64(x.Rejected), f0(x.HugepagesPercent))
	fmt.Fprintf(&b, "  %s\n", m.sparkline(m.width-4))
	p := s.P2Pool
	fmt.Fprintf(&b, "\n  P2Pool  p2p %s conn (%s in)  zmq %s s ago  %s\n", i64(p.P2PConnections), i64(p.P2PIncomingConnections), f0(p.ZMQAgeSeconds), age(s, status.SrcP2PoolP2P))
	fmt.Fprintf(&b, "  stratum %s H/s (15m)  shares %s  sidechain %s found / %s failed  %s\n", f0(p.Hashrate15m), i64(p.StratumShares), i64(p.SidechainSharesFound), i64(p.SidechainSharesFailed), age(s, status.SrcP2PoolStratum))
	fmt.Fprintf(&b, "  network height %s  sidechain height %s  pool %s H/s  %s\n", i64(p.NetworkHeight), i64(p.SidechainHeight), f0(p.PoolHashrate), age(s, status.SrcP2PoolPool))
	if m.prev != nil && m.prev.XMRig.Rejected != nil && x.Rejected != nil && *x.Rejected > *m.prev.XMRig.Rejected {
		b.WriteString("\n  warning: rejected submissions increased since the previous sample\n") // H-04
	}
	issues := 0
	for _, i := range s.Health.Issues {
		if i.Severity != "info" {
			if issues == 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "  %-7s %-24s %s\n", i.Severity, i.Code, i.Message)
			issues++
			if issues == 5 {
				break
			}
		}
	}
	if m.opResult != "" {
		fmt.Fprintf(&b, "\n  %s\n", m.opResult)
	}
	b.WriteString("\n" + m.dialog())
	b.WriteString("\n  q quit  r refresh  s services  l logs  ? help")
	return b.String()
}

func (m Model) dialog() string {
	switch m.mode {
	case modeTarget:
		title := "Control which service?"
		if m.forLogs {
			title = "Show logs of which service?"
		}
		return list(title, targets, m.target)
	case modeAction:
		return list("Action for "+targets[m.target]+":", actions, m.action)
	case modeConfirm:
		units := strings.Join(m.units(), ", ")
		c, o := "[Cancel]", " Confirm "
		if m.confirmOK {
			c, o = " Cancel ", "[Confirm]"
		}
		return fmt.Sprintf("  %s %s?\n  %s   %s   (←/→ then Enter, Esc cancels)\n", actions[m.action], units, c, o)
	case modeHelp:
		return "  Keys: q/Ctrl-C quit (miners keep running)  r refresh  s start/stop/restart  l journalctl -f (Ctrl-C returns)  Esc close\n  Colour carries no meaning; every state is written as text.\n"
	}
	return ""
}

func list(title string, items []string, sel int) string {
	var b strings.Builder
	b.WriteString("  " + title + "\n")
	for i, it := range items {
		mark := "  "
		if i == sel {
			mark = "> "
		}
		b.WriteString("    " + mark + it + "\n")
	}
	b.WriteString("  (↑/↓ Enter, Esc cancels)\n")
	return b.String()
}

func (m Model) compact() string {
	var b strings.Builder
	b.WriteString("monerizer (terminal too small, need 80x24)\n")
	if m.snap != nil {
		for _, sv := range []status.Service{m.snap.Services.P2Pool, m.snap.Services.XMRig} {
			fmt.Fprintf(&b, "%s %s\n", sv.Unit, sv.ActiveState)
		}
		fmt.Fprintf(&b, "health %s\n", m.snap.Health.Level)
	}
	b.WriteString("q quit")
	return b.String()
}

// sparkline draws the last width points of the 10 s hashrate; gaps stay blank (A07).
func (m Model) sparkline(width int) string {
	if width < 10 {
		width = 10
	}
	if width > 120 {
		width = 120
	}
	ramp := []rune("▁▂▃▄▅▆▇█")
	if m.ascii {
		ramp = []rune(".:-=+*#@")
	}
	pts := m.history
	if len(pts) > width {
		pts = pts[len(pts)-width:]
	}
	max := 0.0
	for _, p := range pts {
		if p.v != nil && *p.v > max {
			max = *p.v
		}
	}
	var b strings.Builder
	b.WriteString("10s [")
	for _, p := range pts {
		switch {
		case p.v == nil:
			b.WriteRune(' ')
		case *p.v == 0:
			b.WriteRune('_')
		default:
			i := int(*p.v / max * float64(len(ramp)-1))
			b.WriteRune(ramp[i])
		}
	}
	b.WriteString(strings.Repeat(" ", width-len(pts)))
	fmt.Fprintf(&b, "] max %.0f H/s, %d samples", max, len(pts))
	return b.String()
}

func ago(t time.Time) string { return dur(fp(time.Since(t).Seconds())) }
func fp(f float64) *float64  { return &f }

func age(s *status.Snapshot, name string) string {
	src := s.Sources[name]
	if src.State != status.StateOK && src.State != status.StateStale {
		return "[" + src.State + "]"
	}
	if src.AgeSeconds == nil {
		return "[age unknown]"
	}
	return "[" + src.State + " " + dur(src.AgeSeconds) + " old]"
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
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
