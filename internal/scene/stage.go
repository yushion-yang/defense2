// stage.go — 游戏主场景。
// 管理塔防核心循环：生成敌人、移动、塔战斗、弹射物、经济、胜负判定。
package scene

import (
	"fmt"
	"image/color"
	"log"

	_ "defense2/internal/core/tower/abilities" // 通过 init() 注册能力

	"defense2/internal/config"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
	"defense2/internal/render"
	"defense2/internal/render/hud"

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

// StageScene 游戏主场景。
type StageScene struct {
	switcher    Switcher
	frame       int
	state       stageState
	gameMap     *gamemap.GameMap
	enemies     *enemy.Pool
	spawner     *enemy.Spawner
	towers      *tower.Pool
	projectiles *projectile.Pool
	econ        economy.Config
	lives       int
	gold        int
	kills       int
	towerDefs   []tower.TowerDef
	selectedDef int      // 当前选中的塔类型索引
	hoveredTower *tower.Tower // 鼠标悬停的已放置塔
	lastWave    int      // 上一帧的波次号（用于检测波次完成）
	notification string  // 屏幕中央短暂通知
	notifyTimer float64  // 通知剩余显示时间
}

// NewStageScene 创建游戏主场景，加载 map_01。
func NewStageScene(sw Switcher) *StageScene {
	cfg, err := config.LoadMap("map_01")
	if err != nil {
		log.Printf("地图加载失败: %v", err)
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
		econ:        economy.DefaultConfig(),
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

	// 通知计时器
	if s.notifyTimer > 0 {
		s.notifyTimer -= dt
	}

	return nil
}

func (s *StageScene) handleInput() {
	// 键盘选塔：1-4
	for i := 0; i < len(s.towerDefs) && i < 4; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			s.selectedDef = i
		}
	}

	mx, my := ebiten.CursorPosition()
	fmx, fmy := float32(mx), float32(my)

	// 点击建塔菜单
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		idx := hud.BuildMenuHitTest(fmx, fmy, len(s.towerDefs))
		if idx >= 0 {
			s.selectedDef = idx
		} else {
			s.tryPlaceTower(float64(mx), float64(my))
		}
	}

	// 触摸放塔
	for _, id := range inpututil.JustPressedTouchIDs() {
		tx, ty := ebiten.TouchPosition(id)
		s.tryPlaceTower(float64(tx), float64(ty))
	}

	// 右键卖塔
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		s.trySellTower(float64(mx), float64(my))
	}

	// 更新悬停塔
	s.updateHoveredTower(float64(mx), float64(my))
}

func (s *StageScene) updateHoveredTower(px, py float64) {
	gm := s.gameMap
	cs := float64(gm.CellSize)
	fx := (px - gm.OffsetX) / cs
	fy := (py - gm.OffsetY) / cs
	if fx < 0 || fy < 0 {
		s.hoveredTower = nil
		return
	}
	col := int(fx)
	row := int(fy)
	s.hoveredTower = s.towers.At(row, col)
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
	refund := s.econ.SellRefund(t.Cost)
	s.gold += refund
	s.towers.Remove(t)
	s.hoveredTower = nil
	s.showNotify(fmt.Sprintf("Sold +$%d", refund))
}

func (s *StageScene) showNotify(msg string) {
	s.notification = msg
	s.notifyTimer = 1.5
}

const dt = 1.0 / float64(game.TargetTPS)

func (s *StageScene) updatePlaying() {
	prevWave := s.spawner.Wave

	// 1. 生成敌人
	s.spawner.Update(s.enemies, dt)

	// 2. 敌人状态效果
	pipeline.TickEnemyStatusEffects(s.enemies, dt)

	// 3. 敌人移动
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, dt) {
			s.lives--
			s.enemies.Kill(e)
		}
	})

	// 4. 塔攻击
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, dt)

	// 5. 弹射物移动
	s.projectiles.Update(dt)

	// 6. 弹射物命中（含能力触发）
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers)
	s.kills += kills
	s.gold += kills * s.econ.KillGold()

	// 7. 波次完成奖励
	if s.spawner.Wave > prevWave && prevWave > 0 {
		bonus := s.econ.WaveCompleteGold(prevWave)
		interest := s.econ.InterestGold(s.gold)
		s.gold += bonus + interest
		s.showNotify(fmt.Sprintf("Wave %d clear! +$%d bonus +$%d interest", prevWave, bonus, interest))
	}

	// 8. 胜负判定
	if s.lives <= 0 {
		s.state = stateDefeat
	}
	if s.spawner.IsClear(s.enemies) {
		s.state = stateVictory
	}
}

func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 40, B: 30, A: 255})

	// 地图
	render.DrawMap(screen, s.gameMap)

	// 塔
	render.DrawTowers(screen, s.towers)

	// 敌人
	render.DrawEnemies(screen, s.enemies)

	// 弹射物
	render.DrawProjectiles(screen, s.projectiles)

	// 放塔预览
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

	// HUD：顶部栏
	hud.DrawTopBar(screen, hud.TopBarData{
		Gold:     s.gold,
		Lives:    s.lives,
		Wave:     s.spawner.Wave,
		MaxWaves: s.spawner.MaxWaves,
		Kills:    s.kills,
		Enemies:  s.enemies.Count,
	})

	// HUD：建塔菜单
	hud.DrawBuildMenu(screen, hud.BuildMenuData{
		TowerDefs:   s.towerDefs,
		SelectedIdx: s.selectedDef,
		Gold:        s.gold,
	})

	// HUD：塔信息面板
	sellValue := 0
	if s.hoveredTower != nil {
		sellValue = s.econ.SellRefund(s.hoveredTower.Cost)
	}
	hud.DrawInfoPanel(screen, s.hoveredTower, sellValue)

	// 屏幕中央通知
	if s.notifyTimer > 0 {
		cx := game.ScreenWidth / 2
		ebitenutil.DebugPrintAt(screen, s.notification, cx-60, 50)
	}

	// 胜负覆盖层
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	switch s.state {
	case stateVictory:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("VICTORY! Kills: %d  Press ENTER", s.kills), cx-80, cy)
	case stateDefeat:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DEFEAT! Kills: %d  Press ENTER", s.kills), cx-76, cy)
	}
}
