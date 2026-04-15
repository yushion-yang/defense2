// speech_bubble.go — Reusable speech bubble component for the mascot guide system.
// Renders a white rounded-rect bubble with wrapped text above an anchor point.
package hud

import (
	"image/color"

	"defense2/internal/render"
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

// 技能提示气泡颜色（粉紫配色，与普通白色气泡区分）
var (
	abilityHintBg     = color.RGBA{R: 255, G: 230, B: 245, A: 240}
	abilityHintBorder = color.RGBA{R: 200, G: 120, B: 180, A: 200}
	abilityHintText   = color.RGBA{R: 120, G: 40, B: 100, A: 255}
)

// Speech bubble layout constants.
const (
	bubblePadH    float32 = 14 // horizontal padding
	bubblePadV    float32 = 10 // vertical padding
	bubbleR       float32 = 10 // corner radius
	bubbleGap     float32 = 6  // gap between bubble bottom and anchor
	bubbleStrokeW float32 = 1.2
	bubbleMargin  float32 = 4 // min distance from screen edge
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
	lineH := float64(theme.FontBody) + 3
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

	// Draw background + border.
	ui.Panel(screen, bubbleX, bubbleY, bubbleW, bubbleH, ui.PanelStyle{
		BgColor: bubbleBg, BorderColor: bubbleBorder, Radius: bubbleR, BorderWidth: bubbleStrokeW,
	})

	// Draw text lines.
	ui.Paragraph(screen, vm.Text, float64(bubbleX+bubblePadH), float64(bubbleY+bubblePadV),
		contentW, ui.ParagraphStyle{
			Font: theme.FontBody, Color: bubbleText, LineGap: 3,
		})
}

// measureBubbleHeight 计算气泡总高度（含 padding 和 gap），用于双气泡布局。
// screen 参数仅用于获取 font（保持与 DrawSpeechBubble 一致的排版）。
func measureBubbleHeight(_ *ebiten.Image, text string, maxW float32) float32 {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return 0
	}
	if maxW <= 0 {
		maxW = 240
	}
	contentW := float64(maxW - bubblePadH*2)
	lines := ui.WrapText(fm, text, contentW, theme.FontBody)
	if len(lines) == 0 {
		return 0
	}
	lineH := float64(theme.FontBody) + 3
	textH := float32(lineH * float64(len(lines)))
	return textH + bubblePadV*2 + bubbleGap
}

// DrawAbilityHintBubble 绘制技能提示气泡（粉紫配色），布局逻辑与 DrawSpeechBubble 相同。
func DrawAbilityHintBubble(screen *ebiten.Image, vm SpeechBubbleVM) {
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

	contentW := float64(maxW - bubblePadH*2)
	lines := ui.WrapText(fm, vm.Text, contentW, theme.FontBody)
	if len(lines) == 0 {
		return
	}

	lineH := float64(theme.FontBody) + 3
	textH := float32(lineH * float64(len(lines)))

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

	bubbleX := vm.AnchorX - bubbleW/2
	bubbleY := vm.AnchorY - bubbleH - bubbleGap

	if bubbleX < bubbleMargin {
		bubbleX = bubbleMargin
	}
	if bubbleX+bubbleW > float32(theme.CanvasW)-bubbleMargin {
		bubbleX = float32(theme.CanvasW) - bubbleMargin - bubbleW
	}
	if bubbleY < bubbleMargin {
		bubbleY = bubbleMargin
	}

	// 粉紫背景 + 边框
	ui.Panel(screen, bubbleX, bubbleY, bubbleW, bubbleH, ui.PanelStyle{
		BgColor: abilityHintBg, BorderColor: abilityHintBorder, Radius: bubbleR, BorderWidth: bubbleStrokeW,
	})

	ui.Paragraph(screen, vm.Text, float64(bubbleX+bubblePadH), float64(bubbleY+bubblePadV),
		contentW, ui.ParagraphStyle{
			Font: theme.FontBody, Color: abilityHintText, LineGap: 3,
		})
}
