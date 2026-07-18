package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// promptInput is a minimal single-line text input for the URL prompt.
// The spec called for bubbles/textinput, but that package needs a
// go.sum entry (atotto/clipboard) the pre-locked module files do not
// carry, and dependency files must not be modified — so this small
// stand-in implements the same contract: while focused it consumes
// every key except Enter/Esc/ctrl+c, which the model handles.
type promptInput struct {
	Prompt      string
	Placeholder string
	Width       int

	value   []rune
	cursor  int
	focused bool
}

func newPromptInput() promptInput {
	return promptInput{Prompt: ":", Width: 40}
}

func (p *promptInput) Focus() tea.Cmd {
	p.focused = true
	return nil
}

func (p *promptInput) Blur() {
	p.focused = false
}

func (p promptInput) Focused() bool {
	return p.focused
}

func (p *promptInput) SetValue(s string) {
	p.value = []rune(s)
	p.cursor = len(p.value)
}

func (p promptInput) Value() string {
	return string(p.value)
}

func (p promptInput) Update(msg tea.Msg) (promptInput, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !p.focused {
		return p, nil
	}
	switch keyMsg.Type {
	case tea.KeyRunes:
		p.insert(keyMsg.Runes)
	case tea.KeySpace:
		p.insert([]rune{' '})
	case tea.KeyBackspace:
		if p.cursor > 0 {
			p.value = append(p.value[:p.cursor-1], p.value[p.cursor:]...)
			p.cursor--
		}
	case tea.KeyDelete:
		if p.cursor < len(p.value) {
			p.value = append(p.value[:p.cursor], p.value[p.cursor+1:]...)
		}
	case tea.KeyLeft:
		if p.cursor > 0 {
			p.cursor--
		}
	case tea.KeyRight:
		if p.cursor < len(p.value) {
			p.cursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		p.cursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		p.cursor = len(p.value)
	case tea.KeyCtrlU:
		p.value = append([]rune{}, p.value[p.cursor:]...)
		p.cursor = 0
	case tea.KeyCtrlW:
		p.deleteWordBackward()
	}
	return p, nil
}

func (p *promptInput) insert(runes []rune) {
	p.value = append(p.value[:p.cursor], append(append([]rune{}, runes...), p.value[p.cursor:]...)...)
	p.cursor += len(runes)
}

func (p *promptInput) deleteWordBackward() {
	i := p.cursor
	for i > 0 && p.value[i-1] == ' ' {
		i--
	}
	for i > 0 && p.value[i-1] != ' ' {
		i--
	}
	p.value = append(p.value[:i], p.value[p.cursor:]...)
	p.cursor = i
}

// View renders the prompt with a block cursor, horizontally scrolled so
// the cursor stays visible within Width cells.
func (p promptInput) View() string {
	if len(p.value) == 0 && p.Placeholder != "" {
		return p.Prompt + reverseOn + " " + reverseOff + p.Placeholder
	}
	window := p.Width - len(p.Prompt) - 1
	if window < 1 {
		window = 1
	}
	start := 0
	if p.cursor >= window {
		start = p.cursor - window + 1
	}
	end := min(len(p.value), start+window)
	visible := p.value[start:end]
	cur := p.cursor - start

	before := string(visible[:min(cur, len(visible))])
	at := " "
	after := ""
	if cur < len(visible) {
		at = string(visible[cur])
		after = string(visible[cur+1:])
	}
	if !p.focused {
		return p.Prompt + string(visible)
	}
	return p.Prompt + before + reverseOn + at + reverseOff + after
}
