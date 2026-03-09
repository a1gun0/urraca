package app

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/urraca/internal/engine"
	"github.com/yourusername/urraca/internal/pipeline"
	"github.com/yourusername/urraca/internal/ui"
)

func Run(ctx context.Context, args []string) error {
	// simple flag parsing so we can add options later (timeout, max depth...)
	fs := flag.NewFlagSet("urraca", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	target := fs.String("target", "", "url del objetivo (http/https)")
	// future options example
	// timeout := fs.Duration("timeout", 8*time.Second, "timeout de las peticiones")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *target == "" {
		return errors.New("uso: urraca --target <url>\npasar --help para más información")
	}

	// algunas distribuciones minimalistas (Kali en contenedores, por ejemplo)
	// no exportan TERM; bubbletea requiere un valor válido.
	if os.Getenv("TERM") == "" {
		oos.Setenv("TERM", "xterm-256color")
	}

	// normalizar esquema si el usuario no lo proporciona
	if !strings.HasPrefix(*target, "http://") && !strings.HasPrefix(*target, "https://") {
		*target = "http://" + *target
	}

	if _, err := url.ParseRequestURI(*target); err != nil {
		return fmt.Errorf("target inválido: %w", err)
	}

	cfg := engine.DefaultConfig(*target)
	eng := engine.New(cfg, pipeline.Default())
	model := ui.NewModel(eng)

	// bubbletea program uses its own context; we cancel when the supplied ctx
	// is done.
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}
