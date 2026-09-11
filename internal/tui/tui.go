// Package tui is the interactive panel (ТЗ §12). It renders the same Snapshot
// the CLI prints and never owns the miners' lifetime.
package tui

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Plasmoid77/monerizer/internal/config"
	"github.com/Plasmoid77/monerizer/internal/status"
	"github.com/Plasmoid77/monerizer/internal/systemd"
)

const (
	minWidth, minHeight = 80, 24
	historyLen          = 1200
	historySpan         = 10 * time.Minute
)

type mode int

const (
	modeDashboard mode = iota
	modeTarget         // choose all|p2pool|xmrig for control or logs
	modeAction         // choose start|stop|restart
	modeConfirm
	modeHelp
)

var (
	targets = []string{"all", "p2pool", "xmrig"}
	actions = []string{"start", "stop", "restart"}
)

type point struct {
	t time.Time
	v *float64 // nil = gap
}

type Model struct {
	cfg        *config.Config
	collector  *status.Collector
	run        systemd.Runner
	snap       *status.Snapshot
	prev       *status.Snapshot
	err        string
	width      int
	height     int
	mode       mode
	forLogs    bool
	target     int
	action     int
	confirmOK  bool
	collecting bool
	busy       bool
	opResult   string
	history    []point
	lastInv    string
	ascii      bool
}

type snapshotMsg *status.Snapshot
type tickMsg time.Time
type opMsg struct{ text string }
type logsMsg struct{ err error }

func New(cfg *config.Config, c *status.Collector) Model {
	lang := strings.ToUpper(os.Getenv("LC_ALL") + os.Getenv("LC_CTYPE") + os.Getenv("LANG"))
	return Model{cfg: cfg, collector: c, run: systemd.ExecRunner, ascii: !strings.Contains(lang, "UTF-8") && !strings.Contains(lang, "UTF8")}
}

// Run starts the program; the caller checks for a TTY beforehand.
func Run(m Model) error {
	_, err := tea.NewProgram(m).Run()
	return err
}

func (m Model) Init() tea.Cmd { return tea.Batch(m.collect(), m.tick()) }

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Duration(m.cfg.UI.RefreshMs)*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// refresh starts a collection unless one is already running (DATA-01).
func (m *Model) refresh() tea.Cmd {
	if m.collecting {
		return nil
	}
	m.collecting = true
	return m.collect()
}

func (m Model) collect() tea.Cmd {
	c := m.collector
	return func() tea.Msg { return snapshotMsg(c.Collect(context.Background())) }
}

func (m Model) control(verb string, units []string) tea.Cmd {
	run := m.run
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := systemd.Control(ctx, run, verb, units...); err != nil {
			return opMsg{text: verb + " failed: " + err.Error()}
		}
		return opMsg{text: verb + " " + strings.Join(units, " ") + ": done"}
	}
}

func (m Model) units() []string {
	switch targets[m.target] {
	case "p2pool":
		return []string{m.cfg.Services.P2Pool}
	case "xmrig":
		return []string{m.cfg.Services.XMRig}
	}
	return []string{m.cfg.Services.P2Pool, m.cfg.Services.XMRig}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		if m.collecting {
			return m, m.tick() // DATA-01: skipped ticks do not queue
		}
		m.collecting = true
		return m, tea.Batch(m.collect(), m.tick())
	case snapshotMsg:
		m.collecting = false
		m.prev, m.snap = m.snap, (*status.Snapshot)(msg)
		m.record()
		return m, nil
	case opMsg:
		m.busy = false
		m.opResult = msg.text
		return m, m.refresh()
	case logsMsg:
		signal.Reset(os.Interrupt)
		return m, m.refresh()
	case tea.KeyPressMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m Model) key(k string) (tea.Model, tea.Cmd) {
	if k == "ctrl+c" || (k == "q" && m.mode == modeDashboard) {
		return m, tea.Quit
	}
	switch m.mode {
	case modeDashboard:
		switch k {
		case "r":
			if !m.collecting {
				m.collecting = true
				return m, m.collect()
			}
		case "s":
			if !m.busy {
				m.mode, m.forLogs, m.target = modeTarget, false, 0
			}
		case "l":
			m.mode, m.forLogs, m.target = modeTarget, true, 0
		case "?":
			m.mode = modeHelp
		}
	case modeTarget:
		switch k {
		case "esc", "q":
			m.mode = modeDashboard
		case "up", "k":
			m.target = (m.target + len(targets) - 1) % len(targets)
		case "down", "j":
			m.target = (m.target + 1) % len(targets)
		case "enter":
			if m.forLogs {
				m.mode = modeDashboard
				return m, m.logs()
			}
			m.mode, m.action = modeAction, 0
		}
	case modeAction:
		switch k {
		case "esc", "q":
			m.mode = modeDashboard
		case "up", "k":
			m.action = (m.action + len(actions) - 1) % len(actions)
		case "down", "j":
			m.action = (m.action + 1) % len(actions)
		case "enter":
			if actions[m.action] == "start" {
				return m.launch()
			}
			m.mode, m.confirmOK = modeConfirm, false // UI-03: Cancel is selected by default
		}
	case modeConfirm:
		switch k {
		case "esc", "q", "n":
			m.mode = modeDashboard
		case "left", "right", "tab", "h":
			m.confirmOK = !m.confirmOK
		case "enter":
			if m.confirmOK {
				return m.launch()
			}
			m.mode = modeDashboard
		}
	case modeHelp:
		m.mode = modeDashboard
	}
	return m, nil
}

func (m Model) launch() (tea.Model, tea.Cmd) {
	m.mode = modeDashboard
	if m.busy {
		return m, nil // SYS-09: one control operation at a time
	}
	m.busy = true
	m.opResult = actions[m.action] + " " + targets[m.target] + " in progress…"
	return m, m.control(actions[m.action], m.units())
}

func (m Model) logs() tea.Cmd {
	// Ctrl-C must end journalctl only. A Go handler (not SIG_IGN, which exec
	// would inherit) keeps this process alive; the child gets the default action.
	signal.Notify(make(chan os.Signal, 1), os.Interrupt)
	cmd := exec.Command("journalctl", systemd.JournalArgs(m.units(), 200, true)...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return logsMsg{err: err} })
}

// record appends a hashrate point (DATA-09) with a gap on a new service session (DATA-08).
func (m *Model) record() {
	s := m.snap
	inv := s.Services.XMRig.InvocationID
	if m.lastInv != "" && inv != m.lastInv {
		m.history = append(m.history, point{t: s.CollectedAt})
	}
	m.lastInv = inv
	if s.Sources[status.SrcXMRigSummary].State == status.StateOK && s.Sources[status.SrcXMRigSummary].ErrorCode == "" {
		m.history = append(m.history, point{t: s.CollectedAt, v: s.XMRig.Hashrate10s})
	} else {
		m.history = append(m.history, point{t: s.CollectedAt})
	}
	cut := s.CollectedAt.Add(-historySpan)
	for len(m.history) > historyLen || (len(m.history) > 0 && m.history[0].t.Before(cut)) {
		m.history = m.history[1:]
	}
}
