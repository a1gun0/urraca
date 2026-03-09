package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/urraca/internal/engine"
)

type tickMsg time.Time

type Model struct {
	engine       *engine.Engine
	ctx          context.Context
	cancel       context.CancelFunc
	width        int
	height       int
	stage        string
	findings     []engine.Finding
	queue        []engine.Job
	events       []engine.Event
	booted       bool
	selectedFind int
}

func NewModel(eng *engine.Engine) Model {
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		engine: eng,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.startEngine(), tickCmd())
}

func (m Model) startEngine() tea.Cmd {
	return func() tea.Msg {
		if m.engine.Running() {
			return nil
		}
		// engine runs in background; snapshot will be polled by tickCmd
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// we can't notify directly here, but engine.PushEvent could be used
				}
			}()
			m.engine.Start(m.ctx, func(ev engine.Event) {
				// intentionally empty; View is built from Snapshot
			})
		}()
		return tickMsg(time.Now())
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(220*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.stage, m.findings, m.queue, m.events = m.engine.Snapshot()
		sort.SliceStable(m.findings, func(i, j int) bool {
			return m.findings[i].Timestamp.After(m.findings[j].Timestamp)
		})
		if m.selectedFind >= len(m.findings) {
			m.selectedFind = max(0, len(m.findings)-1)
		}
		return m, tickCmd()
		case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancel()
			return m, tea.Quit
		case "j", "down":
			if m.selectedFind < len(m.findings)-1 {
				m.selectedFind++
			}
		case "k", "up":
			if m.selectedFind > 0 {
				m.selectedFind--
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "cargando URRACA..."
	}

	topHeight := int(float64(m.height-7) * 0.55)
	if topHeight < 12 {
		topHeight = 12
	}
	bottomHeight := m.height - topHeight - 4

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 1

	header := renderHeader(m.width, m.stage, len(m.findings), len(m.queue), m.engine.Running())
	top := joinHorizontal(
		renderPipelinePanel(leftWidth, topHeight, m.stage, m.queue, m.events),
		renderFindingsPanel(rightWidth, topHeight, m.findings, m.selectedFind),
	)
	bottom := renderDetailPanel(m.width, bottomHeight, m.findings, m.selectedFind)

	return strings.Join([]string{header, top, bottom}, "\n")
}

func joinHorizontal(left, right string) string {
	ls := strings.Split(left, "\n")
	rs := strings.Split(right, "\n")
	maxLines := max(len(ls), len(rs))
	for len(ls) < maxLines {
		ls = append(ls, "")
	}
	for len(rs) < maxLines {
		rs = append(rs, "")
	}
	out := make([]string, 0, maxLines)
	for i := 0; i < maxLines; i++ {
		out = append(out, ls[i]+" "+rs[i])
	}
	return strings.Join(out, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ago(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "ahora"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}
