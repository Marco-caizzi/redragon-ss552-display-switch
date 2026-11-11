//go:build windows

package display

import (
	"syscall"
	"unsafe"
)

// Win32 constants from wingdi.h / winuser.h.
const (
	qdcDatabaseCurrent = 0x00000004

	sdcTopologyInternal = 0x00000001
	sdcTopologyClone    = 0x00000002
	sdcTopologyExtend   = 0x00000004
	sdcTopologyExternal = 0x00000008
	sdcApply            = 0x00000080

	topologyInternal = 0x00000001
	topologyClone    = 0x00000002
	topologyExtend   = 0x00000004
	topologyExternal = 0x00000008

	errorSuccess            = 0
	errorInsufficientBuffer = 122

	// sizeof(DISPLAYCONFIG_PATH_INFO) and sizeof(DISPLAYCONFIG_MODE_INFO).
	pathInfoSize = 72
	modeInfoSize = 64
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	procGetDisplayConfigBufferSize = user32.NewProc("GetDisplayConfigBufferSizes")
	procQueryDisplayConfig         = user32.NewProc("QueryDisplayConfig")
	procSetDisplayConfig           = user32.NewProc("SetDisplayConfig")
)

// Windows talks to the real display configuration through user32.dll.
type Windows struct{}

// NewBackend returns the Windows display backend.
func NewBackend() Backend { return Windows{} }

// Current returns the topology Windows considers active (the Win+P selection).
func (Windows) Current() (Mode, error) {
	for attempt := 0; attempt < 3; attempt++ {
		var numPaths, numModes uint32
		r, _, _ := procGetDisplayConfigBufferSize.Call(
			qdcDatabaseCurrent,
			uintptr(unsafe.Pointer(&numPaths)),
			uintptr(unsafe.Pointer(&numModes)),
		)
		if r != errorSuccess {
			return Unknown, &Error{Op: "GetDisplayConfigBufferSizes", Code: r}
		}

		// Allocate at least one element so the pointers are never nil.
		paths := make([]byte, int(max(numPaths, 1))*pathInfoSize)
		modes := make([]byte, int(max(numModes, 1))*modeInfoSize)
		var topology uint32

		r, _, _ = procQueryDisplayConfig.Call(
			qdcDatabaseCurrent,
			uintptr(unsafe.Pointer(&numPaths)),
			uintptr(unsafe.Pointer(&paths[0])),
			uintptr(unsafe.Pointer(&numModes)),
			uintptr(unsafe.Pointer(&modes[0])),
			uintptr(unsafe.Pointer(&topology)),
		)
		if r == errorInsufficientBuffer {
			continue // Configuration changed between the two calls.
		}
		if r != errorSuccess {
			return Unknown, &Error{Op: "QueryDisplayConfig", Code: r}
		}
		return modeFromTopology(topology), nil
	}
	return Unknown, &Error{Op: "QueryDisplayConfig", Code: errorInsufficientBuffer}
}

// Apply switches to a Win+P preset, exactly like DisplaySwitch.exe.
func (Windows) Apply(m Mode) error {
	var flag uintptr
	switch m {
	case Internal:
		flag = sdcTopologyInternal
	case Clone:
		flag = sdcTopologyClone
	case Extend:
		flag = sdcTopologyExtend
	case External:
		flag = sdcTopologyExternal
	default:
		return &Error{Op: "SetDisplayConfig(unsupported mode)", Code: 87}
	}
	r, _, _ := procSetDisplayConfig.Call(0, 0, 0, 0, sdcApply|flag)
	if r != errorSuccess {
		return &Error{Op: "SetDisplayConfig", Code: r}
	}
	return nil
}

func modeFromTopology(t uint32) Mode {
	switch t {
	case topologyInternal:
		return Internal
	case topologyClone:
		return Clone
	case topologyExtend:
		return Extend
	case topologyExternal:
		return External
	default:
		return Unknown
	}
}
