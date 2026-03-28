// stage.go — 游戏主场景。
// 管理塔防核心循环：生成敌人、移动、塔战斗、弹射物、经济、胜负判定。
package scene

import (
	"fmt"
	"image/color"
	"log"

	_ "defense2/internal/core/tower/abilities"  // 通过 init() 注册塔能力
	_ "defense2/internal/core/warden/types"     // 通过 init() 注册战灵类型

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/hero"
	"defense2/internal/core/persistence"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
	"defense2/internal/core/tutorial"
	"defense2/internal/core/warden"
	"defense2/internal/loader"
	"defense2/internal/render"
	"defense2/internal/render/hud"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// stageState 游戏主场景的状态枚举。
type stageState int

const (
	statePlaying stageState = iota // 游戏进行中
	stateVictory                   // 玩家胜利
	stateDefeat                    // 玩家失败
)

// StageScene 游戏主场景，包含所有运行时游戏状态。
type StageScene struct {
	switcher     Switcher          // 场景切换器引用
	frame        int               // 当前帧计数
	state        stageState        // 当前游戏状态（进行中/胜利/失败）
	gameMap      *gamemap.GameMap   // 运行时地图
	enemies      *enemy.Pool       // 敌人对象池
	spawner      *enemy.Spawner    // 波次出怪管理器
	towers       *tower.Pool       // 塔对象池
	projectiles  *projectile.Pool  // 弹射物对象池
	econ         economy.Config    // 经济配置
	lives        int               // 剩余生命值
	gold         int               // 当前金币
	kills        int               // 累计击杀数
	towerDefs    []tower.TowerDef  // 可建造的塔类型列表
	selectedDef  int               // 当前选中的塔类型索引
	hoveredTower    *tower.Tower            // 鼠标悬停的已放置塔（用于信息面板）
	towerRenderer   *render.TowerRenderer  // 塔 SVG 渲染器
	enemyRenderer   *render.EnemyRenderer  // 敌人 SVG 渲染器
	audioMgr        *gameAudio.Manager     // 音效管理器
	heroUnit        *hero.Hero             // 英雄实体
	wardenUnit      *warden.Warden        // 战灵实体
	eventPool       *event.Pool           // 事件池
	appliedEvents   []event.Event         // 已应用的事件列表
	killRewardBonus int                   // 额外击杀金币（事件增益）
	buildDiscount   float64               // 建造折扣比例（事件增益）
	tutorial        *tutorial.Tutorial        // 新手教程
	progressMgr     *persistence.ProgressManager // 持久化进度管理器
	lastWave        int                          // 上一帧的波次号
	notification    string                       // 屏幕中央通知文本
	notifyTimer     float64                      // 通知剩余时间（秒）
}

// NewStageScene 创建游戏主场景，默认加载 map_01。
func NewStageScene(sw Switcher) *StageScene {
	return NewStageSceneWithMap(sw, "map_01")
}

// NewStageSceneWithMap 创建游戏主场景，加载指定地图。
func NewStageSceneWithMap(sw Switcher, mapID string) *StageScene {
	cfg, err := config.LoadMap(mapID)
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

	// 寻找英雄基地位置（CellHeroBase=6），未找到则放在地图中央
	heroX, heroY := float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2
	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			if cfg.Grid[row][col] == config.CellHeroBase {
				p := gm.CellCenter(row, col)
				heroX, heroY = p.X, p.Y
			}
		}
	}

	// 初始化持久化
	store, _ := persistence.DefaultStorage()
	pm := persistence.NewProgressManager(store)

	// 教程（已完成则不再显示）
	tut := tutorial.DefaultTutorial()
	if pm.Progress().TutorialDone {
		tut.Skip()
	}

	s := &StageScene{
		switcher:      sw,
		gameMap:        gm,
		enemies:        enemy.DefaultPool(),
		spawner:        enemy.NewSpawner(gm, cfg.Waves),
		towers:         tower.DefaultPool(),
		projectiles:    projectile.DefaultPool(),
		econ:           economy.DefaultConfig(),
		towerRenderer:  render.NewTowerRenderer(config.GetAssetFS()),
		enemyRenderer:  render.NewEnemyRenderer(config.GetAssetFS()),
		audioMgr:       initAudio(),
		heroUnit:       loader.LoadHero(heroX, heroY),
		wardenUnit:     warden.NewWarden(1, "使者", "envoy"),
		eventPool:      event.NewPool(loader.LoadAllyEvents()),
		tutorial:       tut,
		progressMgr:    pm,
		lives:          20,
		gold:           200,
		towerDefs:      loadTowerDefsOrFallback(),
		selectedDef:    0,
	}

	// 触发教程首步
	tut.OnEvent("gameStart")

	return s
}

// Update 每帧逻辑更新：根据游戏状态分发输入处理和游戏逻辑。
func (s *StageScene) Update() error {
	s.frame++

	switch s.state {
	case statePlaying:
		s.handleInput()
		s.updatePlaying()
	case stateVictory, stateDefeat:
		// 胜利/失败状态：按 Enter 或点击进入结算场景
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s.switcher.SwitchScene(NewResultScene(s.switcher, ResultData{
				MapID:    s.gameMap.Config.ID,
				MapName:  s.gameMap.Config.Name,
				Won:      s.state == stateVictory,
				Kills:    s.kills,
				Waves:    s.spawner.Wave,
				MaxWaves: s.spawner.MaxWaves,
				Gold:     s.gold,
				Towers:   s.towers.Count,
			}))
		}
	}

	// ESC 随时返回标题
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
	}

	// 通知计时器
	if s.notifyTimer > 0 {
		s.notifyTimer -= dt
	}

	return nil
}

// handleInput 处理游戏进行中的输入：键盘选塔、点击放塔、右键卖塔。
func (s *StageScene) handleInput() {
	// 键盘选塔：1-4 对应 4 种塔
	for i := 0; i < len(s.towerDefs) && i < 4; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			s.selectedDef = i
		}
	}

	mx, my := ebiten.CursorPosition()
	fmx, fmy := float32(mx), float32(my)

	// 左键：点击建塔菜单选塔 或 点击地图放塔
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		idx := hud.BuildMenuHitTest(fmx, fmy, len(s.towerDefs))
		if idx >= 0 {
			s.selectedDef = idx
		} else {
			s.tryPlaceTower(float64(mx), float64(my))
		}
	}

	// 触摸放塔（移动端）
	for _, id := range inpututil.JustPressedTouchIDs() {
		tx, ty := ebiten.TouchPosition(id)
		s.tryPlaceTower(float64(tx), float64(ty))
	}

	// 右键卖塔
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		s.trySellTower(float64(mx), float64(my))
	}

	// 更新鼠标悬停的塔（用于信息面板显示）
	s.updateHoveredTower(float64(mx), float64(my))
}

// updateHoveredTower 根据鼠标位置更新悬停塔引用。
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

// tryPlaceTower 尝试在像素位置放置当前选中类型的塔。
func (s *StageScene) tryPlaceTower(px, py float64) {
	gm := s.gameMap
	cellType := gm.CellAt(px, py)
	if cellType != config.CellBuildable {
		return // 不是可建造位置
	}
	cs := float64(gm.CellSize)
	col := int((px - gm.OffsetX) / cs)
	row := int((py - gm.OffsetY) / cs)

	if s.towers.At(row, col) != nil {
		return // 该位置已有塔
	}
	def := s.towerDefs[s.selectedDef]
	if s.gold < def.Cost {
		return // 金币不足
	}

	center := gm.CellCenter(row, col)
	s.towers.Place(row, col, center.X, center.Y, def)
	s.gold -= def.Cost
	s.tutorial.OnEvent("towerBuilt")
}

// trySellTower 尝试出售像素位置上的塔。
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

// showNotify 显示屏幕中央通知，1.5 秒后自动消失。
func (s *StageScene) showNotify(msg string) {
	s.notification = msg
	s.notifyTimer = 1.5
}

// dt 固定时间步长（1/60 秒）。
const dt = 1.0 / float64(game.TargetTPS)

// updatePlaying 游戏进行中的核心循环，按管线顺序执行。
func (s *StageScene) updatePlaying() {
	prevWave := s.spawner.Wave

	// 1. 生成敌人
	s.spawner.Update(s.enemies, dt)
	if s.spawner.Wave > prevWave {
		s.tutorial.OnEvent("waveStarted")
	}

	// 2. 敌人状态效果（减速、流血等）
	pipeline.TickEnemyStatusEffects(s.enemies, dt)

	// 3. 敌人移动（到达终点扣生命）
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, dt) {
			s.lives--
			s.enemies.Kill(e)
		}
	})

	// 4. 英雄 AI + 射击
	s.heroUnit.Update(s.enemies, s.projectiles, dt)

	// 5. 战灵行为
	s.wardenUnit.Tick(&warden.TickContext{
		Enemies: s.enemies,
		Towers:  s.towers,
		DT:      dt,
	})

	// 6. 塔索敌射击
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, dt)

	// 7. 弹射物移动
	s.projectiles.Update(dt)

	// 8. 弹射物命中检测（含能力触发）
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers)
	s.kills += kills
	killGold := s.econ.KillGold() + s.killRewardBonus
	s.gold += kills * killGold
	if kills > 0 {
		s.tutorial.OnEvent("enemyKilled")
	}

	// 英雄/战灵击杀奖励
	if kills > 0 {
		s.heroUnit.AwardXP(kills * 3)
		for i := 0; i < kills; i++ {
			s.wardenUnit.OnKill()
		}
	}

	// 9. 波次完成奖励 + 事件触发
	if s.spawner.Wave > prevWave && prevWave > 0 {
		bonus := s.econ.WaveCompleteGold(prevWave)
		interest := s.econ.InterestGold(s.gold)
		s.gold += bonus + interest
		s.wardenUnit.OnWaveClear()
		s.heroUnit.AwardXP(10)
		s.showNotify(fmt.Sprintf("Wave %d clear! +$%d bonus +$%d interest", prevWave, bonus, interest))
		s.tutorial.OnEvent("waveCleared")

		// 检查是否是事件奖励波次
		for _, rw := range event.RewardWaves() {
			if prevWave == rw {
				s.triggerEventChoice(prevWave)
				break
			}
		}
	}

	// 10. 胜负判定 + 持久化结果
	if s.lives <= 0 && s.state == statePlaying {
		s.state = stateDefeat
		s.progressMgr.RecordGameResult(s.gameMap.Config.ID, s.kills, false)
	}
	if s.spawner.IsClear(s.enemies) && s.state == statePlaying {
		s.state = stateVictory
		s.progressMgr.RecordGameResult(s.gameMap.Config.ID, s.kills, true)
		// 教程完成后持久化标记
		if s.tutorial.IsComplete() {
			s.progressMgr.SetTutorialDone()
		}
	}
}

// triggerEventChoice 在奖励波次触发事件选择（当前自动选第一个，后续改为 UI 选择）。
func (s *StageScene) triggerEventChoice(wave int) {
	picks := s.eventPool.PickTiered(wave)
	if len(picks) == 0 {
		return
	}
	// TODO: 显示事件选择 UI，当前自动应用第一个
	chosen := picks[0]
	event.Apply(&chosen, s)
	s.appliedEvents = append(s.appliedEvents, chosen)
	s.showNotify(fmt.Sprintf("Event: %s", chosen.Label))
}

// ─── GameState 接口实现（供事件处理器调用）───

func (s *StageScene) AddGold(amount int)             { s.gold += amount }
func (s *StageScene) SetBuildDiscount(ratio float64)  { s.buildDiscount = ratio }
func (s *StageScene) SetKillRewardBonus(extra int)    { s.killRewardBonus = extra }

func (s *StageScene) BuffAllTowersDamage(ratio float64) {
	s.towers.Each(func(t *tower.Tower) { t.Damage *= (1 + ratio) })
}
func (s *StageScene) BuffAllTowersRange(ratio float64) {
	s.towers.Each(func(t *tower.Tower) { t.Range *= (1 + ratio) })
}
func (s *StageScene) BuffAllTowersSpeed(ratio float64) {
	s.towers.Each(func(t *tower.Tower) { t.AttackSpeed *= (1 + ratio) })
}
func (s *StageScene) SlowAllEnemies(ratio float64) {
	s.enemies.Each(func(e *enemy.Enemy) {
		e.BaseSpeed *= (1 - ratio)
		e.Speed = e.BaseSpeed
	})
}
func (s *StageScene) BuffFactionTowers(faction string, ratio float64) {
	s.towers.Each(func(t *tower.Tower) {
		if t.Faction == faction {
			t.Damage *= (1 + ratio)
		}
	})
}

// Draw 渲染游戏画面：地图 → 塔 → 敌人 → 弹射物 → 预览 → HUD → 通知 → 胜负覆盖。
func (s *StageScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 40, B: 30, A: 255})

	// 地图
	render.DrawMap(screen, s.gameMap)

	// 塔（优先 SVG 渲染，回退到彩色方块）
	s.towerRenderer.DrawTowers(screen, s.towers)

	// 敌人（优先 SVG 渲染）
	s.enemyRenderer.DrawEnemies(screen, s.enemies)

	// 弹射物
	render.DrawProjectiles(screen, s.projectiles)

	// 英雄
	render.DrawHero(screen, s.heroUnit)

	// 战灵
	render.DrawWarden(screen, s.wardenUnit)

	// 放塔预览（鼠标在可建造位置时显示）
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

	// HUD：顶部状态栏
	hud.DrawTopBar(screen, hud.TopBarData{
		Gold:     s.gold,
		Lives:    s.lives,
		Wave:     s.spawner.Wave,
		MaxWaves: s.spawner.MaxWaves,
		Kills:    s.kills,
		Enemies:  s.enemies.Count,
	})

	// HUD：底部建塔菜单
	hud.DrawBuildMenu(screen, hud.BuildMenuData{
		TowerDefs:   s.towerDefs,
		SelectedIdx: s.selectedDef,
		Gold:        s.gold,
	})

	// HUD：左下塔信息面板
	sellValue := 0
	if s.hoveredTower != nil {
		sellValue = s.econ.SellRefund(s.hoveredTower.Cost)
	}
	hud.DrawInfoPanel(screen, s.hoveredTower, sellValue)

	// 教程提示（顶部偏下位置）
	if msg := s.tutorial.CurrentMessage(); msg != "" {
		ebitenutil.DebugPrintAt(screen, msg, game.ScreenWidth/2-100, 36)
	}

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

// loadTowerDefsOrFallback 从 JSON 配置加载塔定义，失败时回退到硬编码定义。
func loadTowerDefsOrFallback() []tower.TowerDef {
	defs, err := loader.LoadTowerDefs()
	if err != nil {
		log.Printf("塔配置加载失败，使用硬编码定义: %v", err)
		return tower.BaseTowerDefs()
	}
	if len(defs) == 0 {
		return tower.BaseTowerDefs()
	}
	return defs
}

// initAudio 创建音效管理器并从嵌入式文件系统预加载所有 WAV。
func initAudio() *gameAudio.Manager {
	mgr := gameAudio.NewManager()
	assetFS := config.GetAssetFS()
	if assetFS != nil {
		mgr.LoadAllFromFS(assetFS)
	}
	return mgr
}
