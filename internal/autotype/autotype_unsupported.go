//go:build !darwin && !linux && !windows

// Package autotype provides cross-platform automated typing functionality
// for filling credentials into other applications.
package autotype

import "errors"

func init() {
	defaultAutotypeFactory = NewUnsupportedAutotype
}

type unsupportedAutotype struct{}

func (a *unsupportedAutotype) Type(text string) error {
	return errors.New("autotype is not supported on this platform")
}

func NewUnsupportedAutotype() Autotype {
	return &unsupportedAutotype{}
}
