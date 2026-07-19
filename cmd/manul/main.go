// manul is a terminal browser for the markdown web (llms.txt and .md pages).
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/denrou/manul/internal/bookmarks"
	"github.com/denrou/manul/internal/config"
	"github.com/denrou/manul/internal/discover"
	"github.com/denrou/manul/internal/ui"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "manul:", err)
		os.Exit(1)
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("manul " + version)
		return nil
	}

	var initialStatus string
	cfg, err := config.Load()
	if err != nil {
		// The stderr line disappears under the altscreen within
		// milliseconds; the statusbar note is what the user will see.
		fmt.Fprintln(os.Stderr, "manul: config:", err, "(using defaults)")
		initialStatus = "config ignored: " + err.Error()
		cfg = config.Defaults()
	}

	// The glamour style is resolved once, before the TUI takes over the
	// terminal: background detection is unreliable afterwards. Building
	// a throwaway renderer validates a custom style_path up front.
	style := ui.ResolveStyle(cfg)
	if _, err := glamour.NewTermRenderer(style); err != nil {
		return fmt.Errorf("invalid style: %w", err)
	}

	var store *bookmarks.Store
	if dir, err := config.Dir(); err == nil {
		store = bookmarks.New(dir)
	}

	initialURL := flag.Arg(0)
	if initialURL == "" {
		initialURL = cfg.Home // empty Home keeps the built-in start page
	}

	model := ui.New(ui.Options{
		Config:        cfg,
		Style:         style,
		Resolver:      &discover.Resolver{},
		Bookmarks:     store,
		InitialURL:    initialURL,
		InitialStatus: initialStatus,
	})

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.Mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	_, err = tea.NewProgram(model, opts...).Run()
	return err
}
