//go:build test_headless

package clipboard

import "sync"

type nullClipboard struct {
	mu   sync.Mutex
	text string
}

func (c *nullClipboard) Copy(text string) error {
	c.mu.Lock()
	c.text = text
	c.mu.Unlock()
	return nil
}

func (c *nullClipboard) Read() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.text, nil
}

func init() {
	defaultClipboardFactory = NewNullClipboard
}

// NewNullClipboard creates a no-op clipboard for headless testing.
func NewNullClipboard() Clipboard {
	return &nullClipboard{}
}
