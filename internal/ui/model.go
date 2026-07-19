// Package ui implements manul's terminal interface: an async-loading
// markdown viewport with numbered links, history, bookmarks, and a URL
// prompt, themed through glamour.
package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aymanbagabas/go-osc52/v2"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/denrou/manul/internal/bookmarks"
	"github.com/denrou/manul/internal/config"
	"github.com/denrou/manul/internal/discover"
	"github.com/denrou/manul/internal/doc"
)

// resizeDebounce delays re-renders while the terminal is being resized.
const resizeDebounce = 100 * time.Millisecond

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "252"}).
	Background(lipgloss.AdaptiveColor{Light: "252", Dark: "236"}).
	Padding(0, 1)

// page is the currently displayed document plus its render artifacts.
type page struct {
	url       string
	markdown  string // raw source; Truncate runs in the render pipeline
	doc       doc.Document
	rendered  string // cached glamour output with all link markers
	truncated bool
}

// Options configures a Model. Style must be resolved in main() before
// the TUI starts (see ResolveStyle).
type Options struct {
	Config     config.Config
	Style      glamour.TermRendererOption
	Resolver   *discover.Resolver
	Bookmarks  *bookmarks.Store
	InitialURL string

	// InitialStatus seeds the statusbar with a startup notice (for
	// example a rejected config file) that the altscreen would
	// otherwise hide; cleared on the first keypress like any status.
	InitialStatus string
}

// Model is the root bubbletea model.
type Model struct {
	cfg        config.Config
	styleOpt   glamour.TermRendererOption
	resolver   *discover.Resolver
	bookmarks  *bookmarks.Store
	initialURL string

	width, height int
	ready         bool

	renderer      *glamour.TermRenderer
	rendererWidth int
	resizeSeq     int

	page        page
	back        []histEntry
	forward     []histEntry
	replaceNext bool // reload: next load replaces the page, no history push

	viewport   viewport.Model
	prompt     textinput.Model
	promptOpen bool
	promptKind promptKind
	spin       spinner.Model
	help       help.Model
	keys       keyMap

	navSeq      int
	navCtx      context.Context // current navigation's context; nil when idle
	cancel      context.CancelFunc
	loading     bool
	loadingHost string

	selected    int // Index of the selected link, 0 = none
	markerLines map[int]int
	markerLocs  map[int]markerLoc
	digits      string

	status string // transient; cleared on keypress or successful load
}

func New(opts Options) Model {
	resolver := opts.Resolver
	if resolver == nil {
		resolver = &discover.Resolver{}
	}
	style := opts.Style
	if style == nil {
		style = ResolveStyle(opts.Config)
	}
	prompt := textinput.New()
	prompt.Prompt = ":"
	prompt.Placeholder = "url or domain"
	m := Model{
		cfg:        opts.Config,
		styleOpt:   style,
		resolver:   resolver,
		bookmarks:  opts.Bookmarks,
		initialURL: opts.InitialURL,
		prompt:     prompt,
		spin:       spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		help:       help.New(),
		keys:       defaultKeyMap(),
		status:     opts.InitialStatus,
	}
	m.page = page{url: startURL, markdown: m.startMarkdown()}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.initialURL == "" {
		return nil
	}
	target := m.initialURL
	return func() tea.Msg { return navigateToMsg{url: target} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case resizeTickMsg:
		if msg.seq == m.resizeSeq {
			m.applyRender()
		}
		return m, nil

	case navigateToMsg:
		return m, m.startNavigate(msg.url)

	case docLoadedMsg:
		if msg.seq != m.navSeq {
			return m, nil // stale fetch, superseded by a newer navigation
		}
		m.loading = false
		m.cancel = nil
		m.status = ""
		if m.replaceNext {
			m.replaceNext = false
			frac := m.currentFraction()
			m.setPage(msg.url, msg.markdown)
			m.viewport.SetYOffset(restoreOffset(frac, m.viewport.TotalLineCount(), m.viewport.Height))
		} else {
			m.showDocument(msg.url, msg.markdown)
		}
		return m, nil

	case loadFailedMsg:
		if msg.seq != m.navSeq {
			return m, nil
		}
		m.loading = false
		m.cancel = nil
		replace := m.replaceNext
		m.replaceNext = false
		var notMD *discover.NotMarkdownError
		switch {
		case errors.As(msg.err, &notMD):
			if replace {
				m.setPage(notMD.URL, fallbackMarkdown(notMD))
			} else {
				m.showDocument(notMD.URL, fallbackMarkdown(notMD))
			}
		case errors.Is(msg.err, context.Canceled):
			m.status = "load canceled"
		default:
			m.status = "error: " + msg.err.Error()
		}
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case pipeDoneMsg:
		m.status = ""
		m.showDocument(pipeURL, msg.page)
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmds []tea.Cmd
	if m.promptOpen {
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		cmds = append(cmds, cmd)
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	first := !m.ready
	m.width, m.height = msg.Width, msg.Height
	if first {
		m.viewport = viewport.New(msg.Width, max(0, msg.Height-chromeHeight(m.promptOpen)))
		m.ready = true
	} else {
		m.syncViewportSize()
	}
	m.prompt.Width = max(10, msg.Width-4)
	m.help.Width = msg.Width
	if first {
		m.applyRender()
		return m, nil
	}
	if clampWidth(msg.Width, m.cfg.MaxWidth) == m.rendererWidth {
		return m, nil // wrap width unchanged: skip the re-render
	}
	m.resizeSeq++
	seq := m.resizeSeq
	return m, tea.Tick(resizeDebounce, func(time.Time) tea.Msg { return resizeTickMsg{seq: seq} })
}

// handleKey routes key presses. While the URL prompt is focused, every
// key goes to the prompt except Enter, Esc, and ctrl+c.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = "" // transient messages last until the next keypress

	if m.promptOpen {
		switch msg.String() {
		case "ctrl+c":
			m.cancelNav()
			return m, tea.Quit
		case "esc":
			m.closePrompt()
			return m, nil
		case "enter":
			input := strings.TrimSpace(m.prompt.Value())
			kind := m.promptKind
			m.closePrompt()
			if input == "" {
				return m, nil
			}
			if kind == promptPipe {
				m.status = "running: " + input
				return m, runPipe(input, m.page.markdown, m.page.url)
			}
			return m, m.startNavigate(input)
		default:
			var cmd tea.Cmd
			m.prompt, cmd = m.prompt.Update(msg)
			return m, cmd
		}
	}

	if s := msg.String(); len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
		if len(m.digits) < 6 {
			m.digits += s
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		m.cancelNav()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Cancel):
		m.handleEsc()
		return m, nil

	case key.Matches(msg, m.keys.Follow):
		if m.digits != "" {
			n, err := strconv.Atoi(m.digits)
			m.digits = ""
			if err != nil {
				return m, nil
			}
			return m, m.followIndex(n)
		}
		if m.selected != 0 {
			return m, m.followIndex(m.selected)
		}
		// No selection, no digits: fall through to the viewport default.

	case key.Matches(msg, m.keys.NextLink):
		m.selectLink(true)
		return m, nil

	case key.Matches(msg, m.keys.PrevLink):
		m.selectLink(false)
		return m, nil

	case key.Matches(msg, m.keys.Back):
		m.goBack()
		return m, nil

	case key.Matches(msg, m.keys.Forward):
		m.goForward()
		return m, nil

	case key.Matches(msg, m.keys.Prompt):
		return m, m.openPrompt(promptGoto)

	case key.Matches(msg, m.keys.Pipe):
		return m, m.openPrompt(promptPipe)

	case key.Matches(msg, m.keys.Open):
		return m, m.openCurrent()

	case key.Matches(msg, m.keys.Yank):
		m.yank()
		return m, nil

	case key.Matches(msg, m.keys.Bookmark):
		m.addBookmark()
		return m, nil

	case key.Matches(msg, m.keys.Start):
		m.cancelNav()
		m.showDocument(startURL, m.startMarkdown())
		return m, nil

	case key.Matches(msg, m.keys.Reload):
		return m.reload()

	case key.Matches(msg, m.keys.Top):
		m.viewport.GotoTop()
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		m.viewport.GotoBottom()
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.showDocument(helpURL, helpMarkdown)
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleEsc peels one layer of state per press, in fixed precedence.
// Esc never quits.
func (m *Model) handleEsc() {
	switch {
	case m.loading:
		m.cancelNav()
		m.status = "load canceled"
	case m.digits != "":
		m.digits = ""
	case m.selected != 0:
		m.selected = 0
		m.applyContent()
	}
}

// startNavigate launches an async resolve for input and bumps the
// navigation sequence so any in-flight result becomes stale.
func (m *Model) startNavigate(input string) tea.Cmd {
	m.cancelNav()
	// Replace semantics belong to the navigation started by reload()
	// only; a leftover flag from a canceled or superseded reload must
	// not corrupt the history of the next navigation.
	m.replaceNext = false
	m.navSeq++
	seq := m.navSeq
	ctx, cancel := context.WithCancel(context.Background())
	m.navCtx = ctx
	m.cancel = cancel
	m.loading = true
	m.loadingHost = displayHost(input)
	resolver := m.resolver
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		final, markdown, err := resolver.Resolve(ctx, input)
		if err != nil {
			return loadFailedMsg{seq: seq, err: err}
		}
		return docLoadedMsg{seq: seq, url: final, markdown: markdown}
	})
}

// cancelNav cancels any in-flight fetch and marks its result stale.
func (m *Model) cancelNav() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.loading {
		m.loading = false
		m.navSeq++
	}
}

// followIndex resolves link n of the current document and navigates.
func (m *Model) followIndex(n int) tea.Cmd {
	link, ok := m.linkByIndex(n)
	if !ok {
		m.status = fmt.Sprintf("no link %d on this page", n)
		return nil
	}
	target, err := m.page.doc.ResolveLink(link)
	if err != nil {
		m.status = "cannot follow: " + err.Error()
		return nil
	}
	return m.startNavigate(target)
}

func (m *Model) linkByIndex(n int) (doc.Link, bool) {
	for _, l := range m.page.doc.Links {
		if l.Index == n {
			return l, true
		}
	}
	return doc.Link{}, false
}

// selectLink advances the link selection: from nothing, Tab picks the
// first marker at or below the viewport top (Shift+Tab the last above
// the bottom); afterwards both cycle. The viewport scrolls minimally to
// keep the selected marker visible.
func (m *Model) selectLink(forwardDir bool) {
	ids := sortedIndices(m.markerLines)
	if len(ids) == 0 {
		m.status = "no links on this page"
		return
	}
	switch {
	case m.selected == 0 && forwardDir:
		m.selected = pickFirstAtOrBelow(ids, m.markerLines, m.viewport.YOffset)
	case m.selected == 0:
		m.selected = pickLastAtOrAbove(ids, m.markerLines, m.viewport.YOffset+m.viewport.Height-1)
	case forwardDir:
		m.selected = cycleNext(ids, m.selected)
	default:
		m.selected = cyclePrev(ids, m.selected)
	}
	if line, ok := m.markerLines[m.selected]; ok {
		m.viewport.SetYOffset(minimalScroll(m.viewport.YOffset, m.viewport.Height, line))
	}
	m.applyContent()
}

func (m *Model) goBack() {
	if len(m.back) == 0 {
		m.status = "no page to go back to"
		return
	}
	m.cancelNav()
	entry := m.back[len(m.back)-1]
	m.back = m.back[:len(m.back)-1]
	m.forward = pushCapped(m.forward, m.currentEntry())
	m.showEntry(entry)
}

func (m *Model) goForward() {
	if len(m.forward) == 0 {
		m.status = "no page to go forward to"
		return
	}
	m.cancelNav()
	entry := m.forward[len(m.forward)-1]
	m.forward = m.forward[:len(m.forward)-1]
	m.back = pushCapped(m.back, m.currentEntry())
	m.showEntry(entry)
}

func (m Model) reload() (tea.Model, tea.Cmd) {
	if isInternal(m.page.url) {
		frac := m.currentFraction()
		if m.page.url == startURL {
			m.setPage(startURL, m.startMarkdown())
		} else {
			m.setPage(m.page.url, m.page.markdown)
		}
		m.viewport.SetYOffset(restoreOffset(frac, m.viewport.TotalLineCount(), m.viewport.Height))
		return m, nil
	}
	if host := hostOf(m.page.url); host != "" {
		m.resolver.Forget(host)
	}
	// Set after startNavigate: it clears the flag on entry.
	cmd := m.startNavigate(m.page.url)
	m.replaceNext = true
	return m, cmd
}

// promptKind selects what the URL-prompt input drives on Enter.
type promptKind int

const (
	promptGoto promptKind = iota // navigate to the typed URL/domain
	promptPipe                   // pipe the page source through a shell command
)

func (m *Model) openPrompt(kind promptKind) tea.Cmd {
	m.promptKind = kind
	if kind == promptPipe {
		m.prompt.Prompt = "|"
		m.prompt.Placeholder = "shell command (page source on stdin)"
	} else {
		m.prompt.Prompt = ":"
		m.prompt.Placeholder = "url or domain"
	}
	m.promptOpen = true
	m.prompt.SetValue("")
	m.syncViewportSize()
	return m.prompt.Focus()
}

func (m *Model) closePrompt() {
	m.promptOpen = false
	m.prompt.Blur()
	m.prompt.SetValue("")
	m.syncViewportSize()
}

func (m *Model) openCurrent() tea.Cmd {
	if isInternal(m.page.url) {
		m.status = "internal page — nothing to open"
		return nil
	}
	return openExternal(m.page.url)
}

// yank copies the selected link's URL — or the page URL — via OSC 52.
func (m *Model) yank() {
	target := m.page.url
	if m.selected != 0 {
		if link, ok := m.linkByIndex(m.selected); ok {
			if resolved, err := m.page.doc.ResolveLink(link); err == nil {
				target = resolved
			} else {
				target = link.Dest
			}
		}
	}
	if isInternal(target) {
		m.status = "internal page — nothing to yank"
		return
	}
	// bubbletea v1 exposes no clipboard or raw-output API, so the OSC 52
	// sequence is emitted as one direct Write to stdout (mirroring
	// termenv.Copy, but with the write error checked). Delivery cannot
	// be confirmed — terminals may have OSC 52 disabled — so the status
	// claims only that the sequence was sent.
	seq := osc52.New(target)
	if strings.HasPrefix(os.Getenv("TERM"), "screen") {
		seq = seq.Screen()
	}
	if _, err := seq.WriteTo(os.Stdout); err != nil {
		m.status = "yank failed: " + err.Error()
		return
	}
	m.status = "sent to clipboard (OSC 52): " + target
}

func (m *Model) addBookmark() {
	if isInternal(m.page.url) {
		m.status = "internal page — not bookmarkable"
		return
	}
	if m.bookmarks == nil {
		m.status = "bookmarks unavailable (no config directory)"
		return
	}
	title := firstH1(m.page.markdown)
	if title == "" {
		title = m.page.url
	}
	if err := m.bookmarks.Add(title, m.page.url); err != nil {
		m.status = "bookmark failed: " + err.Error()
		return
	}
	m.status = "bookmarked: " + title
}

// showDocument makes a document current, pushing the previous page onto
// the back stack and clearing the forward stack.
func (m *Model) showDocument(url, markdown string) {
	if url != m.page.url {
		m.back = pushCapped(m.back, m.currentEntry())
		m.forward = nil
	}
	m.setPage(url, markdown)
	if m.ready {
		m.viewport.SetYOffset(0)
	}
}

// showEntry restores a history entry, including its scroll position.
func (m *Model) showEntry(e histEntry) {
	m.setPage(e.url, e.markdown)
	if m.ready {
		m.viewport.SetYOffset(restoreOffset(e.frac, m.viewport.TotalLineCount(), m.viewport.Height))
	}
}

func (m *Model) currentEntry() histEntry {
	return histEntry{url: m.page.url, markdown: m.page.markdown, frac: m.currentFraction()}
}

func (m *Model) currentFraction() float64 {
	fits := m.viewport.TotalLineCount() <= m.viewport.Height
	return clampFraction(m.viewport.ScrollPercent(), fits)
}

// setPage replaces the current page and re-runs the render pipeline.
func (m *Model) setPage(url, markdown string) {
	m.page = page{url: url, markdown: markdown}
	m.selected = 0
	m.digits = ""
	m.applyRender()
}

// applyRender runs the full pipeline for the current page at the
// current width: renderer (re)build, Truncate → Parse → Annotate →
// glamour render → marker index, then content swap.
func (m *Model) applyRender() {
	if !m.ready {
		return
	}
	m.ensureRenderer()
	m.renderCurrent()
	m.applyContent()
}

// ensureRenderer rebuilds the cached glamour renderer only when the
// clamped wrap width changed.
func (m *Model) ensureRenderer() {
	clamp := clampWidth(m.width, m.cfg.MaxWidth)
	if m.renderer != nil && m.rendererWidth == clamp {
		return
	}
	renderer, err := glamour.NewTermRenderer(m.styleOpt, glamour.WithWordWrap(clamp))
	if err != nil {
		m.status = "renderer error: " + err.Error()
		return
	}
	m.renderer = renderer
	m.rendererWidth = clamp
}

// renderCurrent renders the current page once and caches the result;
// selection highlighting happens later on the cached string.
func (m *Model) renderCurrent() {
	src, truncated := doc.Truncate(m.page.markdown, maxRenderBytes)
	m.page.truncated = truncated
	m.page.doc = doc.Parse(m.page.url, src)
	annotated := doc.Annotate(src, m.page.doc.Links, 0)
	rendered := annotated
	if m.renderer != nil {
		out, err := m.renderer.Render(annotated)
		if err != nil {
			m.status = "render error: " + err.Error()
		} else {
			rendered = out
		}
	}
	m.page.rendered = rendered
	m.markerLocs = markerIndex(rendered, m.page.doc.Links)
	m.markerLines = make(map[int]int, len(m.markerLocs))
	for idx, loc := range m.markerLocs {
		m.markerLines[idx] = loc.line
	}
	annotatable := 0
	for _, l := range m.page.doc.Links {
		if l.Annotatable() {
			annotatable++
		}
	}
	if missing := annotatable - len(m.markerLocs); missing > 0 && m.status == "" {
		m.status = fmt.Sprintf("%d link(s) not reachable via tab — follow by number instead", missing)
	}
}

// applyContent pushes the cached render (with the selection highlight,
// if any) into the viewport, preserving the scroll offset.
func (m *Model) applyContent() {
	if !m.ready {
		return
	}
	content := m.page.rendered
	if m.selected != 0 {
		// Use the resolved marker location rather than re-searching:
		// a bold literal like "# Notes [2]" earlier in the document
		// must not steal the highlight from the real marker.
		if loc, ok := m.markerLocs[m.selected]; ok {
			content = highlightSpan(content, loc.start, loc.end)
		}
	}
	offset := m.viewport.YOffset
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(offset)
}

// chromeHeight is the number of rows of fixed chrome for the input
// mode; the viewport gets the remaining terminal rows.
func chromeHeight(promptOpen bool) int {
	if promptOpen {
		return 2 // prompt row + statusbar
	}
	return 1 // statusbar
}

func (m *Model) syncViewportSize() {
	m.viewport.Width = m.width
	m.viewport.Height = max(0, m.height-chromeHeight(m.promptOpen))
}

func (m Model) View() string {
	if !m.ready {
		return "starting…"
	}
	var b strings.Builder
	b.WriteString(m.viewport.View())
	if m.promptOpen {
		b.WriteString("\n")
		b.WriteString(m.prompt.View())
	}
	b.WriteString("\n")
	b.WriteString(m.statusBar())
	return b.String()
}

func (m Model) statusBar() string {
	pct := int(clampFraction(m.viewport.ScrollPercent(), m.viewport.TotalLineCount() <= m.viewport.Height) * 100)
	left := statusBarStyle.Render(m.statusLeft())
	right := statusBarStyle.Render(fmt.Sprintf("%d%% · %s", pct, m.help.ShortHelpView(m.keys.ShortHelp())))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		left = statusBarStyle.MaxWidth(max(0, m.width-lipgloss.Width(right))).Render(m.statusLeft())
		gap = max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) statusLeft() string {
	switch {
	case m.loading:
		s := m.spin.View() + " loading " + m.loadingHost + "… (esc cancels)"
		if m.digits != "" {
			// Digits typed during a load still target the visible page;
			// keep the buffer on screen so Enter's effect is predictable.
			s += " · follow: " + m.digits + "_"
		}
		return s
	case m.digits != "":
		return "follow: " + m.digits + "_"
	case m.status != "":
		return m.status
	case m.selected != 0:
		if link, ok := m.linkByIndex(m.selected); ok {
			return fmt.Sprintf("[%d] %s", link.Index, link.Dest)
		}
	}
	title := strings.TrimPrefix(m.page.url, "manul:")
	s := "manul · " + title
	if m.page.truncated {
		s += " · truncated"
	}
	return s
}
