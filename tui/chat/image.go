package chat

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// termProtocol identifies which terminal image protocol to use.
type termProtocol int

const (
	protoNone   termProtocol = iota // no inline image support — show text placeholder
	protoIterm2                     // iTerm2 inline image protocol (iTerm2, Ghostty, WezTerm, Hyper)
)

// detectTermProtocol checks environment variables to choose the best protocol.
// Inside tmux/screen, image escape sequences are normally stripped unless
// allow-passthrough is explicitly enabled, so we default to protoNone there.
func detectTermProtocol() termProtocol {
	// In multiplexers, inline images are unreliable without extra configuration.
	term := os.Getenv("TERM")
	if strings.HasPrefix(term, "tmux") || strings.HasPrefix(term, "screen") {
		// Allow override: if NETCHAT_TERM_PROGRAM is set to a known image-capable
		// terminal, trust it (user has configured allow-passthrough in tmux.conf).
		if override := strings.ToLower(os.Getenv("NETCHAT_TERM_PROGRAM")); override != "" {
			return detectByProgName(override)
		}
		return protoNone
	}

	prog := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	if p := detectByProgName(prog); p != protoNone {
		return p
	}
	// Some terminals set LC_TERMINAL instead of TERM_PROGRAM (e.g. older iTerm2).
	lc := strings.ToLower(os.Getenv("LC_TERMINAL"))
	return detectByProgName(lc)
}

func detectByProgName(name string) termProtocol {
	switch name {
	case "iterm.app", "iterm2", "ghostty", "wezterm", "hyper":
		return protoIterm2
	}
	return protoNone
}

// RenderImageBytes converts raw image bytes (PNG/JPEG/GIF/WebP) to a terminal
// string using the best available inline image protocol.
// Returns empty string when no supported protocol is detected — the caller
// should show a text placeholder instead (avoids pixelated block art).
func RenderImageBytes(data []byte, maxCols int) string {
	if len(data) == 0 || maxCols < 4 {
		return ""
	}
	switch detectTermProtocol() {
	case protoIterm2:
		return renderIterm2(data, maxCols)
	default:
		return "" // unsupported terminal — caller shows text placeholder
	}
}

// renderIterm2 encodes the image bytes using the iTerm2 inline image protocol.
// Supported by iTerm2, Ghostty, WezTerm, Hyper, and compatible terminals.
func renderIterm2(data []byte, maxCols int) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	// ESC ] 1337 ; File=inline=1;width=<cols>;preserveAspectRatio=1 : <base64> BEL
	return fmt.Sprintf("\033]1337;File=inline=1;width=%d;preserveAspectRatio=1:%s\a\n", maxCols, b64)
}
