// Package clipboard provides clipboard operations with build-tagged implementations.
package clipboard

// Clipboard defines the interface for clipboard operations.
type Clipboard interface {
	Copy(text string) error
	Read() (string, error)
}
