package scene

import (
	"image/color"

	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// StageScene is the main gameplay scene.
type StageScene struct {
	switcher Switcher
	frame    int
}

// NewStageScene creates a new gameplay scene.
func NewStageScene(sw Switcher) *StageScene {
	return &StageScene{switcher: sw}
}

func (s *StageScene) Update() error {
	s.frame++
	// ESC returns to title
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}
	return nil
}

func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 50, B: 30, A: 255})
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	ebitenutil.DebugPrintAt(screen, "STAGE (empty - P1 will add gameplay)", cx-100, cy)
	ebitenutil.DebugPrintAt(screen, "Press ESC to return", cx-55, cy+20)
}
