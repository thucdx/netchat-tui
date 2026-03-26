package chat

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/BourgeoisBear/rasterm"
	_ "golang.org/x/image/webp" // register WebP decoder (common for Mattermost custom emoji)
)

// InlineImageCols is the width (terminal cells) for inline image renders.
// 56 cells gives good resolution on wide terminals while still fitting in an
// 80-column layout (chat pane is typically 52+ cols after the sidebar).
const InlineImageCols = 56

// InlineImageRows is the height (terminal rows) for inline image renders.
const InlineImageRows = 20

// InlineEmojiCols is the width (terminal cells) for inline custom emoji renders.
// 4 cells keeps it roughly the same visual footprint as two Unicode emoji side-by-side.
const InlineEmojiCols = 4

// InlineEmojiRows is the height (terminal rows) for inline custom emoji renders.
// 2 rows = 4 pixel rows; enough detail to recognise a simple emoji shape.
const InlineEmojiRows = 2

// ─── terminal protocol detection (lazy, cached) ────────────────────────────

var (
	termKitty  bool
	termIterm  bool
	termInTmux bool
	termOnce   sync.Once
)

// initTermProto detects terminal image-protocol capability once and caches the
// result.  rasterm checks TERM_PROGRAM directly, but when the user runs tmux
// inside Ghostty, TERM_PROGRAM is overridden to "tmux" and rasterm misses it.
//
// Ghostty sets GHOSTTY_RESOURCES_DIR in the environment of every shell it
// spawns; this var survives into tmux panes.  We use it as a secondary signal.
//
// Note: for Kitty-protocol images to actually render through tmux the user must
// add the following line to ~/.tmux.conf (requires tmux ≥ 3.3):
//
//	set -g allow-passthrough on
func initTermProto() {
	termOnce.Do(func() {
		termProgram := os.Getenv("TERM_PROGRAM")
		term := os.Getenv("TERM")
		ghosttyDir := os.Getenv("GHOSTTY_RESOURCES_DIR")
		kittyWin := os.Getenv("KITTY_WINDOW_ID")
		lcTerminal := os.Getenv("LC_TERMINAL")
		tmuxEnv := os.Getenv("TMUX")

		log.Printf("image/proto: env TERM=%q TERM_PROGRAM=%q GHOSTTY_RESOURCES_DIR=%q KITTY_WINDOW_ID=%q LC_TERMINAL=%q TMUX=%q",
			term, termProgram, ghosttyDir, kittyWin, lcTerminal, tmuxEnv)

		termKitty = rasterm.IsKittyCapable()
		termIterm = rasterm.IsItermCapable()

		log.Printf("image/proto: rasterm detection — kitty=%v iterm=%v", termKitty, termIterm)

		// If rasterm didn't recognise the terminal (e.g. TERM_PROGRAM=tmux
		// overrides the real value), fall back to Ghostty's own env marker.
		if !termKitty && ghosttyDir != "" {
			log.Printf("image/proto: GHOSTTY_RESOURCES_DIR set — enabling Kitty protocol (ensure 'set -g allow-passthrough on' in tmux.conf)")
			termKitty = true
		}

		termInTmux = tmuxEnv != ""

		log.Printf("image/proto: final — kitty=%v iterm=%v inTmux=%v", termKitty, termIterm, termInTmux)
	})
}

// ─── Kitty Unicode Placeholder support ──────────────────────────────────────

// kittyNextID is an atomic counter for assigning unique Kitty image IDs.
// IDs must be unique across all images currently loaded in the terminal GPU.
var kittyNextID atomic.Uint32

func init() { kittyNextID.Store(1) }

// IsKittyCapable returns true if the terminal supports the Kitty Graphics Protocol.
func IsKittyCapable() bool {
	initTermProto()
	return termKitty
}

// kittyPlaceholderSize computes the (cols, rows) cell grid for a Kitty virtual
// placement that fits within (maxCols, maxRows) while matching the image's
// pixel aspect ratio as closely as possible.
//
// Standard terminal cells are approximately twice as tall as they are wide
// (e.g. 8 px wide × 16 px tall), so a cols×rows grid of cells represents a
// roughly (cols : rows*2) pixel rectangle.
func kittyPlaceholderSize(imgW, imgH, maxCols, maxRows int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 {
		return maxCols, maxRows
	}
	const cellHeightToWidth = 2 // pixels per cell height ÷ pixels per cell width

	// Start with maximum width and derive the matching height.
	cols = maxCols
	rows = (imgH*cols + imgW*cellHeightToWidth/2) / (imgW * cellHeightToWidth)
	if rows < 1 {
		rows = 1
	}

	if rows > maxRows {
		// Image is too tall; fit to maxRows and shrink cols proportionally.
		rows = maxRows
		cols = (imgW*rows*cellHeightToWidth + imgH/2) / imgH
		if cols < 1 {
			cols = 1
		}
		if cols > maxCols {
			cols = maxCols
		}
	}
	return cols, rows
}

// UploadKittyImage uploads a PNG-encoded image to the terminal using the Kitty
// Graphics Protocol with U=1 (Unicode placeholder mode).  The image is stored
// in the terminal's GPU but NOT displayed until U+10EEEE placeholder characters
// appear in the text stream.
//
// It writes directly to /dev/tty (bypassing Bubbletea's output) because Kitty
// APC sequences must not pass through Lipgloss/reflow which would corrupt them.
//
// The virtual placement dimensions are computed from the image's actual pixel
// aspect ratio so the terminal does not letterbox the image with black bars.
//
// Returns the assigned image ID, placement ID, and the actual (cols, rows)
// used for the virtual placement.  imgID == 0 signals an error.
func UploadKittyImage(imgData []byte, maxCols, maxRows int) (imgID, placementID uint32, cols, rows int) {
	initTermProto()

	// Decode and re-encode as PNG for the terminal.
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		log.Printf("image/kitty-upload: decode error — %v", err)
		return 0, 0, 0, 0
	}

	b := img.Bounds()
	cols, rows = kittyPlaceholderSize(b.Dx(), b.Dy(), maxCols, maxRows)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		log.Printf("image/kitty-upload: png encode error — %v", err)
		return 0, 0, 0, 0
	}

	imgID = kittyNextID.Add(1) - 1
	placementID = imgID // use same value for simplicity

	// Upload image with U=1 (store only, display via placeholders).
	encoded := base64.StdEncoding.EncodeToString(pngBuf.Bytes())
	chunkSize := 4096
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		more := 1
		if end == len(encoded) {
			more = 0
		}
		var cmd string
		if i == 0 {
			cmd = fmt.Sprintf("\033_Ga=T,t=d,f=100,i=%d,U=1,q=2,m=%d;%s\033\\", imgID, more, chunk)
		} else {
			cmd = fmt.Sprintf("\033_Gm=%d;%s\033\\", more, chunk)
		}
		writeToTTY(cmd)
	}

	// Create virtual placement scaled to cols × rows cells.
	placementCmd := fmt.Sprintf("\033_Ga=p,U=1,i=%d,p=%d,c=%d,r=%d,q=2\033\\",
		imgID, placementID, cols, rows)
	writeToTTY(placementCmd)

	log.Printf("image/kitty-upload: uploaded imgID=%d placementID=%d cols=%d rows=%d (src %dx%d px) pngBytes=%d",
		imgID, placementID, cols, rows, b.Dx(), b.Dy(), pngBuf.Len())
	return imgID, placementID, cols, rows
}

// kittyRowDiacritics is the ordered list of combining characters used to encode
// row indices in Kitty Unicode placeholder mode.  The index is the row number.
// These are NOT a simple sequential range — they come from the official table:
// https://sw.kovidgoyal.net/kitty/_downloads/f0a0de9ec8d9ff4456206db8e0814937/rowcolumn-diacritics.txt
var kittyRowDiacritics = []rune{
	'\u0305', '\u030D', '\u030E', '\u0310', '\u0312', '\u033D', '\u033E', '\u033F',
	'\u0346', '\u034A', '\u034B', '\u034C', '\u0350', '\u0351', '\u0352', '\u0357',
	'\u035B', '\u0363', '\u0364', '\u0365', '\u0366', '\u0367', '\u0368', '\u0369',
	'\u036A', '\u036B', '\u036C', '\u036D', '\u036E', '\u036F',
}

// BuildKittyPlaceholder returns a grid of U+10EEEE placeholder characters that
// the terminal replaces with actual image pixels.  The image ID is encoded as
// the foreground color and the placement ID as the underline color.  Each
// placeholder is followed by the correct Kitty row diacritic so the terminal
// knows which image row to sample for that line.
//
// The returned string is safe to embed in Lipgloss-rendered content because it
// contains only regular Unicode text and standard ANSI color sequences.
func BuildKittyPlaceholder(imgID, placementID uint32, cols, rows int) string {
	ir, ig, ib := (imgID>>16)&0xff, (imgID>>8)&0xff, imgID&0xff
	pr, pg, pb := (placementID>>16)&0xff, (placementID>>8)&0xff, placementID&0xff

	// \033[38;2;R;G;Bm = set fg color (image ID)
	// \033[58;2;R;G;Bm = set underline color (placement ID)
	style := fmt.Sprintf("\033[38;2;%d;%d;%dm\033[58;2;%d;%d;%dm",
		ir, ig, ib, pr, pg, pb)
	reset := "\033[0m"
	placeholder := "\U0010EEEE"

	var sb strings.Builder
	for row := 0; row < rows; row++ {
		sb.WriteString(style)
		// Each placeholder must be followed by the correct Kitty row diacritic.
		// These are a specific non-sequential set of combining chars; using
		// simple arithmetic (0x0305 + row) produces invalid diacritics for
		// rows ≥ 1 that Ghostty ignores, causing all rows to sample row 0.
		rowDiacritic := string(kittyRowDiacritics[row%len(kittyRowDiacritics)])
		for col := 0; col < cols; col++ {
			sb.WriteString(placeholder)
			sb.WriteString(rowDiacritic)
		}
		sb.WriteString(reset)
		if row < rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// DeleteKittyImages sends Kitty delete commands for the given image IDs to free
// terminal GPU memory.  Call this when switching channels.
func DeleteKittyImages(ids []uint32) {
	if len(ids) == 0 {
		return
	}
	initTermProto()
	if !termKitty {
		return
	}
	for _, id := range ids {
		cmd := fmt.Sprintf("\033_Ga=d,d=I,i=%d,q=2\033\\", id)
		writeToTTY(cmd)
	}
	log.Printf("image/kitty-delete: deleted %d images", len(ids))
}

// writeToTTY writes raw escape sequences directly to /dev/tty, bypassing
// Bubbletea's output.  When inside tmux, sequences are wrapped in DCS
// passthrough envelopes.
func writeToTTY(cmd string) {
	if termInTmux {
		cmd = wrapTmuxPassthrough(cmd)
	}
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		log.Printf("image/tty: open /dev/tty error — %v", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(cmd); err != nil {
		log.Printf("image/tty: write error — %v", err)
	}
}

// ─── public rendering entry-points ─────────────────────────────────────────

// RenderImageProtocol decodes raw image bytes and renders them using the best
// protocol available in the current terminal: Kitty > iTerm2 > half-block.
//
// The returned string is designed to be embedded in a Bubbletea View() string.
// It includes the escape sequences for the chosen protocol followed by enough
// newline characters so that Bubbletea's internal line counter correctly
// accounts for the height the image will occupy.
func RenderImageProtocol(data []byte, maxCols, maxRows int) string {
	if len(data) == 0 || maxCols < 1 || maxRows < 1 {
		log.Printf("image/render: skip — dataLen=%d maxCols=%d maxRows=%d", len(data), maxCols, maxRows)
		return ""
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("image/render: decode error — %v", err)
		return ""
	}
	bounds := img.Bounds()
	log.Printf("image/render: decoded format=%q size=%dx%d → target %dx%d cells",
		format, bounds.Dx(), bounds.Dy(), maxCols, maxRows)

	initTermProto()

	if termKitty {
		log.Printf("image/render: trying Kitty protocol")
		if s := renderKittyImg(img, maxCols, maxRows); s != "" {
			log.Printf("image/render: Kitty OK — output %d bytes", len(s))
			return s
		}
		log.Printf("image/render: Kitty failed, falling through")
	}
	if termIterm {
		log.Printf("image/render: trying iTerm2 protocol")
		if s := renderItermImg(img, maxCols, maxRows); s != "" {
			log.Printf("image/render: iTerm2 OK — output %d bytes", len(s))
			return s
		}
		log.Printf("image/render: iTerm2 failed, falling through")
	}
	log.Printf("image/render: using half-block fallback")
	return renderHalfBlock(img, maxCols, maxRows)
}

// RenderEmojiHalfBlock renders a custom emoji image as a tiny inline half-block
// thumbnail (InlineEmojiCols × InlineEmojiRows cells).  The trailing newline is
// stripped so the art can be inlined into a text flow.
// Returns an empty string when the data cannot be decoded.
//
// Custom emoji use the half-block renderer intentionally: they are substituted
// inline within glamour-formatted text where the Kitty/iTerm2 cursor-movement
// semantics would misalign subsequent text.
func RenderEmojiHalfBlock(data []byte) string {
	art := RenderImageHalfBlock(data, InlineEmojiCols, InlineEmojiRows)
	return strings.TrimRight(art, "\n")
}

// RenderImageHalfBlock decodes raw image bytes (PNG/JPEG/GIF/WebP) and renders
// them as terminal half-block (▄) pixel art using 24-bit true-color ANSI
// sequences.  Used as the universal fallback when no graphics protocol is
// available and as the emoji renderer.
//
// Each terminal cell represents two vertical pixels: the top pixel becomes the
// cell background and the bottom pixel becomes the foreground of ▄.
// Returns an empty string when the data cannot be decoded.
func RenderImageHalfBlock(data []byte, maxCols, maxRows int) string {
	if len(data) == 0 || maxCols < 1 || maxRows < 1 {
		return ""
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	return renderHalfBlock(img, maxCols, maxRows)
}

// ─── protocol-specific renderers ───────────────────────────────────────────

// renderKittyImg renders img using the Kitty Graphics Protocol.
//
// Cursor alignment: after the Kitty APC sequence the terminal cursor sits at
// the bottom-right of the image area (startRow + DstRows − 1, DstCols).  To
// keep Bubbletea's newline-based line counter in sync we:
//  1. Move the cursor back up to the image's top row with \033[{n}A.
//  2. Emit DstRows newlines, advancing the cursor to (startRow + DstRows, 0).
//
// Bubbletea counts DstRows newlines → its model matches the terminal cursor.
func renderKittyImg(img image.Image, maxCols, maxRows int) string {
	var buf strings.Builder
	opts := rasterm.KittyImgOpts{
		DstCols: uint32(maxCols),
		DstRows: uint32(maxRows),
	}
	if err := rasterm.KittyWriteImage(&buf, img, opts); err != nil {
		log.Printf("image/kitty: KittyWriteImage error — %v", err)
		return ""
	}

	// When inside tmux, the Kitty APC sequences must be wrapped in DCS
	// passthrough envelopes so the outer terminal actually receives them.
	kittySeq := buf.String()
	if termInTmux {
		kittySeq = wrapTmuxPassthrough(kittySeq)
		log.Printf("image/kitty: wrapped %d bytes in tmux DCS passthrough → %d bytes", buf.Len(), len(kittySeq))
	}

	var out strings.Builder
	out.WriteString(kittySeq)
	// Cursor is now at (R + maxRows − 1, maxCols).
	// Move up maxRows−1 rows to align with the image's top row.
	if maxRows > 1 {
		fmt.Fprintf(&out, "\033[%dA", maxRows-1)
	}
	// Advance through all image rows (newlines don't erase pixel content).
	for i := 0; i < maxRows; i++ {
		out.WriteByte('\n')
	}
	return out.String()
}

// renderItermImg renders img using the iTerm2 / WezTerm inline-image protocol.
// Applies the same cursor-alignment trick as renderKittyImg.
func renderItermImg(img image.Image, maxCols, maxRows int) string {
	var buf strings.Builder
	opts := rasterm.ItermImgOpts{
		Width:         strconv.Itoa(maxCols),
		Height:        strconv.Itoa(maxRows),
		DisplayInline: true,
	}
	if err := rasterm.ItermWriteImageWithOptions(&buf, img, opts); err != nil {
		log.Printf("image/iterm: ItermWriteImageWithOptions error — %v", err)
		return ""
	}
	if maxRows > 1 {
		fmt.Fprintf(&buf, "\033[%dA", maxRows-1)
	}
	for i := 0; i < maxRows; i++ {
		buf.WriteByte('\n')
	}
	return buf.String()
}

// ─── tmux DCS passthrough ───────────────────────────────────────────────────

// wrapTmuxPassthrough wraps each Kitty APC sequence (\033_G...\033\\) in a
// tmux DCS passthrough envelope so the outer terminal (e.g. Ghostty) receives
// it.  Every ESC byte inside the payload is doubled per the tmux passthrough
// specification.  Requires tmux ≥ 3.3 with "set -g allow-passthrough on".
func wrapTmuxPassthrough(raw string) string {
	var out strings.Builder
	i := 0
	for i < len(raw) {
		// Find next APC start: ESC _
		start := strings.Index(raw[i:], "\033_")
		if start < 0 {
			out.WriteString(raw[i:])
			break
		}
		out.WriteString(raw[i : i+start]) // pass through non-APC content
		i += start

		// Find APC end: ESC backslash
		end := strings.Index(raw[i:], "\033\\")
		if end < 0 {
			out.WriteString(raw[i:])
			break
		}
		end += 2 // include the ESC \

		apc := raw[i : i+end]

		// Wrap: \033Ptmux; <payload with doubled ESC> \033\\
		out.WriteString("\033Ptmux;")
		for j := 0; j < len(apc); j++ {
			if apc[j] == 0x1b {
				out.WriteByte(0x1b)
				out.WriteByte(0x1b)
			} else {
				out.WriteByte(apc[j])
			}
		}
		out.WriteString("\033\\")

		i += end
	}
	return out.String()
}

// ─── half-block fallback ────────────────────────────────────────────────────

// renderHalfBlock scales img to fit in maxCols × maxRows terminal cells and
// renders it using ▄ half-block characters with ANSI true-color sequences.
func renderHalfBlock(img image.Image, maxCols, maxRows int) string {
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return ""
	}

	// Each terminal row holds 2 pixel rows.
	pixW := maxCols
	pixH := maxRows * 2

	// Scale uniformly to fit, preserving aspect ratio.
	scaleX := float64(pixW) / float64(srcW)
	scaleY := float64(pixH) / float64(srcH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	// pixelAt maps a destination pixel to a source colour via nearest-neighbour.
	pixelAt := func(px, py int) color.RGBA {
		sx := b.Min.X + px*srcW/dstW
		sy := b.Min.Y + py*srcH/dstH
		if sx >= b.Max.X {
			sx = b.Max.X - 1
		}
		if sy >= b.Max.Y {
			sy = b.Max.Y - 1
		}
		r, g, bv, _ := img.At(sx, sy).RGBA()
		return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bv >> 8), 255}
	}

	var sb strings.Builder
	for row := 0; row < dstH; row += 2 {
		for col := 0; col < dstW; col++ {
			top := pixelAt(col, row)
			bot := top // same colour when image has odd height
			if row+1 < dstH {
				bot = pixelAt(col, row+1)
			}
			// ▄ = lower-half-block: fg = bottom pixel, bg = top pixel.
			fmt.Fprintf(&sb,
				"\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▄\x1b[0m",
				bot.R, bot.G, bot.B,
				top.R, top.G, top.B,
			)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
