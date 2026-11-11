package render

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
)

func allViews() map[string]View {
	return map[string]View{
		"1_active_extend":    {Mode: display.Extend, Status: Active},
		"2_pending_external": {Mode: display.External, Status: Pending, Remaining: 0.6, Seconds: 3},
		"3_pending_internal": {Mode: display.Internal, Status: Pending, Remaining: 1, Seconds: 5},
		"4_pending_clone":    {Mode: display.Clone, Status: Pending, Remaining: 0.2, Seconds: 1},
		"5_applied":          {Mode: display.External, Status: Applied},
		"6_failed":           {Mode: display.Clone, Status: Failed},
		"7_unknown":          {Mode: display.Unknown, Status: Active},
	}
}

func TestSVGIsWellFormed(t *testing.T) {
	for name, v := range allViews() {
		svg := SVG(v)
		dec := xml.NewDecoder(strings.NewReader(svg))
		for {
			_, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s: invalid SVG: %v", name, err)
			}
		}
	}
}

func TestDataURI(t *testing.T) {
	if !strings.HasPrefix(DataURI("<svg/>"), "data:image/svg+xml;base64,") {
		t.Fatal("unexpected data URI prefix")
	}
}

// Set RENDER_PREVIEW_DIR to write every state as an SVG file for visual review.
func TestWritePreviews(t *testing.T) {
	dir := os.Getenv("RENDER_PREVIEW_DIR")
	if dir == "" {
		t.Skip("RENDER_PREVIEW_DIR not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, v := range allViews() {
		if err := os.WriteFile(filepath.Join(dir, name+".svg"), []byte(SVG(v)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
