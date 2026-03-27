package scene

import (
	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game implements ebiten.Game and manages scene transitions.
type Game struct {
	current Scene
	next    Scene
	width   int
	height  int
}

// NewGame creates a new Game starting at the title scene.
func NewGame() *Game {
	g := &Game{
		width:  game.ScreenWidth,
		height: game.ScreenHeight,
	}
	g.current = NewTitleScene(g)
	return g
}

// SwitchScene queues a scene transition for the next frame.
func (g *Game) SwitchScene(next Scene) {
	g.next = next
}

func (g *Game) Update() error {
	if g.next != nil {
		g.current = g.next
		g.next = nil
	}
	return g.current.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.current.Draw(screen)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return g.width, g.height
}
