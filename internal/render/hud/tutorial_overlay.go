// tutorial_overlay.go — Tutorial guided overlay.
// Renders a bottom-center text box with the current tutorial instruction
// and a step indicator (e.g. "2/8").
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// TutorialVM is the view-model for the tutorial overlay.
type TutorialVM struct {
	Visible       bool
	Message       string
	Step          int  // 1-based current step
	Total         int  // total steps
	ClickToAdvance bool // show "点击继续" hint
}

// DrawTutorialOverlay renders the tutorial instruction box.
func DrawTutorialOverlay(screen *ebiten.Image, vm TutorialVM) {
	if !vm.Visible {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	// Box dimensions — centered, above the ActionBar.
	const (
		boxW    float32 = 480
		boxH    float32 = 52
		borderR float32 = 12
	)
	boxX := (float32(game.ScreenWidth) - boxW) / 2
	boxY := float32(game.ScreenHeight) - float32(theme.BottomMargin) - float32(theme.ActionBarH) - boxH - 12

	// Semi-transparent dark background.
	draw.RoundRect(screen, boxX, boxY, boxW, boxH, borderR,
		color.RGBA{R: 10, G: 15, B: 30, A: 220})
	// Subtle border.
	draw.StrokeRoundRect(screen, boxX, boxY, boxW, boxH, borderR, 1,
		color.RGBA{R: 100, G: 150, B: 255, A: 100})

	// Message text (centered in box, shrink font if too wide).
	msgSize := ui.ShrinkFontSize(fm, vm.Message, 460, theme.FontH2, theme.FontSM)
	msgY := float64(boxY) + float64(boxH)/2 - msgSize/2
	fm.DrawCenteredText(screen, vm.Message,
		float64(game.ScreenWidth)/2, msgY, msgSize, color.White)

	// Step indicator: "2/8" at bottom-right of box.
	stepTxt := strconv.Itoa(vm.Step) + "/" + strconv.Itoa(vm.Total)
	fm.DrawText(screen, stepTxt,
		float64(boxX+boxW)-36, float64(boxY+boxH)-14, theme.FontCaption, theme.TextMuted)

	// "点击继续" hint for click-to-advance steps.
	if vm.ClickToAdvance {
		hintY := float64(boxY+boxH) + 4
		fm.DrawCenteredText(screen, i18n.T("hud.tutorial.click_continue"),
			float64(game.ScreenWidth)/2, hintY, theme.FontCaption, theme.TextMuted)
	}
}
