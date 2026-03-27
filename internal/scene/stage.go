package scene

import (
	"fmt"
	"image/color"
	"log"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// StageScene is the main gameplay scene.
type StageScene struct {
	switcher Switcher
	frame    int
	gameMap  *gamemap.GameMap
}

// NewStageScene creates a new gameplay scene with map_01 loaded.
func NewStageScene(sw Switcher) *StageScene {
	cfg, err := config.LoadMap("map_01")
	if err != nil {
		log.Printf("failed to load map: %v", err)
		cfg = &config.MapConfig{
			ID: "fallback", Cols: 20, Rows: 9, CellSize: 60,
			Grid: make([][]int, 9),
		}
		for i := range cfg.Grid {
			cfg.Grid[i] = make([]int, 20)
		}
	}
	gm := gamemap.NewGameMap(cfg)
	return &StageScene{
		switcher: sw,
		gameMap:  gm,
	}
}

func (s *StageScene) Update() error {
	s.frame++
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}
	return nil
}

func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 40, B: 30, A: 255})

	// Draw map
	render.DrawMap(screen, s.gameMap)

	// Debug info
	ebitenutil.DebugPrintAt(screen,
		fmt.Sprintf("Map: %s  Waves: %d  Press ESC to return",
			s.gameMap.Config.Name, s.gameMap.Config.Waves),
		game.ScreenWidth/2-120, 4)
}
