//go:build test_headless

package autotype

import "sync"

type nullAutotype struct {
	mu   sync.Mutex
	text string
}

func (a *nullAutotype) Type(text string) error {
	a.mu.Lock()
	a.text = text
	a.mu.Unlock()
	return nil
}

func (a *nullAutotype) LastTyped() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.text
}

func init() {
	defaultAutotypeFactory = NewNullAutotype
}

func NewNullAutotype() Autotype {
	return &nullAutotype{}
}
