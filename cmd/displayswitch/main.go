// displayswitch is a StreamDock plugin that switches the Windows display mode (Win+P) from a knob.
//
// StreamDock launches it as:
//
//	displayswitch.exe -port <n> -pluginUUID <id> -registerEvent <event> -info <json>
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/display"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/knob"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/render"
	"github.com/Marco-caizzi/redragon-ss552-display-switch/internal/streamdock"
)

// ActionUUID must match manifest.json.
const ActionUUID = "com.marco.displayswitch.knob"

const (
	tickInterval = 250 * time.Millisecond
	maxLogSize   = 1 << 20
)

func main() {
	logger := newLogger()
	args := parseArgs(os.Args[1:])
	port, _ := strconv.Atoi(args["port"])
	pluginUUID, registerEvent := args["pluginUUID"], args["registerEvent"]

	if port == 0 || pluginUUID == "" || registerEvent == "" {
		logger.Printf("missing launch arguments: %v", os.Args[1:])
		fmt.Fprintln(os.Stderr, "this program is started by StreamDock")
		os.Exit(2)
	}

	if err := run(port, pluginUUID, registerEvent, logger); err != nil {
		logger.Printf("plugin stopped: %v", err)
		os.Exit(1)
	}
}

// parseArgs reads "-name value", "--name value" and "-name=value" pairs.
// Unknown names are kept and ignored, so new arguments from StreamDock never stop the plugin.
func parseArgs(argv []string) map[string]string {
	args := map[string]string{}
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			args[name[:eq]] = name[eq+1:]
			continue
		}
		if i+1 < len(argv) && !isFlagName(argv[i+1]) {
			args[name] = argv[i+1]
			i++
		} else {
			args[name] = ""
		}
	}
	return args
}

// isFlagName reports whether s looks like a flag rather than a value (negative numbers are values).
func isFlagName(s string) bool {
	if !strings.HasPrefix(s, "-") || len(s) < 2 {
		return false
	}
	c := s[1]
	return c == '-' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func run(port int, pluginUUID, registerEvent string, logger *log.Logger) error {
	client, err := streamdock.Connect(port, pluginUUID, registerEvent)
	if err != nil {
		return err
	}
	defer client.Close()
	logger.Printf("connected to StreamDock on port %d", port)

	draw := func(context, svg string) {
		if err := client.SetImage(context, render.DataURI(svg)); err != nil {
			logger.Printf("setImage failed: %v", err)
		}
	}
	manager := knob.NewManager(knob.DefaultConfig, display.NewBackend(), draw, logger.Printf, nil)
	logger.Printf("active display mode: %s", manager.Active())

	events := make(chan streamdock.Event)
	readErr := make(chan error, 1)
	go func() {
		for {
			ev, err := client.Next()
			if err != nil {
				readErr <- err
				return
			}
			events <- ev
		}
	}()

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case ev := <-events:
			handleEvent(manager, ev, logger)
		case <-ticker.C:
			manager.Tick()
		case err := <-readErr:
			// StreamDock closed the socket (app exit or plugin reload).
			return err
		}
	}
}

// handleEvent routes StreamDock events for this action to the knob manager.
func handleEvent(m *knob.Manager, ev streamdock.Event, logger *log.Logger) {
	if ev.Action != "" && ev.Action != ActionUUID {
		return
	}
	switch ev.Event {
	case "willAppear":
		m.Appear(ev.Context)
	case "willDisappear":
		m.Disappear(ev.Context)
	case "dialRotate":
		var p streamdock.DialPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			logger.Printf("invalid dialRotate payload: %v", err)
			return
		}
		m.Rotate(ev.Context, p.Ticks)
	case "dialDown":
		m.Press(ev.Context)
	}
}

// newLogger writes to displayswitch.log next to the executable, starting fresh when it grows too large.
func newLogger() *log.Logger {
	var out io.Writer = io.Discard
	if exe, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(exe), "displayswitch.log")
		flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
		if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
			flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		}
		if f, err := os.OpenFile(path, flags, 0o644); err == nil {
			out = f
		}
	}
	return log.New(out, "", log.LstdFlags)
}
