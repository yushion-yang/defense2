// wave_announce.go — Wave start announcement with slide-in/out animation.
// Shows dramatic text when waves start: normal waves, boss waves (every 5th),
// and the final wave each get distinct visual treatments.
package hud

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/easing"

	"github.com/hajimehoshi/ebiten/v2"
)

// announcePhase tracks the current stage of the wave announcement animation.
type announcePhase int

const (
	announceIdle     announcePhase = iota
	announceEntrance               // boss/final: extended warning flashes before slide-in
	announceSlideIn                // 0.3s: text slides down from top
	announceHold                   // 0.8s (1.0s for boss): text stays visible
	announceSlideOut               // 0.3s: text slides up and disappears
)

// Timing constants for the announcement animation.
const (
	slideInDuration      = 0.3
	holdNormal           = 0.8
	holdBoss             = 1.0
	slideOutDuration     = 0.3
	flashInterval        = 0.15 // normal boss warning flash interval
	entranceFlashCount   = 6    // boss entrance: 6 dramatic pulses
	entranceFlashInterval = 0.4 // boss entrance: slower interval for drama
	targetY              = 80.0 // vertical center position for the text
	offscreenY           = -40.0
)

// WaveAnnounce manages the animated text overlay shown when a new wave begins.
type WaveAnnounce struct {
	phase    announcePhase
	timer    float64
	wave     int
	maxWaves int
	isBoss   bool
	isFinal  bool

	// Boss warning: flash red border before showing wave text.
	warningFlashes      int     // remaining flashes (2 for normal boss, 6 for entrance)
	warningTimer        float64 // countdown to next flash toggle
	warningTotalFlashes int     // total flashes for this sequence (used for intensity calc)
}

// NewWaveAnnounce creates a new idle wave announcement overlay.
func NewWaveAnnounce() *WaveAnnounce {
	return &WaveAnnounce{}
}

// Trigger starts the announcement animation for the given wave.
func (wa *WaveAnnounce) Trigger(wave, maxWaves int, isBoss bool) {
	wa.wave = wave
	wa.maxWaves = maxWaves
	wa.isBoss = isBoss
	wa.isFinal = wave == maxWaves && maxWaves > 0
	if isBoss || wa.isFinal {
		// Boss/final waves: extended entrance phase with 6 dramatic pulses
		wa.phase = announceEntrance
		wa.timer = 0
		wa.warningFlashes = entranceFlashCount
		wa.warningTotalFlashes = entranceFlashCount
		wa.warningTimer = entranceFlashInterval
	} else {
		wa.phase = announceSlideIn
		wa.timer = slideInDuration
		wa.warningFlashes = 0
		wa.warningTotalFlashes = 0
	}
}

// Update advances the announcement state machine by dt seconds.
func (wa *WaveAnnounce) Update(dt float64) {
	if wa.phase == announceIdle {
		return
	}

	// Boss entrance phase: extended dramatic pulses before slide-in.
	if wa.phase == announceEntrance {
		wa.warningTimer -= dt
		if wa.warningTimer <= 0 {
			wa.warningFlashes--
			if wa.warningFlashes <= 0 {
				// Entrance done → transition to slide-in
				wa.phase = announceSlideIn
				wa.timer = slideInDuration
			} else {
				wa.warningTimer = entranceFlashInterval
			}
		}
		return
	}

	// Legacy warning flashes (kept for any non-entrance path).
	if wa.warningFlashes > 0 {
		wa.warningTimer -= dt
		if wa.warningTimer <= 0 {
			wa.warningFlashes--
			wa.warningTimer = flashInterval
		}
		return
	}

	wa.timer -= dt
	if wa.timer <= 0 {
		switch wa.phase {
		case announceSlideIn:
			hold := holdNormal
			if wa.isBoss || wa.isFinal {
				hold = holdBoss
			}
			wa.phase = announceHold
			wa.timer = hold
		case announceHold:
			wa.phase = announceSlideOut
			wa.timer = slideOutDuration
		case announceSlideOut:
			wa.phase = announceIdle
		}
	}
}

// Draw renders the announcement overlay onto the screen.
func (wa *WaveAnnounce) Draw(screen *ebiten.Image) {
	if wa.phase == announceIdle {
		return
	}

	// Draw entrance phase or legacy warning flashes.
	if wa.phase == announceEntrance {
		wa.drawEntranceFlash(screen)
		return
	}
	if wa.warningFlashes > 0 {
		wa.drawWarningFlash(screen)
		return
	}

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	cx := float64(game.ScreenWidth) / 2

	// Calculate Y position based on phase (slide in from top, hold, slide out to top).
	y := wa.currentY()

	// Determine text content, color, and font size.
	var text string
	var textClr color.RGBA
	fontSize := 28.0

	switch {
	case wa.isFinal:
		text = i18n.T("hud.wave.final")
		textClr = color.RGBA{R: 255, G: 215, B: 0, A: 255} // gold
		fontSize = 32
	case wa.isBoss:
		text = i18n.TF("hud.wave.wave_n", wa.wave)
		textClr = color.RGBA{R: 255, G: 80, B: 60, A: 255} // red-orange
		fontSize = 32
	default:
		text = i18n.TF("hud.wave.wave_n", wa.wave)
		textClr = color.RGBA{R: 255, G: 255, B: 255, A: 230}
	}

	// Background pill behind text.
	textW := fm.MeasureText(text, fontSize)
	pillW := float32(textW + 40)
	pillH := float32(fontSize + 16)
	pillX := float32(cx) - pillW/2
	pillY := float32(y) - pillH/2

	draw.RoundRect(screen, pillX, pillY, pillW, pillH, 8, color.RGBA{A: 140})

	// Centered text.
	fm.DrawCenteredText(screen, text, cx, y-fontSize/4, fontSize, textClr)
}

// currentY computes the vertical position based on the current animation phase.
func (wa *WaveAnnounce) currentY() float64 {
	switch wa.phase {
	case announceSlideIn:
		progress := 1.0 - wa.timer/slideInDuration // 0 → 1
		return offscreenY + (targetY-offscreenY)*easing.EaseOutQuad(progress)
	case announceHold:
		return targetY
	case announceSlideOut:
		progress := 1.0 - wa.timer/slideOutDuration // 0 → 1
		return targetY + (offscreenY-targetY)*easing.EaseInQuad(progress)
	default:
		return offscreenY
	}
}

// drawEntranceFlash renders dramatic escalating border flashes during the boss entrance phase.
// Each successive flash is brighter/wider: intensity = (total - remaining) / total.
func (wa *WaveAnnounce) drawEntranceFlash(screen *ebiten.Image) {
	// Flicker: visible on even "ticks" of the warning timer.
	if int(wa.warningTimer*20)%2 != 0 {
		return
	}

	// Intensity ramps up as flashes progress: 0.17 → 0.33 → ... → 1.0
	intensity := float64(wa.warningTotalFlashes-wa.warningFlashes) / float64(wa.warningTotalFlashes)
	if intensity < 0.15 {
		intensity = 0.15
	}

	baseAlpha := uint8(60 + intensity*140) // 60 → 200
	var borderClr color.RGBA
	if wa.isFinal {
		borderClr = color.RGBA{R: 255, G: 200, B: 50, A: baseAlpha} // gold
	} else {
		borderClr = color.RGBA{R: 255, G: 40, B: 40, A: baseAlpha} // red
	}

	w := float32(game.ScreenWidth)
	h := float32(game.ScreenHeight)
	thickness := float32(3 + intensity*4) // 3px → 7px

	draw.FilledRect(screen, 0, 0, w, thickness, borderClr, false)          // top
	draw.FilledRect(screen, 0, h-thickness, w, thickness, borderClr, false) // bottom
	draw.FilledRect(screen, 0, 0, thickness, h, borderClr, false)           // left
	draw.FilledRect(screen, w-thickness, 0, thickness, h, borderClr, false) // right
}

// drawWarningFlash renders pulsing red (boss) or gold (final) border edges.
func (wa *WaveAnnounce) drawWarningFlash(screen *ebiten.Image) {
	// Flicker: visible on even "ticks" of the warning timer.
	if int(wa.warningTimer*20)%2 != 0 {
		return
	}

	var borderClr color.RGBA
	if wa.isFinal {
		borderClr = color.RGBA{R: 255, G: 200, B: 50, A: 80} // gold
	} else {
		borderClr = color.RGBA{R: 255, G: 40, B: 40, A: 80} // red
	}

	w := float32(game.ScreenWidth)
	h := float32(game.ScreenHeight)
	const thickness float32 = 3

	draw.FilledRect(screen, 0, 0, w, thickness, borderClr, false)         // top
	draw.FilledRect(screen, 0, h-thickness, w, thickness, borderClr, false) // bottom
	draw.FilledRect(screen, 0, 0, thickness, h, borderClr, false)          // left
	draw.FilledRect(screen, w-thickness, 0, thickness, h, borderClr, false) // right
}
