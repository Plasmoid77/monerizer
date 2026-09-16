package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Plasmoid77/monerizer/internal/config"
	"github.com/Plasmoid77/monerizer/internal/payouts"
	"github.com/Plasmoid77/monerizer/internal/status"
)

func model() Model {
	cfg := &config.Config{}
	cfg.Services.P2Pool, cfg.Services.XMRig = "p.service", "x.service"
	cfg.UI.RefreshMs = 1000
	return New(cfg, nil)
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		next, _ := m.key(k)
		m = next.(Model)
	}
	return m
}

func TestMenuCancelIsDefault(t *testing.T) {
	m := press(model(), "s", "enter", "j", "enter")
	if m.mode != modeConfirm || m.confirmOK || actions[m.action] != "stop" {
		t.Fatalf("mode %v ok %v action %s", m.mode, m.confirmOK, actions[m.action])
	}
	m = press(m, "enter")
	if m.mode != modeDashboard || m.busy {
		t.Fatal("Enter on Cancel must not launch")
	}
	m = press(model(), "s", "j", "j", "enter", "j", "j", "enter", "right", "enter")
	if !m.busy || m.opResult != "restart xmrig in progress…" {
		t.Fatalf("busy %v result %q", m.busy, m.opResult)
	}
	if u := m.units(); len(u) != 1 || u[0] != "x.service" {
		t.Fatalf("units %v", u)
	}
	if m2 := press(m, "s"); m2.mode != modeDashboard {
		t.Fatal("menu must stay closed while an operation is in flight")
	}
}

func TestQuitNeverControls(t *testing.T) {
	m := model()
	for _, k := range []string{"q", "ctrl+c"} {
		_, cmd := m.key(k)
		if cmd == nil || cmd() != (tea.QuitMsg{}) {
			t.Fatalf("%s must quit", k)
		}
	}
}

func TestHistoryGapsAndBounds(t *testing.T) {
	m := model()
	base := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	snap := func(i int, inv string, hr *float64) *status.Snapshot {
		s := &status.Snapshot{CollectedAt: base.Add(time.Duration(i) * time.Second), Sources: map[string]status.Source{status.SrcXMRigSummary: {State: status.StateOK}}}
		s.Services.XMRig.InvocationID = inv
		s.XMRig.Hashrate10s = hr
		return s
	}
	v := 100.0
	for i := 0; i <= 1300; i++ {
		m.snap = snap(i, "a", &v)
		m.record()
	}
	if len(m.history) > historyLen || m.history[0].t.Before(base.Add(1300*time.Second-historySpan)) {
		t.Fatalf("history not bounded: %d", len(m.history))
	}
	m.snap = snap(1301, "b", &v)
	m.record()
	h := m.history
	if h[len(h)-2].v != nil || h[len(h)-1].v == nil {
		t.Fatal("new invocation must insert a gap before the fresh point")
	}
	zero := 0.0
	m.snap = snap(1302, "b", &zero)
	m.record()
	if p := m.history[len(m.history)-1]; p.v == nil || *p.v != 0 {
		t.Fatal("a real zero must be recorded as zero, not as a gap")
	}
	m.width = 100
	if line := m.sparkline(40); len([]rune(line)) == 0 || !contains(line, "_") {
		t.Fatalf("sparkline %q", line)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestPayoutsView(t *testing.T) {
	m := model()
	m.mode = modePayouts
	if v := m.payoutsView(); !contains(v, "loading") {
		t.Fatalf("loading state: %q", v)
	}
	m.pay = &payouts.Report{Payouts: []payouts.Payout{{At: time.Date(2026, 9, 16, 0, 47, 51, 0, time.UTC), XMR: "0.001234567890", Block: 3763276}}, TotalXMR: "0.001234567890", BlocksWithout: 2}
	v := m.payoutsView()
	if !contains(v, "0.001234567890") || !contains(v, "3763276") || !contains(v, "2 pool blocks without") || !contains(v, time.Date(2026, 9, 16, 0, 47, 51, 0, time.UTC).Local().Format("2006-01-02 15:04:05")) {
		t.Fatalf("view %q", v)
	}
	m2 := press(m, "esc")
	if m2.mode != modeDashboard {
		t.Fatal("esc must close payouts")
	}
}
