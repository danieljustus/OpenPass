//go:build !test_headless

package clipboard

import "github.com/atotto/clipboard"

func init() {
	defaultClipboardFactory = NewSystemClipboard
}

type systemClipboard struct{}

func (c *systemClipboard) Copy(text string) error {
	return clipboard.WriteAll(text)
}

func (c *systemClipboard) Read() (string, error) {
	return clipboard.ReadAll()
}

// NewSystemClipboard creates a clipboard that uses the real system clipboard.
func NewSystemClipboard() Clipboard {
	return &systemClipboard{}
}
