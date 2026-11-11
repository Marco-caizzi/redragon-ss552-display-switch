// Package knob implements the select-then-confirm behaviour of the display switch knob.
//
// Rotating moves through the Win+P modes. Pressing applies the selected mode.
// Without a press, the selection expires and the screen returns to the active mode.
package knob

import (
	"math"
	"time"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/render"
)

// Config tunes the knob behaviour.
type Config struct {
	// ConfirmWindow is how long a selection waits for a press before reverting.
	ConfirmWindow time.Duration
	// PollInterval is how often the active mode is re-read from Windows,
	// so changes made with Win+P show up on the knob.
	PollInterval time.Duration
	// FeedbackDuration is how long "Aplicado" / "No se pudo aplicar" stays on screen.
	FeedbackDuration time.Duration
}

// DefaultConfig is used when no settings are provided.
var DefaultConfig = Config{
	ConfirmWindow:    5 * time.Second,
	PollInterval:     3 * time.Second,
	FeedbackDuration: 2 * time.Second,
}

// DrawFunc receives a knob context and the SVG to show on it.
type DrawFunc func(context, svg string)

// LogFunc receives diagnostic messages.
type LogFunc func(format string, args ...any)

type knobState struct {
	candidate     display.Mode
	pendingUntil  time.Time // Zero when nothing is waiting for confirmation.
	feedback      render.Status
	feedbackUntil time.Time // Zero when no feedback is shown.
	lastSVG       string
}

// Manager tracks every knob showing the display switch action.
// It is not safe for concurrent use; call it from a single goroutine.
type Manager struct {
	cfg      Config
	backend  display.Backend
	draw     DrawFunc
	logf     LogFunc
	now      func() time.Time
	knobs    map[string]*knobState
	active   display.Mode
	lastPoll time.Time
}

// NewManager creates a manager. now may be nil to use time.Now.
func NewManager(cfg Config, backend display.Backend, draw DrawFunc, logf LogFunc, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	m := &Manager{
		cfg:     cfg,
		backend: backend,
		draw:    draw,
		logf:    logf,
		now:     now,
		knobs:   map[string]*knobState{},
		active:  display.Unknown,
	}
	m.refreshActive()
	return m
}

// Active returns the last known Windows display mode.
func (m *Manager) Active() display.Mode { return m.active }

// Appear registers a knob and draws the active mode on it.
func (m *Manager) Appear(context string) {
	m.knobs[context] = &knobState{candidate: m.active}
	m.redraw(context, true)
}

// Disappear forgets a knob.
func (m *Manager) Disappear(context string) {
	delete(m.knobs, context)
}

// Rotate moves the selection by ticks (positive is clockwise).
func (m *Manager) Rotate(context string, ticks int) {
	k := m.knobs[context]
	if k == nil || ticks == 0 {
		return
	}
	k.feedbackUntil = time.Time{}
	k.candidate = k.candidate.Step(ticks)
	if k.candidate == m.active {
		// Back on the active mode: nothing to confirm.
		k.pendingUntil = time.Time{}
	} else {
		k.pendingUntil = m.now().Add(m.cfg.ConfirmWindow)
	}
	m.redraw(context, false)
}

// Press applies the pending selection. Without a pending selection it re-reads the active mode.
func (m *Manager) Press(context string) {
	k := m.knobs[context]
	if k == nil {
		return
	}
	if k.pendingUntil.IsZero() || k.candidate == m.active {
		m.refreshActive()
		k.candidate = m.active
		k.pendingUntil = time.Time{}
		m.redraw(context, true)
		return
	}

	target := k.candidate
	k.pendingUntil = time.Time{}
	m.logf("applying display mode %s (was %s)", target, m.active)
	err := m.backend.Apply(target)
	m.refreshActive()

	if err != nil {
		m.logf("apply %s failed: %v", target, err)
		k.feedback = render.Failed
	} else {
		k.feedback = render.Applied
	}
	k.feedbackUntil = m.now().Add(m.cfg.FeedbackDuration)
	if err != nil {
		k.candidate = target // Keep showing what failed while the error is visible.
	} else {
		k.candidate = m.active
	}
	m.redrawAll(true)
}

// Tick handles expirations, countdown updates and polling. Call it a few times per second.
func (m *Manager) Tick() {
	now := m.now()
	if now.Sub(m.lastPoll) >= m.cfg.PollInterval {
		previous := m.active
		m.refreshActive()
		if m.active != previous {
			m.logf("display mode changed outside the knob: %s -> %s", previous, m.active)
			for _, k := range m.knobs {
				if k.pendingUntil.IsZero() && k.feedbackUntil.IsZero() {
					k.candidate = m.active
				}
			}
		}
	}

	for context, k := range m.knobs {
		if !k.feedbackUntil.IsZero() && !now.Before(k.feedbackUntil) {
			k.feedbackUntil = time.Time{}
			k.candidate = m.active
		}
		if !k.pendingUntil.IsZero() && !now.Before(k.pendingUntil) {
			// No confirmation: go back to what Windows is actually using.
			k.pendingUntil = time.Time{}
			k.candidate = m.active
		}
		m.redraw(context, false)
	}
}

func (m *Manager) refreshActive() {
	m.lastPoll = m.now()
	mode, err := m.backend.Current()
	if err != nil {
		m.logf("reading display mode failed: %v", err)
		return
	}
	m.active = mode
}

func (m *Manager) redrawAll(force bool) {
	for context := range m.knobs {
		m.redraw(context, force)
	}
}

// redraw sends the view only when it changed, unless force is set.
func (m *Manager) redraw(context string, force bool) {
	k := m.knobs[context]
	if k == nil {
		return
	}
	svg := render.SVG(m.view(k))
	if !force && svg == k.lastSVG {
		return
	}
	k.lastSVG = svg
	m.draw(context, svg)
}

func (m *Manager) view(k *knobState) render.View {
	now := m.now()
	if !k.feedbackUntil.IsZero() && now.Before(k.feedbackUntil) {
		return render.View{Mode: k.candidate, Status: k.feedback}
	}
	if !k.pendingUntil.IsZero() && now.Before(k.pendingUntil) {
		left := k.pendingUntil.Sub(now)
		return render.View{
			Mode:      k.candidate,
			Status:    render.Pending,
			Remaining: float64(left) / float64(m.cfg.ConfirmWindow),
			Seconds:   int(math.Ceil(left.Seconds())),
		}
	}
	return render.View{Mode: m.active, Status: render.Active}
}
