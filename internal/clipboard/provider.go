package clipboard

var (
	globalClipboard         Clipboard
	defaultClipboardFactory func() Clipboard
)

// DefaultClipboard returns the global clipboard or creates a system clipboard.
func DefaultClipboard() Clipboard {
	if globalClipboard != nil {
		return globalClipboard
	}
	if defaultClipboardFactory != nil {
		return defaultClipboardFactory()
	}
	return nil
}

// SetClipboard sets the global clipboard implementation.
func SetClipboard(c Clipboard) {
	globalClipboard = c
}
