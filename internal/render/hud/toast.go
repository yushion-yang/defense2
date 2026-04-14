// toast.go — Temporary on-screen notification toast.
// Shows a centered message that fades out after a short duration.
package hud

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	toastDuration = 1.5 // total display time in seconds
)

// Toast represents a temporary notification message.
type Toast struct {
	message string
	timer   float64
	alpha   float64
}

var activeToast *Toast

// ClearToast 清除当前活跃的 toast 通知。
func ClearToast() { activeToast = nil }

// ShowToast displays a new toast message, replacing any existing one.
func ShowToast(msg string) {
	activeToast = &Toast{
		message: msg,
		timer:   toastDuration,
		alpha:   1.0,
	}
}

// UpdateToast advances the toast timer and handles fade-out.
func UpdateToast(dt float64) {
	if activeToast == nil || activeToast.timer <= 0 {
		return
	}

	activeToast.timer -= dt
	if activeToast.timer <= 0 {
		activeToast = nil
		return
	}

	// Fade out during the last ToastFadeDuration seconds.
	if activeToast.timer < theme.ToastFadeDuration {
		activeToast.alpha = activeToast.timer / theme.ToastFadeDuration
	} else {
		activeToast.alpha = 1.0
	}
}

// DrawToast renders the active toast if present.
func DrawToast(screen *ebiten.Image) {
	if activeToast == nil || activeToast.timer <= 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		toastW = float32(theme.ToastW)
		toastH = float32(theme.ToastH)
		toastR = float32(theme.ToastRadius)
	)

	toastX := (float32(theme.CanvasW) - toastW) / 2
	toastY := float32(theme.TopBarY) + float32(theme.TopBarH) + 8

	// Modulate background alpha by fade.
	bg := theme.HUDToastBg
	bg.A = uint8(float64(bg.A) * activeToast.alpha)
	ui.Panel(screen, toastX, toastY, toastW, toastH, ui.PanelStyle{
		BgColor: bg, Radius: toastR,
	})

	// Text with faded alpha (shrink font if message too wide).
	msgSize := ui.ShrinkFontSize(fm, activeToast.message, 480, theme.FontMD, theme.FontSM)
	textAlpha := uint8(255 * activeToast.alpha)
	cx := float64(toastX) + float64(toastW)/2
	cy := float64(toastY) + float64(toastH)/2 - 6
	ui.LabelV(screen, activeToast.message, cx, cy, float64(toastW)-20, ui.LabelStyle{
		Font: msgSize, Color: color.RGBA{R: 255, G: 255, B: 255, A: textAlpha},
	})
}
