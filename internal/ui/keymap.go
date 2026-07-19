package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds the bindings manul handles itself. Plain scrolling keys
// (j/k/arrows/space/b/d/u/pgup/pgdn) stay on the viewport's defaults.
type keyMap struct {
	NextLink   key.Binding
	PrevLink   key.Binding
	Follow     key.Binding
	Back       key.Binding
	Forward    key.Binding
	Prompt     key.Binding
	Pipe       key.Binding
	Search     key.Binding
	SearchNext key.Binding
	SearchPrev key.Binding
	Hint       key.Binding
	Open       key.Binding
	Yank       key.Binding
	Bookmark   key.Binding
	Start      key.Binding
	Reload     key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Help       key.Binding
	Quit       key.Binding
	Cancel     key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		NextLink:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next link")),
		PrevLink:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous link")),
		Follow:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "follow link")),
		Back:       key.NewBinding(key.WithKeys("backspace", "[", "B", "H"), key.WithHelp("H", "back")),
		Forward:    key.NewBinding(key.WithKeys("]", "L"), key.WithHelp("L", "forward")),
		Prompt:     key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "open url")),
		Pipe:       key.NewBinding(key.WithKeys("|"), key.WithHelp("|", "pipe page")),
		Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		SearchNext: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		SearchPrev: key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "previous match")),
		Hint:       key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "link hints")),
		Open:       key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		Yank:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yank url")),
		Bookmark:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add bookmark")),
		Start:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start page")),
		Reload:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Cancel:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

// ShortHelp is the statusbar help line. Back is included because an
// invisible back button reads as a missing one (friction log, day one).
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Prompt, k.NextLink, k.Back, k.Help, k.Quit}
}

// FullHelp satisfies help.KeyMap; the full reference lives on the
// embedded help page ('?').
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextLink, k.PrevLink, k.Follow, k.Hint},
		{k.Back, k.Forward, k.Prompt, k.Reload},
		{k.Search, k.SearchNext, k.SearchPrev},
		{k.Pipe, k.Open, k.Yank, k.Bookmark, k.Start},
		{k.Top, k.Bottom, k.Help, k.Quit},
	}
}
