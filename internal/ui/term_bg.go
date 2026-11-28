package ui

import (
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var (
	termBgMu         sync.Mutex
	termBgWriter     io.Writer
	termBgLastHex    string
	termBgOverridden bool
)

// EnableTerminalBackgroundSync enables OSC 11 / OSC 111 synchronization so the
// terminal's outer padding/backdrop matches curated built-in colorschemes (such
// as light themes) and restores cleanly on exit. Only activates on real TTYs.
func EnableTerminalBackgroundSync(f *os.File) {
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return
	}
	termBgMu.Lock()
	termBgWriter = f
	termBgLastHex = ""
	termBgOverridden = false
	termBgMu.Unlock()
	SyncTerminalBackground()
}

func setTerminalBgWriterForTest(w io.Writer) {
	termBgMu.Lock()
	termBgWriter = w
	termBgLastHex = ""
	termBgOverridden = false
	termBgMu.Unlock()
}

// SyncTerminalBackground emits OSC 11 to match activePalette.CanvasBg when a
// solid curated colorscheme is active, or OSC 111 to restore the terminal's
// native background (e.g. when using the translucent "omarchy" theme).
func SyncTerminalBackground() {
	termBgMu.Lock()
	defer termBgMu.Unlock()
	if termBgWriter == nil {
		return
	}
	if paintCanvasBg {
		hex := strings.TrimSpace(string(canvasBgColor()))
		if isHexColor(hex) {
			if termBgOverridden && termBgLastHex == hex {
				return
			}
			_, _ = io.WriteString(termBgWriter, "\x1b]11;"+hex+"\x07")
			termBgLastHex = hex
			termBgOverridden = true
			return
		}
	}
	if termBgOverridden {
		_, _ = io.WriteString(termBgWriter, "\x1b]111\x07")
		termBgLastHex = ""
		termBgOverridden = false
	}
}

// ResetTerminalBackground emits OSC 111 to restore the terminal's original
// background color if Larry previously changed it via OSC 11.
func ResetTerminalBackground() {
	termBgMu.Lock()
	defer termBgMu.Unlock()
	if termBgWriter == nil || !termBgOverridden {
		return
	}
	_, _ = io.WriteString(termBgWriter, "\x1b]111\x07")
	termBgLastHex = ""
	termBgOverridden = false
}
