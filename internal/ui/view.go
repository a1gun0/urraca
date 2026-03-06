package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"urraca/internal/engine"
)

var (
	cyan      = lipgloss.Color("45")
	brightCyn = lipgloss.Color("51")
	red       = lipgloss.Color("196")
	dim       = lipgloss.Color("240")
	white     = lipgloss.Color("255")

	baseStyle = lipgloss.NewStyle().Foreground(cyan)
	boxStyle  = lipgloss.NewStyle().
			Foreground(cyan).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(brightCyn).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(brightCyn).
			Bold(true)

	alertStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(dim)

	headerStyle = lipgloss.NewStyle().
			Foreground(brightCyn).
			Bold(true)
)

const splash = `
██╗   ██╗██████╗ ██████╗  █████╗  ██████╗ █████╗ 
██║   ██║██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗
██║   ██║██████╔╝██████╔╝███████║██║     ███████║
██║   ██║██╔══██╗██╔══██╗██╔══██║██║     ██╔══██║
╚██████╔╝██║  ██║██║  ██║██║  ██║╚██████╗██║  ██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝
`

func renderHeader(width int, stage string, findings int, queued int, running bool) string {
	status := "STOP"
	if running {
		status = "RUN"
	}
	meta := fmt.Sprintf(" stage=%s | findings=%d | queued=%d | status=%s | q salir | ↑↓ navegar ", stage, findings, queued, status)
	top := headerStyle.Width(width).Render(splash)
	bottom := baseStyle.Width(width).Render(meta)
	return top + "\n" + bottom
}

func renderPipelinePanel(width, height int, stage string, queue []engine.Job, events []engine.Event) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("PIPELINE / SCHEDULER"))
	b.WriteString("\n")
	b.WriteString(baseStyle.Render("stage actual: " + stage))
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("cola"))
	b.WriteString("\n")
	limit := min(8, len(queue))
	if limit == 0 {
		b.WriteString(dimStyle.Render("sin jobs pendientes"))
		b.WriteString("\n")
	}
	for i := 0; i < limit; i++ {
		j := queue[i]
		line := fmt.Sprintf("[%02d] %-10s pri=%03d %s", i+1, j.Stage, j.Priority, ago(j.CreatedAt))
		b.WriteString(baseStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("eventos"))
	b.WriteString("\n")
	start := 0
	if len(events) > 8 {
		start = len(events) - 8
	}
	for _, ev := range events[start:] {
		line := fmt.Sprintf("%-7s %s", string(ev.Kind), ev.Message)
		style := baseStyle
		if ev.Kind == engine.EventFinding {
			style = alertStyle
		}
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	content := fitLines(b.String(), height-2)
	return boxStyle.Width(width).Height(height).Render(content)
}

func renderFindingsPanel(width, height int, findings []engine.Finding, selected int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("HALLAZGOS"))
	b.WriteString("\n")

	if len(findings) == 0 {
		b.WriteString(dimStyle.Render("sin hallazgos todavía"))
		b.WriteString("\n")
	} else {
		limit := min(12, len(findings))
		for i := 0; i < limit; i++ {
			f := findings[i]
			prefix := " "
			style := baseStyle
			if i == selected {
				prefix = ">"
			}
			if f.Confidence >= 70 || f.Severity == "high" || f.Severity == "critical" {
				style = alertStyle
			}
			line1 := fmt.Sprintf("%s [%s/%s] %s", prefix, f.Module, f.Subtype, f.Value)
			line2 := fmt.Sprintf("  conf=%d status=%d sev=%s", f.Confidence, f.Status, f.Severity)
			b.WriteString(style.Render(line1))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(line2))
			b.WriteString("\n")
		}
	}

	content := fitLines(b.String(), height-2)
	return boxStyle.Width(width).Height(height).Render(content)
}

func renderDetailPanel(width, height int, findings []engine.Finding, selected int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("DETALLE"))
	b.WriteString("\n")

	if len(findings) == 0 || selected >= len(findings) {
		b.WriteString(dimStyle.Render("todavía no hay detalle disponible"))
		b.WriteString("\n")
	} else {
		f := findings[selected]
		lines := []string{
			fmt.Sprintf("target:     %s", f.Target),
			fmt.Sprintf("module:     %s", f.Module),
			fmt.Sprintf("category:   %s/%s", f.Category, f.Subtype),
			fmt.Sprintf("url:        %s", f.URL),
			fmt.Sprintf("status:     %d", f.Status),
			fmt.Sprintf("confidence: %d", f.Confidence),
			fmt.Sprintf("severity:   %s", f.Severity),
			fmt.Sprintf("timestamp:  %s", f.Timestamp.Format("2006-01-02 15:04:05")),
			"",
			"evidence:",
			f.Evidence,
		}
		for _, line := range lines {
			style := baseStyle
			if strings.Contains(strings.ToLower(line), "confidence: 9") || strings.Contains(strings.ToLower(line), "severity:   critical") {
				style = alertStyle
			}
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	content := fitLines(b.String(), height-2)
	return boxStyle.Width(width).Height(height).Render(content)
}

func fitLines(s string, height int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
