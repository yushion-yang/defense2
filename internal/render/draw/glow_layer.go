// glow_layer.go — offscreen glow compositing.
// Batches all Glow() draws to a separate image, then composites with
// additive blending (BlendLighter) for a bloom-like effect without shaders.
package draw

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	glowTarget *ebiten.Image
	glowActive bool
)

// BeginGlowPass starts capturing glow draws to an offscreen buffer.
// All subsequent Glow() calls will draw to this buffer instead of their
// screen parameter. Non-glow draws (FilledCircle, Line, etc.) are unaffected.
func BeginGlowPass(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if glowTarget == nil || glowTarget.Bounds() != image.Rect(0, 0, w, h) {
		glowTarget = ebiten.NewImage(w, h)
	}
	glowTarget.Clear()
	glowActive = true
}

// EndGlowPass composites the glow buffer onto the screen with additive blending
// and resets the glow state. Safe to call even if BeginGlowPass was not called.
func EndGlowPass(screen *ebiten.Image) {
	if !glowActive || glowTarget == nil {
		glowActive = false
		return
	}
	glowActive = false

	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendLighter
	screen.DrawImage(glowTarget, op)
}

// GlowTarget returns the current glow render target.
// Returns nil when no glow pass is active.
func GlowTarget() *ebiten.Image {
	if glowActive {
		return glowTarget
	}
	return nil
}

// GlowPassActive returns true if a glow pass is currently in progress.
func GlowPassActive() bool {
	return glowActive
}
