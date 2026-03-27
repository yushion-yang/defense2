package scene

import (
	"fmt"
	"image/color"
	"log"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type stageState int

const (
	statePlaying stageState = iota
	stateVictory
	stateDefeat
)

// StageScene is the main gameplay scene.
type StageScene struct {
	switcher Switcher
	frame    int
	state    stageState
	gameMap  *gamemap.GameMap
	enemies  *enemy.Pool
	spawner  *enemy.Spawner
	lives    int
}

// NewStageScene creates a new gameplay scene with map_01 loaded.
func NewStageScene(sw Switcher) *StageScene {
	cfg, err := config.LoadMap("map_01")
	if err != nil {
		log.Printf("failed to load map: %v", err)
		cfg = &config.MapConfig{
			ID: "fallback", Cols: 20, Rows: 9, CellSize: 60,
			Grid:  make([][]int, 9),
			Waves: 3,
		}
		for i := range cfg.Grid {
			cfg.Grid[i] = make([]int, 20)
		}
	}
	gm := gamemap.NewGameMap(cfg)
	return &StageScene{
		switcher: sw,
		gameMap:  gm,
		enemies:  enemy.DefaultPool(),
		spawner:  enemy.NewSpawner(gm.Waypoints, cfg.Waves),
		lives:    20,
	}
}

func (s *StageScene) Update() error {
	s.frame++

	switch s.state {
	case statePlaying:
		s.updatePlaying()
	case stateVictory, stateDefeat:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s.switcher.SwitchScene(NewTitleScene(s.switcher))
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}
	return nil
}

const dt = 1.0 / float64(game.TargetTPS)

func (s *StageScene) updatePlaying() {
	// Spawn enemies
	s.spawner.Update(s.enemies, dt)

	// Move enemies along path
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, dt) {
			s.lives--
			s.enemies.Kill(e)
		}
	})

	// Check win/lose
	if s.lives <= 0 {
		s.state = stateDefeat
	}
	if s.spawner.IsClear(s.enemies) {
		s.state = stateVictory
	}
}

func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 40, B: 30, A: 255})

	// Map
	render.DrawMap(screen, s.gameMap)

	// Enemies
	render.DrawEnemies(screen, s.enemies)

	// HUD
	info := fmt.Sprintf("Map: %s  Wave: %d/%d  Lives: %d  Enemies: %d",
		s.gameMap.Config.Name, s.spawner.Wave, s.spawner.MaxWaves,
		s.lives, s.enemies.Count)
	ebitenutil.DebugPrintAt(screen, info, 8, 4)

	// Overlays
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	switch s.state {
	case stateVictory:
		ebitenutil.DebugPrintAt(screen, "VICTORY! Press ENTER", cx-60, cy)
	case stateDefeat:
		ebitenutil.DebugPrintAt(screen, "DEFEAT! Press ENTER", cx-56, cy)
	}
}
