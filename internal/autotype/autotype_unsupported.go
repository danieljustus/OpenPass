//go:build !darwin && !linux && !windows

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
