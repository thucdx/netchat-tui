package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

func wrapTmuxPassthrough(raw string) string {
	var out strings.Builder
	i := 0
	for i < len(raw) {
		start := strings.Index(raw[i:], "\033_")
		if start < 0 {
			out.WriteString(raw[i:])
			break
		}
		out.WriteString(raw[i : i+start])
		i += start
		end := strings.Index(raw[i:], "\033\\")
		if end < 0 {
			out.WriteString(raw[i:])
			break
		}
		end += 2
		apc := raw[i : i+end]
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

func emit(s string) {
	if os.Getenv("TMUX") != "" {
		fmt.Print(wrapTmuxPassthrough(s))
	} else {
		fmt.Print(s)
	}
}

// uploadImage transmits a PNG image with U=1 (Unicode placeholder mode).
// The image is stored but NOT displayed until placeholders appear.
func uploadImage(imgID uint32, pngData []byte) {
	encoded := base64.StdEncoding.EncodeToString(pngData)
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
		if i == 0 {
			emit(fmt.Sprintf("\033_Ga=T,t=d,f=100,i=%d,U=1,q=2,m=%d;%s\033\\", imgID, more, chunk))
		} else {
			emit(fmt.Sprintf("\033_Gm=%d;%s\033\\", more, chunk))
		}
	}
}

// createPlacement creates a virtual placement that scales the image to
// fit exactly cols × rows terminal cells.
func createPlacement(imgID, placementID uint32, cols, rows int) {
	cmd := fmt.Sprintf("\033_Ga=p,U=1,i=%d,p=%d,c=%d,r=%d,q=2\033\\",
		imgID, placementID, cols, rows)
	emit(cmd)
}

// kittyRowDiacritics is the ordered set of combining characters for Kitty row
// encoding.  The index is the row number.  NOT a sequential range — must use
// this specific table from the Kitty spec.
// https://sw.kovidgoyal.net/kitty/_downloads/f0a0de9ec8d9ff4456206db8e0814937/rowcolumn-diacritics.txt
var kittyRowDiacritics = []rune{
	'\u0305', '\u030D', '\u030E', '\u0310', '\u0312', '\u033D', '\u033E', '\u033F',
	'\u0346', '\u034A', '\u034B', '\u034C', '\u0350', '\u0351', '\u0352', '\u0357',
	'\u035B', '\u0363', '\u0364', '\u0365', '\u0366', '\u0367', '\u0368', '\u0369',
	'\u036A', '\u036B', '\u036C', '\u036D', '\u036E', '\u036F',
}

// renderPlaceholders emits a grid of U+10EEEE characters.
//   - Foreground color encodes the image ID (24-bit RGB).
//   - Underline color encodes the placement ID (24-bit RGB).
//   - Each placeholder is followed by the correct Kitty row diacritic.
func renderPlaceholders(imgID, placementID uint32, cols, rows int) string {
	// Encode IDs as 24-bit colors
	ir, ig, ib := (imgID>>16)&0xff, (imgID>>8)&0xff, imgID&0xff
	pr, pg, pb := (placementID>>16)&0xff, (placementID>>8)&0xff, placementID&0xff

	// \033[38;2;R;G;Bm  = set fg color (image ID)
	// \033[58;2;R;G;Bm  = set underline color (placement ID)
	style := fmt.Sprintf("\033[38;2;%d;%d;%dm\033[58;2;%d;%d;%dm",
		ir, ig, ib, pr, pg, pb)
	reset := "\033[0m"
	placeholder := "\U0010EEEE"

	var sb strings.Builder
	for row := range rows {
		rowDiacritic := string(kittyRowDiacritics[row%len(kittyRowDiacritics)])
		sb.WriteString(style)
		for range cols {
			sb.WriteString(placeholder)
			sb.WriteString(rowDiacritic)
		}
		sb.WriteString(reset)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func main() {
	fmt.Fprintf(os.Stderr, "TERM=%s  TMUX=%s  GHOSTTY_RESOURCES_DIR=%s\n",
		os.Getenv("TERM"), os.Getenv("TMUX"), os.Getenv("GHOSTTY_RESOURCES_DIR"))

	// Create a 400×240 gradient image (more pixels to see scaling quality)
	img := image.NewRGBA(image.Rect(0, 0, 400, 240))
	for y := range 240 {
		for x := range 400 {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / 400),
				G: uint8(y * 255 / 240),
				B: 128,
				A: 255,
			})
		}
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	pngData := pngBuf.Bytes()
	fmt.Fprintf(os.Stderr, "image: 400x240 pixels, PNG %d bytes\n", len(pngData))

	imgID := uint32(1)

	// Upload image (stored, not displayed)
	uploadImage(imgID, pngData)

	// --- Test A: 30×8, with virtual placement ---
	placementA := uint32(10)
	createPlacement(imgID, placementA, 30, 8)
	fmt.Println("=== Test A: 30×8 cells (virtual placement) ===")
	fmt.Print(renderPlaceholders(imgID, placementA, 30, 8))
	fmt.Println("=== end A ===")
	fmt.Println()

	// --- Test B: 60×16, with virtual placement ---
	placementB := uint32(20)
	createPlacement(imgID, placementB, 60, 16)
	fmt.Println("=== Test B: 60×16 cells (virtual placement) ===")
	fmt.Print(renderPlaceholders(imgID, placementB, 60, 16))
	fmt.Println("=== end B ===")
	fmt.Println()

	// --- Test C: 80×24, with virtual placement ---
	placementC := uint32(30)
	createPlacement(imgID, placementC, 80, 24)
	fmt.Println("=== Test C: 80×24 cells (virtual placement) ===")
	fmt.Print(renderPlaceholders(imgID, placementC, 80, 24))
	fmt.Println("=== end C ===")
}
