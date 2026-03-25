package chat

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"                   // register GIF decoder
	_ "image/jpeg"                  // register JPEG decoder
	_ "image/png"                   // register PNG decoder
	"strings"

	_ "golang.org/x/image/webp"    // register WebP decoder (common for Mattermost custom emoji)
)

// InlineImageCols is the width (terminal cells) for inline thumbnail renders.
const InlineImageCols = 30

// InlineImageRows is the height (terminal rows) for inline thumbnail renders.
// Each row represents 2 pixels via the half-block character ▄.
const InlineImageRows = 8

// InlineEmojiCols is the width (terminal cells) for inline custom emoji renders.
// 4 cells keeps it roughly the same visual footprint as two Unicode emoji side-by-side.
const InlineEmojiCols = 4

// InlineEmojiRows is the height (terminal rows) for inline custom emoji renders.
// 2 rows = 4 pixel rows; enough detail to recognise a simple emoji shape.
const InlineEmojiRows = 2

// RenderEmojiHalfBlock renders a custom emoji image as a tiny inline half-block
// thumbnail (InlineEmojiCols × InlineEmojiRows cells).  The trailing newline is
// stripped so the art can be inlined into a text flow.
// Returns an empty string when the data cannot be decoded.
func RenderEmojiHalfBlock(data []byte) string {
	art := RenderImageHalfBlock(data, InlineEmojiCols, InlineEmojiRows)
	return strings.TrimRight(art, "\n")
}

// RenderImageHalfBlock decodes raw image bytes (PNG/JPEG/GIF) and renders them
// as terminal half-block (▄) pixel art using 24-bit true-color ANSI sequences.
//
// Each terminal cell represents two vertical pixels: the top pixel becomes the
// cell background and the bottom pixel becomes the foreground of ▄ (lower-half-
// block character).  The image is scaled to fit within maxCols × maxRows
// terminal cells while preserving its aspect ratio.
//
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
			sb.WriteString(fmt.Sprintf(
				"\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▄\x1b[0m",
				bot.R, bot.G, bot.B,
				top.R, top.G, top.B,
			))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
