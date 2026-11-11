package main

import (
	"encoding/json"
	"io"
	"log"
	"testing"
	"time"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/knob"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/streamdock"
)

type recordingBackend struct {
	mode    display.Mode
	applied []display.Mode
}

func (b *recordingBackend) Current() (display.Mode, error) { return b.mode, nil }
func (b *recordingBackend) Apply(m display.Mode) error {
	b.applied = append(b.applied, m)
	b.mode = m
	return nil
}

func event(name, action, context string, payload any) streamdock.Event {
	raw, _ := json.Marshal(payload)
	return streamdock.Event{Event: name, Action: action, Context: context, Payload: raw}
}

// Uses the exact event shapes documented by the StreamDock SDK.
func TestHandleEventRoutesDialEvents(t *testing.T) {
	backend := &recordingBackend{mode: display.Extend}
	draws := 0
	now := time.Now()
	m := knob.NewManager(knob.DefaultConfig, backend, func(string, string) { draws++ }, nil, func() time.Time { return now })
	logger := log.New(io.Discard, "", 0)

	handleEvent(m, event("willAppear", ActionUUID, "k1", map[string]any{"controller": "Knob", "settings": map[string]any{}}), logger)
	handleEvent(m, event("dialRotate", ActionUUID, "k1", map[string]any{"controller": "Encoder", "ticks": -1, "pressed": false}), logger)
	handleEvent(m, event("dialDown", ActionUUID, "k1", map[string]any{"controller": "Encoder"}), logger)

	if len(backend.applied) != 1 || backend.applied[0] != display.Clone {
		t.Fatalf("applied = %v, want [clone]", backend.applied)
	}
	if draws == 0 {
		t.Fatal("knob was never drawn")
	}
}

func TestParseArgs(t *testing.T) {
	info := `{"application":{"language":"es","platform":"windows"},"devices":[{"id":"X","type":7}]}`
	got := parseArgs([]string{
		"-port", "28196",
		"--pluginUUID", "com.marco.displayswitch",
		"-registerEvent=registerPlugin",
		"-info", info,
		"-unknownFlag",
		"-offset", "-5",
	})
	want := map[string]string{
		"port":          "28196",
		"pluginUUID":    "com.marco.displayswitch",
		"registerEvent": "registerPlugin",
		"info":          info,
		"unknownFlag":   "",
		"offset":        "-5",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestHandleEventIgnoresOtherActions(t *testing.T) {
	backend := &recordingBackend{mode: display.Extend}
	m := knob.NewManager(knob.DefaultConfig, backend, func(string, string) {}, nil, nil)
	logger := log.New(io.Discard, "", 0)

	handleEvent(m, event("willAppear", "com.other.plugin", "k1", map[string]any{}), logger)
	handleEvent(m, event("dialRotate", "com.other.plugin", "k1", map[string]any{"ticks": 1}), logger)
	handleEvent(m, event("dialDown", "com.other.plugin", "k1", map[string]any{}), logger)

	if len(backend.applied) != 0 {
		t.Fatalf("events for another action applied %v", backend.applied)
	}
}
