// Package selector provides a lightweight cursor component for list-style UI
// elements. It holds the list of option items (label + key), tracks the
// selected index, handles up/down key navigation, and renders itself.
// Disabled items are skipped during navigation but still displayed dimmed.
package selector

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Unselected is the sentinel cursor value meaning no item is highlighted.
const Unselected = -1

// ItemStyle controls how a selector item is rendered when it is highlighted.
type ItemStyle int

const (
	StyleDefault ItemStyle = iota
	StyleDanger
)

// Item is a single entry in a Selector.
// Label is the human-readable display string; Key is arbitrary client data.
// Disabled items are shown dimmed and cannot be navigated to.
type Item struct {
	Label    string
	Key      string
	Disabled bool
	Style    ItemStyle
}

// Selection is the value returned by Selected(). It bundles the chosen item's
// Label, Key, Disabled flag, and its position Index in the list.
type Selection struct {
	Label    string
	Key      string
	Index    int
	Disabled bool
	Style    ItemStyle
}

var (
	itemNormalStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2)
	itemSelectStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2).
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("62"))
	itemDangerSelectStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2).
				Bold(true).
				Foreground(lipgloss.Color("231")).
				Background(lipgloss.Color("9"))
	itemDisabledStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2).
				Foreground(lipgloss.Color("240"))
)

// Selector manages cursor position within a list of Items.
type Selector struct {
	items  []Item
	cursor int
}

// New returns a Selector holding items with the given initial cursor.
// Pass Unselected (-1) if no item should be initially highlighted.
func New(items []Item, cursor int) Selector {
	if len(items) == 0 || cursor == Unselected {
		return Selector{items: items, cursor: Unselected}
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}
	return Selector{items: items, cursor: cursor}
}

// Cursor returns the current cursor index, or Unselected (-1).
func (s Selector) Cursor() int { return s.cursor }

// Selected returns the selected item and its index.
// The second return value is false when the cursor is Unselected.
func (s Selector) Selected() (Selection, bool) {
	if s.cursor == Unselected {
		return Selection{}, false
	}
	item := s.items[s.cursor]
	return Selection{Label: item.Label, Key: item.Key, Index: s.cursor, Disabled: item.Disabled, Style: item.Style}, true
}

// Init satisfies the bubbletea component interface.
func (s Selector) Init() tea.Cmd { return nil }

// Update processes any tea.Msg. On up/k or down/j the cursor moves, skipping
// over disabled items. Returns the updated Selector and a nil Cmd.
func (s Selector) Update(msg tea.Msg) (Selector, tea.Cmd) {
	kMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	switch kMsg.String() {
	case "up", "k":
		for prev := s.cursor - 1; prev >= 0; prev-- {
			if !s.items[prev].Disabled {
				s.cursor = prev
				return s, nil
			}
		}
	case "down", "j":
		for next := s.cursor + 1; next < len(s.items); next++ {
			if !s.items[next].Disabled {
				s.cursor = next
				return s, nil
			}
		}
	}
	return s, nil
}

// View renders the list with a highlighted cursor row and dimmed disabled rows.
// It returns a string so it can be embedded in a parent model's View output.
func (s Selector) View() string {
	var sb strings.Builder
	for i, item := range s.items {
		switch {
		case item.Disabled:
			sb.WriteString(itemDisabledStyle.Render(item.Label))
		case i == s.cursor && item.Style == StyleDanger:
			sb.WriteString(itemDangerSelectStyle.Render(item.Label))
		case i == s.cursor:
			sb.WriteString(itemSelectStyle.Render(item.Label))
		default:
			sb.WriteString(itemNormalStyle.Render(item.Label))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
