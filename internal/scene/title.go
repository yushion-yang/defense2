package scene

import (
	"image/color"

	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TitleScene displays the title screen.
type TitleScene struct {
	switcher Switcher
}

// NewTitleScene creates a new title scene.
func NewTitleScene(sw Switcher) *TitleScene {
	return &TitleScene{switcher: sw}
}

func (s *TitleScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.JustPressedTouchIDs()) > 0 {
		s.switcher.SwitchScene(NewStageScene(s.switcher))
	}
	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 35, B: 20, A: 255})
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	ebitenutil.DebugPrintAt(screen, "TOWER DEFENSE", cx-40, cy-20)
	ebitenutil.DebugPrintAt(screen, "ENTER / TAP to start", cx-58, cy+20)
}
