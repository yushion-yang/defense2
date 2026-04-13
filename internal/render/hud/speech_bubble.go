// speech_bubble.go — Reusable speech bubble component for the mascot guide system.
// Renders a white rounded-rect bubble with wrapped text above an anchor point.
package hud

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// Speech bubble colors.
var (
	bubbleBg     = color.RGBA{R: 255, G: 255, B: 255, A: 235}
	bubbleBorder = color.RGBA{R: 160, G: 140, B: 200, A: 180}
	bubbleText   = color.RGBA{R: 40, G: 35, B: 55, A: 255}
)

// Speech bubble layout constants.
const (
	bubblePadH  float32 = 14  // horizontal padding
	bubblePadV  float32 = 10  // vertical padding
	bubbleR     float32 = 10  // corner radius
	bubbleGap   float32 = 6   // gap between bubble bottom and anchor
	bubbleStrokeW float32 = 1.2
	bubbleMargin float32 = 4  // min distance from screen edge
)

// SpeechBubbleVM holds data for rendering a speech bubble.
type SpeechBubbleVM struct {
	Text    string
	AnchorX float32 // tail points to this X
	AnchorY float32 // tail points to this Y (mascot top)
	MaxW    float32 // max bubble width
}

// DrawSpeechBubble renders a white rounded rect bubble with text above the anchor point.
func DrawSpeechBubble(screen *ebiten.Image, vm SpeechBubbleVM) {
	if vm.Text == "" {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	maxW := vm.MaxW
	if maxW <= 0 {
		maxW = 240
	}

	// Wrap text to fit within bubble.
	contentW := float64(maxW - bubblePadH*2)
	lines := ui.WrapText(fm, vm.Text, contentW, theme.FontBody)
	if len(lines) == 0 {
		return
	}

	// Measure actual content dimensions.
	lineH := float64(theme.FontBody) + 3 // line height with small leading
	textH := float32(lineH * float64(len(lines)))

	// Find widest line to size the bubble tightly.
	var maxLineW float64
	for _, line := range lines {
		w := fm.MeasureText(line, theme.FontBody)
		if w > maxLineW {
			maxLineW = w
		}
	}

	bubbleW := float32(maxLineW) + bubblePadH*2
	if bubbleW > maxW {
		bubbleW = maxW
	}
	bubbleH := textH + bubblePadV*2

	// Position bubble centered above anchor.
	bubbleX := vm.AnchorX - bubbleW/2
	bubbleY := vm.AnchorY - bubbleH - bubbleGap

	// Clamp to screen edges.
	if bubbleX < bubbleMargin {
		bubbleX = bubbleMargin
	}
	if bubbleX+bubbleW > float32(theme.CanvasW)-bubbleMargin {
		bubbleX = float32(theme.CanvasW) - bubbleMargin - bubbleW
	}
	if bubbleY < bubbleMargin {
		bubbleY = bubbleMargin
	}

	// Draw background.
	draw.RoundRect(screen, bubbleX, bubbleY, bubbleW, bubbleH, bubbleR, bubbleBg)

	// Draw border.
	draw.StrokeRoundRect(screen, bubbleX, bubbleY, bubbleW, bubbleH, bubbleR, bubbleStrokeW, bubbleBorder)

	// Draw text lines.
	textX := float64(bubbleX + bubblePadH)
	textY := float64(bubbleY + bubblePadV)
	for i, line := range lines {
		fm.DrawText(screen, line, textX, textY+float64(i)*lineH, theme.FontBody, bubbleText)
	}
}

