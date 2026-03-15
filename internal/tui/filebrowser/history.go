package filebrowser

// historyEntry records the path and cursor position of a visited directory.
type historyEntry struct {
	path   string
	cursor int
}

// history is a stack of visited directory entries.
type history struct {
	entries []historyEntry
}

// Push saves the current path and cursor position onto the stack.
func (h *history) Push(path string, cursor int) {
	h.entries = append(h.entries, historyEntry{path: path, cursor: cursor})
}

// Pop removes and returns the top entry. When the stack is empty it returns
// an entry with cursor 0, so callers never need a special-case for empty history.
func (h *history) Pop() historyEntry {
	if len(h.entries) == 0 {
		return historyEntry{cursor: 0}
	}
	top := h.entries[len(h.entries)-1]
	h.entries = h.entries[:len(h.entries)-1]
	return top
}

// Paths returns the slice of visited directory paths.
func (h *history) Paths() []string {
	paths := make([]string, len(h.entries))
	for i, e := range h.entries {
		paths[i] = e.path
	}
	return paths
}
