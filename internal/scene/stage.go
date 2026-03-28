package scene

import (
	"fmt"
	"image/color"
	"log"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
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
	switcher    Switcher
	frame       int
	state       stageState
	gameMap     *gamemap.GameMap
	enemies     *enemy.Pool
	spawner     *enemy.Spawner
	towers      *tower.Pool
	projectiles *projectile.Pool
	lives       int
	gold        int
	kills       int
	towerDef    tower.TowerDef
}

// NewStageScene creates a new gameplay scene with map_01 loaded.
func NewStageScene(sw Switcher) *StageScene {
	cfg, err := config.LoadMap("map_01")
	if err != nil {
		log.Printf("failed to load map: %v", err)
		cfg = &config.MapConfig{
			ID: "fallback", Cols: 20, Rows: 9, CellSize: 60,
			Grid: make([][]int, 9), Waves: 3,
		}
		for i := range cfg.Grid {
			cfg.Grid[i] = make([]int, 20)
		}
	}
	gm := gamemap.NewGameMap(cfg)
	return &StageScene{
		switcher:    sw,
		gameMap:     gm,
		enemies:     enemy.DefaultPool(),
		spawner:     enemy.NewSpawner(gm.Waypoints, cfg.Waves),
		towers:      tower.DefaultPool(),
		projectiles: projectile.DefaultPool(),
		lives:       20,
		gold:        200,
		towerDef:    tower.DefaultTowerDef(),
	}
}

func (s *StageScene) Update() error {
	s.frame++

	switch s.state {
	case statePlaying:
		s.handleInput()
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

func (s *StageScene) handleInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		s.tryPlaceTower(float64(mx), float64(my))
	}
	for _, id := range inpututil.JustPressedTouchIDs() {
		tx, ty := ebiten.TouchPosition(id)
		s.tryPlaceTower(float64(tx), float64(ty))
	}
}

func (s *StageScene) tryPlaceTower(px, py float64) {
	gm := s.gameMap
	cellType := gm.CellAt(px, py)
	if cellType != config.CellBuildable {
		return
	}
	cs := float64(gm.CellSize)
	col := int((px - gm.OffsetX) / cs)
	row := int((py - gm.OffsetY) / cs)

	// Check if already occupied
	if s.towers.At(row, col) != nil {
		return
	}
	// Check gold
	if s.gold < s.towerDef.Cost {
		return
	}

	center := gm.CellCenter(row, col)
	s.towers.Place(row, col, center.X, center.Y, s.towerDef)
	s.gold -= s.towerDef.Cost
}

const dt = 1.0 / float64(game.TargetTPS)

func (s *StageScene) updatePlaying() {
	// 1. Spawn enemies
	s.spawner.Update(s.enemies, dt)

	// 2. Move enemies
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, dt) {
			s.lives--
			s.enemies.Kill(e)
		}
	})

	// 3. Tower combat (targeting + firing)
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, dt)

	// 4. Move projectiles
	s.projectiles.Update(dt)

	// 5. Projectile hits
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies)
	s.kills += kills
	s.gold += kills * 15 // gold per kill

	// 6. Check win/lose
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

	// Towers
	render.DrawTowers(screen, s.towers)

	// Enemies
	render.DrawEnemies(screen, s.enemies)

	// Projectiles
	render.DrawProjectiles(screen, s.projectiles)

	// Mouse hover preview
	if s.state == statePlaying {
		mx, my := ebiten.CursorPosition()
		cellType := s.gameMap.CellAt(float64(mx), float64(my))
		if cellType == config.CellBuildable {
			cs := float64(s.gameMap.CellSize)
			col := int((float64(mx) - s.gameMap.OffsetX) / cs)
			row := int((float64(my) - s.gameMap.OffsetY) / cs)
			center := s.gameMap.CellCenter(row, col)
			valid := s.towers.At(row, col) == nil && s.gold >= s.towerDef.Cost
			render.DrawTowerRangePreview(screen, float32(center.X), float32(center.Y), s.towerDef.Range, valid)
		}
	}

	// HUD
	info := fmt.Sprintf("Wave: %d/%d  Lives: %d  Gold: %d  Towers: %d  Kills: %d  [Click buildable cell to place tower]",
		s.spawner.Wave, s.spawner.MaxWaves, s.lives, s.gold, s.towers.Count, s.kills)
	ebitenutil.DebugPrintAt(screen, info, 8, 4)

	// Overlays
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	switch s.state {
	case stateVictory:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("VICTORY! Kills: %d  Press ENTER", s.kills), cx-80, cy)
	case stateDefeat:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DEFEAT! Kills: %d  Press ENTER", s.kills), cx-76, cy)
	}
}
