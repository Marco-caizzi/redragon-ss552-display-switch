//go:build !windows

package display

import "sync"

// Fake is an in-memory backend used on non-Windows systems for development and tests.
type Fake struct {
	mu       sync.Mutex
	Mode     Mode
	ApplyErr error
	Applied  []Mode
}

// NewBackend returns an in-memory backend starting in Extend mode.
func NewBackend() Backend { return &Fake{Mode: Extend} }

func (f *Fake) Current() (Mode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Mode, nil
}

func (f *Fake) Apply(m Mode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ApplyErr != nil {
		return f.ApplyErr
	}
	f.Applied = append(f.Applied, m)
	f.Mode = m
	return nil
}
