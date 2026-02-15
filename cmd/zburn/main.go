package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zarlcorp/core/pkg/zapp"
	"github.com/zarlcorp/zburn/internal/cli"
	"github.com/zarlcorp/zburn/internal/identity"
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
	case "email":
		cli.CmdEmail()
	case "identity":
		cli.CmdIdentity(os.Args[2:])
	case "list":
		cli.CmdList(os.Args[2:])
	case "forget":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: zburn forget <id>")
			os.Exit(1)
		}
		cli.CmdForget(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "zburn: unknown command %q\n", cmd)
		os.Exit(1)
	}
}

func runTUI() error {
	dataDir := cli.DataDir()
	gen := identity.New()
	firstRun := cli.IsFirstRun(dataDir)

	m := tui.New(version, dataDir, gen, firstRun)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if fm, ok := finalModel.(tui.Model); ok {
		fm.Close()
	}

	return nil
}
