package knob

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
)

type fakeBackend struct {
	mode     display.Mode
	applyErr error
	applied  []display.Mode
}

func (f *fakeBackend) Current() (display.Mode, error) { return f.mode, nil }

func (f *fakeBackend) Apply(m display.Mode) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	f.applied = append(f.applied, m)
	f.mode = m
	return nil
}

type harness struct {
	t       *testing.T
	now     time.Time
	backend *fakeBackend
	drawn   map[string]string
	m       *Manager
}

func newHarness(t *testing.T, mode display.Mode) *harness {
	h := &harness{
		t:       t,
		now:     time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC),
		backend: &fakeBackend{mode: mode},
		drawn:   map[string]string{},
	}
	h.m = NewManager(DefaultConfig, h.backend,
		func(ctx, svg string) { h.drawn[ctx] = svg },
		nil,
		func() time.Time { return h.now })
	return h
}

func (h *harness) advance(d time.Duration) {
	h.now = h.now.Add(d)
	h.m.Tick()
}

// expect checks the label and status line currently drawn on a knob.
func (h *harness) expect(ctx string, wantLabel, wantStatus string) {
	h.t.Helper()
	svg := h.drawn[ctx]
	if !strings.Contains(svg, ">"+wantLabel+"<") {
		h.t.Fatalf("knob %s: label %q not drawn in %s", ctx, wantLabel, svg)
	}
	if !strings.Contains(svg, wantStatus) {
		h.t.Fatalf("knob %s: status %q not drawn in %s", ctx, wantStatus, svg)
	}
}

func TestAppearShowsActiveMode(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.expect("k1", "Extender", "Activo")
}

func TestRotateThenPressApplies(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")

	h.m.Rotate("k1", 1) // Extend -> External
	h.expect("k1", "Solo pantalla 2", "Pulsá para aplicar · 5s")
	if len(h.backend.applied) != 0 {
		t.Fatal("rotating must not apply anything")
	}

	h.m.Press("k1")
	if got := h.backend.applied; len(got) != 1 || got[0] != display.External {
		t.Fatalf("applied = %v, want [external]", got)
	}
	h.expect("k1", "Solo pantalla 2", "Aplicado")

	h.advance(2 * time.Second) // Feedback expires.
	h.expect("k1", "Solo pantalla 2", "Activo")
}

func TestNoConfirmationRevertsToActive(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.m.Rotate("k1", -2) // Extend -> Clone -> Internal

	h.advance(2 * time.Second)
	h.expect("k1", "Solo pantalla 1", "Pulsá para aplicar · 3s")

	h.advance(3 * time.Second)
	h.expect("k1", "Extender", "Activo")
	if len(h.backend.applied) != 0 {
		t.Fatal("expired selection must not be applied")
	}

	// A late press after expiry does not apply the old selection.
	h.m.Press("k1")
	if len(h.backend.applied) != 0 {
		t.Fatal("press after expiry applied a mode")
	}
}

func TestRotationWrapsAround(t *testing.T) {
	h := newHarness(t, display.External)
	h.m.Appear("k1")
	h.m.Rotate("k1", 1) // External wraps to Internal.
	h.expect("k1", "Solo pantalla 1", "Pulsá para aplicar")
}

func TestRotatingBackToActiveClearsPending(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.m.Rotate("k1", 1)
	h.m.Rotate("k1", -1)
	h.expect("k1", "Extender", "Activo")
	h.m.Press("k1")
	if len(h.backend.applied) != 0 {
		t.Fatal("press on the active mode must not re-apply it")
	}
}

func TestEachRotationRestartsTheTimer(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.m.Rotate("k1", 1)
	h.advance(4 * time.Second)
	h.m.Rotate("k1", 1) // External -> Internal, timer restarts.
	h.advance(4 * time.Second)
	h.expect("k1", "Solo pantalla 1", "Pulsá para aplicar · 1s")
	h.m.Press("k1")
	if got := h.backend.applied; len(got) != 1 || got[0] != display.Internal {
		t.Fatalf("applied = %v, want [internal]", got)
	}
}

func TestApplyFailureShowsErrorThenReverts(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.backend.applyErr = errors.New("only one monitor")
	h.m.Appear("k1")
	h.m.Rotate("k1", -1) // Clone
	h.m.Press("k1")
	h.expect("k1", "Duplicar", "No se pudo aplicar")
	h.advance(2 * time.Second)
	h.expect("k1", "Extender", "Activo")
}

func TestExternalChangeIsPickedUpByPolling(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.backend.mode = display.Clone // Changed with Win+P.
	h.advance(1 * time.Second)
	h.expect("k1", "Extender", "Activo") // Not polled yet.
	h.advance(2 * time.Second)
	h.expect("k1", "Duplicar", "Activo")
}

func TestUnknownModeStartsFromExtend(t *testing.T) {
	h := newHarness(t, display.Unknown)
	h.m.Appear("k1")
	h.expect("k1", "Personalizado", "Girá para elegir")
	h.m.Rotate("k1", 1)
	h.expect("k1", "Solo pantalla 2", "Pulsá para aplicar")
}

func TestKnobsAreIndependentButShareActiveMode(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.m.Appear("k2")
	h.m.Rotate("k1", 1)
	h.expect("k2", "Extender", "Activo")
	h.m.Press("k1")
	h.expect("k2", "Solo pantalla 2", "Activo")
}

func TestDisappearedKnobIgnoresEvents(t *testing.T) {
	h := newHarness(t, display.Extend)
	h.m.Appear("k1")
	h.m.Disappear("k1")
	h.m.Rotate("k1", 1)
	h.m.Press("k1")
	if len(h.backend.applied) != 0 {
		t.Fatal("events on a removed knob must be ignored")
	}
}
