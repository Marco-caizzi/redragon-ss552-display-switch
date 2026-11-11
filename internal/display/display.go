// Package display reads and applies the Windows display topology,
// the same four modes offered by Win+P.
package display

import "fmt"

// Mode is a Win+P display topology.
type Mode int

const (
	// Unknown is reported when the current layout is not one of the four presets.
	Unknown Mode = iota
	Internal
	Clone
	Extend
	External
)

// Selectable lists the modes in knob order.
var Selectable = []Mode{Internal, Clone, Extend, External}

// Label returns the Spanish label shown on the knob screen.
func (m Mode) Label() string {
	switch m {
	case Internal:
		return "Solo pantalla 1"
	case Clone:
		return "Duplicar"
	case Extend:
		return "Extender"
	case External:
		return "Solo pantalla 2"
	default:
		return "Personalizado"
	}
}

func (m Mode) String() string {
	switch m {
	case Internal:
		return "internal"
	case Clone:
		return "clone"
	case Extend:
		return "extend"
	case External:
		return "external"
	default:
		return "unknown"
	}
}

// Index returns the position of m in Selectable, or -1 if it is not selectable.
func (m Mode) Index() int {
	for i, s := range Selectable {
		if s == m {
			return i
		}
	}
	return -1
}

// Step moves n positions through Selectable, wrapping around.
// An Unknown mode starts from Extend, the usual multi-monitor default.
func (m Mode) Step(n int) Mode {
	i := m.Index()
	if i < 0 {
		i = Extend.Index()
	}
	count := len(Selectable)
	i = ((i+n)%count + count) % count
	return Selectable[i]
}

// Backend reads and applies the display topology.
type Backend interface {
	Current() (Mode, error)
	Apply(Mode) error
}

// Error wraps a Win32 return code.
type Error struct {
	Op   string
	Code uintptr
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s failed with code %d", e.Op, e.Code)
}
