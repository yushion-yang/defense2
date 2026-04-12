// mascot_sprite.go — Mascot sprite loader for the guide system.
// Loads mascot expression PNGs into an anim.Animator.
package render

import (
	"bytes"
	"fmt"
	"image/png"
	"log"

	"defense2/internal/render/anim"

	"github.com/hajimehoshi/ebiten/v2"
)

// MascotAnimConfig defines animation parameters per expression.
var MascotAnimConfig = map[string]struct {
	FPS  float64
	Loop bool
}{
	"idle":      {FPS: 4, Loop: true},
	"talk":      {FPS: 6, Loop: true},
	"happy":     {FPS: 4, Loop: true},
	"surprised": {FPS: 4, Loop: false},
}

// LoadMascotSprites loads mascot expression PNGs into an Animator.
// Looks for: assets/mascot/mascot-{expression}-{frame}.png (multi-frame)
// Fallback: assets/mascot/mascot-{expression}.png (single frame)
// Expressions: idle, talk, happy, surprised.
// Missing files are skipped gracefully.
func LoadMascotSprites(fs AssetReader) *anim.Animator {
	a := anim.NewAnimator()
	if fs == nil {
		return a
	}

	for expr, cfg := range MascotAnimConfig {
		frames := loadMascotFrames(fs, expr)
		if len(frames) > 0 {
			a.AddAnim(expr, frames, cfg.FPS, cfg.Loop)
		}
	}

	// If no animation frames found, try loading a single fallback PNG.
	if len(a.Anims) == 0 {
		img := loadMascotSinglePNG(fs, "assets/mascot/mascot.png")
		if img != nil {
			a.AddAnim("idle", []*ebiten.Image{img}, 1, true)
		}
	}

	return a
}

// loadMascotFrames tries to load multi-frame PNGs: mascot-{expr}-0.png, -1.png, ...
// Falls back to single frame: mascot-{expr}.png
func loadMascotFrames(fs AssetReader, expr string) []*ebiten.Image {
	prefix := fmt.Sprintf("assets/mascot/mascot-%s", expr)

	// Try multi-frame first.
	var frames []*ebiten.Image
	for i := 0; i < 16; i++ {
		path := fmt.Sprintf("%s-%d.png", prefix, i)
		img := loadMascotSinglePNG(fs, path)
		if img == nil {
			break
		}
		frames = append(frames, img)
	}
	if len(frames) > 0 {
		return frames
	}

	// Fall back to single frame.
	img := loadMascotSinglePNG(fs, prefix+".png")
	if img != nil {
		return []*ebiten.Image{img}
	}
	return nil
}

// loadMascotSinglePNG loads a single PNG file, returning nil on failure (silent).
func loadMascotSinglePNG(fs AssetReader, path string) *ebiten.Image {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("mascot: PNG decode failed %s: %v", path, err)
		return nil
	}
	return ebiten.NewImageFromImage(img)
}
