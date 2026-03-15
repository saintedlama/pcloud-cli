package filebrowser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistoryPushPop(t *testing.T) {
	h := history{}

	// Pop on empty history returns cursor 0
	entry := h.Pop()
	assert.Equal(t, 0, entry.cursor)
	assert.Equal(t, "", entry.path)
	assert.Equal(t, 0, len(h.Paths()))

	// Push entries
	h.Push("/foo", 2)
	h.Push("/bar", 5)
	assert.Equal(t, 2, len(h.Paths()))

	// Pop returns last pushed
	entry = h.Pop()
	assert.Equal(t, "/bar", entry.path)
	assert.Equal(t, 5, entry.cursor)
	assert.Equal(t, 1, len(h.Paths()))

	entry = h.Pop()
	assert.Equal(t, "/foo", entry.path)
	assert.Equal(t, 2, entry.cursor)
	assert.Equal(t, 0, len(h.Paths()))

	// Pop again returns cursor 0
	entry = h.Pop()
	assert.Equal(t, 0, entry.cursor)
	assert.Equal(t, "", entry.path)
	assert.Equal(t, 0, len(h.Paths()))
}
