package scene

import (
	"fmt"
	"image/color"
	"log"

	_ "defense2/internal/core/tower/abilities" // register abilities via init()

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
	towerDefs   []tower.TowerDef
	selectedDef int // index into towerDefs
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
		towerDefs:   tower.BaseTowerDefs(),
		selectedDef: 0,
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
	// Tower selection: keys 1-4
	for i := 0; i < len(s.towerDefs) && i < 4; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			s.selectedDef = i
		}
	}

	// Tower placement
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		s.tryPlaceTower(float64(mx), float64(my))
	}
	for _, id := range inpututil.JustPressedTouchIDs() {
		tx, ty := ebiten.TouchPosition(id)
		s.tryPlaceTower(float64(tx), float64(ty))
	}

	// Sell tower: right click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		s.trySellTower(float64(mx), float64(my))
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

	if s.towers.At(row, col) != nil {
		return
	}
	def := s.towerDefs[s.selectedDef]
	if s.gold < def.Cost {
		return
	}

	center := gm.CellCenter(row, col)
	s.towers.Place(row, col, center.X, center.Y, def)
	s.gold -= def.Cost
}

func (s *StageScene) trySellTower(px, py float64) {
	gm := s.gameMap
	cs := float64(gm.CellSize)
	fx := (px - gm.OffsetX) / cs
	fy := (py - gm.OffsetY) / cs
	if fx < 0 || fy < 0 {
		return
	}
	col := int(fx)
	row := int(fy)
	t := s.towers.At(row, col)
	if t == nil {
		return
	}
	s.gold += t.Cost / 2
	s.towers.Remove(t)
}

const dt = 1.0 / float64(game.TargetTPS)

func (s *StageScene) updatePlaying() {
	// 1. Spawn enemies
	s.spawner.Update(s.enemies, dt)

	// 2. Enemy status effects (slow, bleed)
	pipeline.TickEnemyStatusEffects(s.enemies, dt)

	// 3. Move enemies
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, dt) {
			s.lives--
			s.enemies.Kill(e)
		}
	})

	// 4. Tower combat (targeting + firing)
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, dt)

	// 5. Move projectiles
	s.projectiles.Update(dt)

	// 6. Projectile hits (with abilities)
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers)
	s.kills += kills
	s.gold += kills * 15

	// 7. Check win/lose
	if s.lives <= 0 {
		s.state = stateDefeat
	}
	if s.spawner.IsClear(s.enemies) {
		s.state = stateVictory
	}
}

func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 40, B: 30, A: 255})

	render.DrawMap(screen, s.gameMap)
	render.DrawTowers(screen, s.towers)
	render.DrawEnemies(screen, s.enemies)
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
			def := s.towerDefs[s.selectedDef]
			valid := s.towers.At(row, col) == nil && s.gold >= def.Cost
			render.DrawTowerRangePreview(screen, float32(center.X), float32(center.Y), def.Range, valid)
		}
	}

	// HUD top
	def := s.towerDefs[s.selectedDef]
	info := fmt.Sprintf("Wave %d/%d  Lives %d  Gold %d  Kills %d",
		s.spawner.Wave, s.spawner.MaxWaves, s.lives, s.gold, s.kills)
	ebitenutil.DebugPrintAt(screen, info, 8, 4)

	// Tower selection bar
	selInfo := fmt.Sprintf("[1-%d] Select tower | Selected: %s ($%d) | Right-click to sell",
		len(s.towerDefs), def.Label, def.Cost)
	ebitenutil.DebugPrintAt(screen, selInfo, 8, 18)

	// Tower list
	for i, d := range s.towerDefs {
		marker := " "
		if i == s.selectedDef {
			marker = ">"
		}
		txt := fmt.Sprintf("%s%d:%s($%d)", marker, i+1, d.Label, d.Cost)
		ebitenutil.DebugPrintAt(screen, txt, 8+i*120, game.ScreenHeight-16)
	}

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
