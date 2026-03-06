package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"urraca/internal/engine"
	"urraca/internal/pipeline"
	"urraca/internal/ui"
)

func Run(args []string) error {
	if len(args) == 0 {
		return errors.New("uso: urraca <target>")
	}

	target := args[0]
	if _, err := url.ParseRequestURI(target); err != nil {
		return fmt.Errorf("target inválido: %w", err)
	}

	cfg := engine.DefaultConfig(target)
	eng := engine.New(cfg, pipeline.Default())
	model := ui.NewModel(eng)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}
