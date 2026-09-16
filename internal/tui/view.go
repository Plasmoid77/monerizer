package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Plasmoid77/monerizer/internal/ansi"
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
	w := m.width
	if w == 0 {
		w = minWidth
	}
	s := m.snap
	title := " monerizer " + m.version + "   " + m.host
	if s == nil {
		return m.fit(ansi.Banner(pad(title+"   collecting…", w))+"\n", w)
	}
	var b strings.Builder
	b.WriteString(ansi.Banner(pad(fmt.Sprintf("%s   %s   health %s   collected %s ago", title, s.CollectedAt.Local().Format("15:04:05 MST"), strings.ToUpper(s.Health.Level), ago(s.CollectedAt)), w)) + "\n\n")

	fmt.Fprintf(&b, "  %s\n", ansi.Orange(fmt.Sprintf("%-28s %-18s %-9s %-8s %s", "SERVICE", "STATE", "ENABLED", "UPTIME", "RESTARTS")))
	for _, sv := range []status.Service{s.Services.P2Pool, s.Services.XMRig} {
		state := sv.ActiveState + "/" + sv.SubState
		if sv.LoadState != "loaded" && sv.LoadState != "" {
			state = sv.LoadState
		}
		fmt.Fprintf(&b, "  %-28s %s %-9s %-8s %s\n", sv.Unit, ansi.Unit(fmt.Sprintf("%-18s", state)), sv.EnabledState, dur(sv.UptimeSeconds), i64(sv.RestartCount))
	}

	x := s.XMRig
	bw := w - 50 // meter width: label + value + trailing text fit into the rest
	if bw > 60 {
		bw = 60
	}
	fmt.Fprintf(&b, "\n  %s %s   connected %s   pool %s   %s\n", ansi.Orange("XMRIG"), x.Version, boolStr(x.Connected), x.Pool, age(s, status.SrcXMRigSummary))
	max := m.historyMax()
	frac := 0.0
	if x.Hashrate10s != nil && max > 0 {
		frac = *x.Hashrate10s / max
	}
	fmt.Fprintf(&b, "  10s %s H/s  [%s] max %.0f\n", ansi.White(fmt.Sprintf("%8s", f0(x.Hashrate10s))), ansi.Bar(bw, frac, m.ascii), max)
	rej := i64(x.Rejected)
	if x.Rejected != nil && *x.Rejected > 0 {
		rej = ansi.Red(rej)
	}
	fmt.Fprintf(&b, "  60s %s H/s   15m %s H/s   accepted %s   rejected %s\n", ansi.White(fmt.Sprintf("%8s", f0(x.Hashrate60s))), ansi.White(f0(x.Hashrate15m)), i64(x.Accepted), rej)
	hp := 0.0
	if x.HugepagesPercent != nil {
		hp = *x.HugepagesPercent / 100
	}
	fmt.Fprintf(&b, "  hugepages %5s%%  [%s]\n", f0(x.HugepagesPercent), ansi.Bar(bw, hp, m.ascii))
	fmt.Fprintf(&b, "  %s\n", m.sparkline(w-40))

	p := s.P2Pool
	fmt.Fprintf(&b, "\n  %s   p2p %s conn (%s in)   zmq %s s ago   %s\n", ansi.Orange("P2POOL"), i64(p.P2PConnections), i64(p.P2PIncomingConnections), f0(p.ZMQAgeSeconds), age(s, status.SrcP2PoolP2P))
	fmt.Fprintf(&b, "  stratum %s H/s (15m)  shares %s  sidechain %s found / %s failed  %s\n", ansi.White(f0(p.Hashrate15m)), i64(p.StratumShares), ansi.White(i64(p.SidechainSharesFound)), i64(p.SidechainSharesFailed), age(s, status.SrcP2PoolStratum))
	sync := "peers —"
	if p.SidechainHeight != nil && p.PeerMaxHeight != nil && *p.PeerMaxHeight > 0 {
		sync = fmt.Sprintf("peers %d  [%s]", *p.PeerMaxHeight, ansi.Bar(20, float64(*p.SidechainHeight)/float64(*p.PeerMaxHeight), m.ascii))
	}
	fmt.Fprintf(&b, "  sidechain height %s / %s   network height %s   %s\n", ansi.White(i64(p.SidechainHeight)), sync, i64(p.NetworkHeight), age(s, status.SrcP2PoolNetwork))
	share := "—"
	if p.Hashrate15m != nil && p.PoolHashrate != nil && *p.PoolHashrate > 0 {
		share = fmt.Sprintf("%.3f", *p.Hashrate15m / *p.PoolHashrate * 100)
	}
	fmt.Fprintf(&b, "  pool %s   your share ≈ %s %%   %s\n", hs(p.PoolHashrate), ansi.White(share), age(s, status.SrcP2PoolPool))

	fmt.Fprintf(&b, "\n  %s   %s\n", ansi.Orange("PAYOUTS"), m.payoutsLine())

	if m.prev != nil && m.prev.XMRig.Rejected != nil && x.Rejected != nil && *x.Rejected > *m.prev.XMRig.Rejected {
		b.WriteString("\n  " + ansi.Red("warning: rejected submissions increased since the previous sample") + "\n") // H-04
	}
	issues := 0
	for _, i := range s.Health.Issues {
		if i.Severity != "info" {
			if issues == 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "  %s %-24s %s\n", ansi.Severity(fmt.Sprintf("%-7s", i.Severity)), i.Code, i.Message)
			issues++
			if issues == 5 {
				break
			}
		}
	}
	if m.opResult != "" {
		fmt.Fprintf(&b, "\n  %s\n", ansi.White(m.opResult))
	}
	if d := m.dialog(); d != "" {
		b.WriteString("\n" + d)
	}
	return m.fit(b.String(), w)
}

// fit pads or trims the body so the key bar sits on the last terminal row
// (htop-style); over-long lines are clipped by the renderer.
func (m Model) fit(body string, w int) string {
	if m.height > 0 {
		if lines := strings.Split(body, "\n"); len(lines) > m.height-1 {
			body = strings.Join(lines[:m.height-1], "\n") + "\n"
		}
		for strings.Count(body, "\n") < m.height-1 {
			body += "\n"
		}
	}
	return body + ansi.Banner(pad(" q quit  r refresh  s services  l logs  p payouts  ? help", w))
}

// payoutsLine is the one-line dashboard summary of the journal payouts.
func (m Model) payoutsLine() string {
	switch {
	case m.pay == nil && m.payErr != "":
		return "error: " + m.payErr
	case m.pay == nil:
		return "loading…"
	case len(m.pay.Payouts) == 0:
		return fmt.Sprintf("%s payouts yet   %d pool blocks without a share of yours", ansi.White("no"), m.pay.BlocksWithout)
	}
	last := m.pay.Payouts[len(m.pay.Payouts)-1]
	at := "unknown time"
	if !last.At.IsZero() {
		at = last.At.Local().Format("2006-01-02 15:04")
	}
	stale := ""
	if m.payErr != "" {
		stale = "   " + ansi.Red("stale: "+m.payErr)
	}
	return fmt.Sprintf("%s payouts   total %s XMR   last %s +%s   %d pool blocks without a share%s",
		ansi.White(fmt.Sprint(len(m.pay.Payouts))), ansi.White(m.pay.TotalXMR), at, last.XMR, m.pay.BlocksWithout, stale)
}

func pad(s string, w int) string {
	if n := w - len([]rune(s)); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func (m Model) historyMax() float64 {
	max := 0.0
	for _, p := range m.history {
		if p.v != nil && *p.v > max {
			max = *p.v
		}
	}
	return max
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
		return "  Keys: q/Ctrl-C quit (miners keep running)  r refresh  s start/stop/restart  l journalctl -f (Ctrl-C returns)  p payouts  Esc close\n  Colour (Monero orange/white) only highlights; every state is written as text. NO_COLOR disables it.\n"
	case modePayouts:
		return m.payoutsView()
	}
	return ""
}

// payoutsView lists the last payouts from the P2Pool journal in local time.
func (m Model) payoutsView() string {
	var b strings.Builder
	b.WriteString("  Payouts (P2Pool journal)\n")
	switch {
	case m.payErr != "":
		b.WriteString("    error: " + m.payErr + "\n")
	case m.pay == nil:
		b.WriteString("    loading…\n")
	case len(m.pay.Payouts) == 0:
		fmt.Fprintf(&b, "    none yet; %d pool blocks found without a share of yours in the PPLNS window\n", m.pay.BlocksWithout)
	default:
		rows := m.pay.Payouts
		if len(rows) > 12 {
			rows = rows[len(rows)-12:]
		}
		fmt.Fprintf(&b, "    %-20s %-16s %s\n", "TIME (local)", "XMR", "BLOCK")
		for _, p := range rows {
			at := "unknown"
			if !p.At.IsZero() {
				at = p.At.Local().Format("2006-01-02 15:04:05")
			}
			fmt.Fprintf(&b, "    %-20s %-16s %d\n", at, p.XMR, p.Block)
		}
		fmt.Fprintf(&b, "    %d payouts, total %s XMR; %d pool blocks without a payout\n", len(m.pay.Payouts), m.pay.TotalXMR, m.pay.BlocksWithout)
	}
	b.WriteString("  (r reload, Esc close)\n")
	return b.String()
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
	var b, g strings.Builder
	for _, p := range pts {
		switch {
		case p.v == nil:
			g.WriteRune(' ')
		case *p.v == 0:
			g.WriteRune('_')
		default:
			i := int(*p.v / max * float64(len(ramp)-1))
			g.WriteRune(ramp[i])
		}
	}
	b.WriteString("10 min [" + ansi.Orange(g.String()) + strings.Repeat(" ", width-len(pts)))
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
		return "[" + src.State + "]" // live HTTP sources carry no file age
	}
	return "[" + src.State + " " + dur(src.AgeSeconds) + " old]"
}

// hs formats a large hashrate with an SI prefix for the pool line only.
func hs(v *float64) string {
	switch {
	case v == nil:
		return "— H/s"
	case *v >= 1e9:
		return fmt.Sprintf("%.2f GH/s", *v/1e9)
	case *v >= 1e6:
		return fmt.Sprintf("%.2f MH/s", *v/1e6)
	case *v >= 1e4:
		return fmt.Sprintf("%.1f kH/s", *v/1e3)
	}
	return fmt.Sprintf("%.0f H/s", *v)
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
