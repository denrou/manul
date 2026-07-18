// manul is a terminal browser for the markdown web (llms.txt and .md pages).
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/denrou/manul/internal/fetch"
	"github.com/denrou/manul/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "manul:", err)
		os.Exit(1)
	}
}

func run() error {
	title, content := "welcome", ui.WelcomeMarkdown

	if len(os.Args) > 1 {
		target, err := fetch.Normalize(os.Args[1])
		if err != nil {
			return err
		}
		content, err = fetch.Markdown(context.Background(), target)
		if err != nil {
			return err
		}
		title = target
	}

	program := tea.NewProgram(ui.New(title, content), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
