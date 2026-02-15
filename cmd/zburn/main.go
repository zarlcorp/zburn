package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zapp"
	"github.com/zarlcorp/zburn/internal/tui"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	app := zapp.New(zapp.WithName("zburn"))

	ctx, cancel := zapp.SignalContext(context.Background())
	defer cancel()

	if len(os.Args) > 1 {
		runCLI(ctx, os.Args[1])
		_ = app.Close()
		return
	}

	if err := runTUI(); err != nil {
		slog.Error("tui", "err", err)
		_ = app.Close()
		os.Exit(1)
	}

	if err := app.Close(); err != nil {
		slog.Error("shutdown", "err", err)
		os.Exit(1)
	}
}

func runCLI(_ context.Context, cmd string) {
	switch cmd {
	case "version":
		fmt.Printf("zburn %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "zburn: unknown command %q\n", cmd)
		os.Exit(1)
	}
}

func runTUI() error {
	p := tea.NewProgram(tui.New(version))
	_, err := p.Run()
	return err
}
