// mascot_overlay.go — Mascot character overlay for the guide system.
// Combines a mascot sprite with an optional speech bubble.
// Pure render component — zero core dependency.
package hud

import (
	"math"

	"defense2/internal/render/draw" //nolint:hud — draw.SpriteScaled/RoundRect for sprite rendering
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// Mascot overlay layout constants.
const (
	mascotMarginRight float32 = 12  // distance from right edge
	mascotSpriteW     float32 = 80  // sprite area width
	mascotSpriteH     float32 = 160 // sprite area height (includes bob range)
	mascotBubbleMaxW  float32 = 240 // max speech bubble width
)

// mascotPlaceholderClr is the fallback color when no sprite is available.
var mascotPlaceholderClr = theme.PanelBg

// MascotOverlayVM is the view-model for the mascot overlay.
type MascotOverlayVM struct {
	Visible         bool
	HasDialog       bool
	Text            string
	Expression      string
	CanClick        bool
	Sprite          *ebiten.Image // current frame (nil = use placeholder)
	AnimTime        float64       // for idle bob
	AbilityReady    bool          // true when mascot ability is usable
	AbilityHintText string        // 独立技能提示文本（可与 Text 同时显示）
	CooldownPct     float64       // 0.0 = ready, 1.0 = full cooldown
}

// mascotBasePos returns the mascot center-bottom position in logical coordinates.
// Bottom-right corner, above BottomMargin + ActionBarH.
func mascotBasePos() (cx, bottomY float32) {
	cx = float32(theme.CanvasW) - mascotMarginRight - mascotSpriteW/2
	bottomY = float32(theme.CanvasH) - float32(theme.BottomMargin) - float32(theme.ActionBarH)
	return
}

// MascotHitTest returns true if (x, y) is within the mascot clickable region.
func MascotHitTest(x, y float64) bool {
	cx, bottomY := mascotBasePos()
	left := float64(cx - mascotSpriteW/2)
	right := float64(cx + mascotSpriteW/2)
	top := float64(bottomY - mascotSpriteH)
	bottom := float64(bottomY)
	return x >= left && x <= right && y >= top && y <= bottom
}

// DrawMascotOverlay renders the mascot character and optional speech bubble.
func DrawMascotOverlay(screen *ebiten.Image, vm MascotOverlayVM) {
	if !vm.Visible {
		return
	}

	cx, bottomY := mascotBasePos()

	// Idle bob animation.
	bobY := float32(math.Sin(vm.AnimTime*2.5) * 2.0)

	// Sprite center position (vertically centered in sprite area, with bob).
	spriteCX := float64(cx)
	spriteCY := float64(bottomY-mascotSpriteH/2) + float64(bobY)

	if vm.Sprite != nil {
		// Calculate scale to fit sprite within the 80x160 area.
		imgW := float64(vm.Sprite.Bounds().Dx())
		imgH := float64(vm.Sprite.Bounds().Dy())
		if imgW > 0 && imgH > 0 {
			scaleW := float64(mascotSpriteW) / imgW
			scaleH := float64(mascotSpriteH) / imgH
			scale := scaleW
			if scaleH < scaleW {
				scale = scaleH
			}
			draw.SpriteScaled(screen, vm.Sprite, spriteCX, spriteCY, scale) //nolint:hud
		}
	} else {
		// Placeholder rounded rect.
		phW := mascotSpriteW * 0.6
		phH := mascotSpriteH * 0.7
		phX := cx - phW/2
		phY := float32(spriteCY) - phH/2
		ui.Panel(screen, phX, phY, phW, phH, ui.PanelStyle{
			BgColor: mascotPlaceholderClr, Radius: 12,
		})
	}

	// ── 双气泡渲染 ──
	// 普通对话气泡（白色）在萌妹头顶，技能提示气泡（粉色）在更上方。
	// 两者可同时存在，互不干扰。
	anchorX := cx
	anchorY := bottomY - mascotSpriteH + float32(bobY)

	// 普通对话气泡
	var dialogBubbleH float32
	if vm.HasDialog && vm.Text != "" {
		dialogBubbleH = measureBubbleHeight(screen, vm.Text, mascotBubbleMaxW)
		DrawSpeechBubble(screen, SpeechBubbleVM{
			Text:    vm.Text,
			AnchorX: anchorX,
			AnchorY: anchorY,
			MaxW:    mascotBubbleMaxW,
		})
	}

	// 技能提示气泡（独立通道，粉紫配色）
	if vm.AbilityHintText != "" {
		hintAnchorY := anchorY
		if dialogBubbleH > 0 {
			// 有普通对话时，技能提示在普通气泡上方
			hintAnchorY = anchorY - dialogBubbleH - bubbleGap*2
		}
		DrawAbilityHintBubble(screen, SpeechBubbleVM{
			Text:    vm.AbilityHintText,
			AnchorX: anchorX,
			AnchorY: hintAnchorY,
			MaxW:    mascotBubbleMaxW,
		})
	}
}
