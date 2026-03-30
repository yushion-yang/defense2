// stage.go — 游戏主场景。
// 管理塔防核心循环：生成敌人、移动、塔战斗、弹射物、经济、胜负判定。
package scene

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"strings"

	_ "defense2/internal/core/skill"           // 通过 init() 注册技能
	_ "defense2/internal/core/tower/abilities" // 通过 init() 注册塔能力
	_ "defense2/internal/core/warden/types"    // 通过 init() 注册战灵类型

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/debug"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/gamemode"
	"defense2/internal/core/persistence"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/skill"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/tutorial"
	"defense2/internal/core/warden"
	"defense2/internal/input"
	"defense2/internal/loader"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/particle"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// stageState 游戏主场景的胜负状态。
type stageState int

const (
	statePlaying stageState = iota // 游戏进行中
	stateVictory                   // 玩家胜利
	stateDefeat                    // 玩家失败
)

// interactMode 交互状态机。
// 控制玩家当前的操作模式和 UI 显示。
//
// 状态流转图：
//
//	                ┌─────────────────────────┐
//	                │        modeIdle         │ ◄── 默认状态
//	                └──┬────┬────┬────┬───────┘
//	   点击"造塔"/B    │    │    │    │ 点击已有塔
//	                ▼    │    │    │    ▼
//	         ┌──────────┐│    │    │ ┌──────────────┐
//	         │modeBuild-││    │    │ │modeTowerSel  │
//	         │  Menu    ││    │    │ │（显示面板+射程）│
//	         └──┬───────┘│    │    │ └──┬───────────┘
//	选择塔型    │        │    │    │    │ 点空地/ESC
//	         ▼        │    │    │    ▼
//	   ┌───────────┐  │    │    │  → modeIdle
//	   │modeBuild- │  │    │    │
//	   │  Place    │  │    │    │
//	   │（放塔模式） │  │    │    │
//	   └───────────┘  │    │    │
//	放塔后保持/ESC退出│    │    │
//	                  │    │    │
//	        事件触发  ▼    │    │
//	         ┌──────────┐ │    │
//	         │modeEvent │ │    │
//	         │（选择事件）│ │    │
//	         └──────────┘ │    │
//	                      │    │
//	           菜单按钮   ▼    │
//	            ┌───────────┐  │
//	            │modePaused │  │
//	            └───────────┘  │
type interactMode int

const (
	modeIdle         interactMode = iota // 空闲：观察游戏
	modeBuildMenu                        // 建塔面板打开：选择塔类型
	modeBuildPlace                       // 放塔模式：已选塔型，点击可建位放塔
	modeTowerSel                         // 塔选中：显示信息面板+射程
	modeSpawnMenu                        // 造怪菜单：选择敌人类型
	modeSpawnPlace                       // 造怪放置：点击地图放置敌人
	modeEvent                            // 事件选择：弹窗选事件
	modePaused                           // 暂停菜单
	modeWardenSelect                     // 战灵选择覆盖层
)

// StageOptions 创建 StageScene 的配置选项。
type StageOptions struct {
	MapID        string
	WardenType   string
	ModeID       string // 游戏模式 ID（默认 "campaign"）
	DifficultyID string // 难度 ID（默认 "normal"）
	Gold         int    // 0 = 默认（由难度决定）
	Lives        int    // 0 = 默认 20
	Waves        int    // 0 = 地图默认; -1 = 无波次
	TestMode     bool   // 测试模式（启用调试面板）
	ScenarioID   string // 测试场景 ID
	EnemyFilter  string // ground-only/flying-only/elite-only/boss-only/all-static/mixed/stress/dummy/none
	ManualWave   bool   // 仅手动开波
}

// StageScene 游戏主场景，包含所有运行时游戏状态。
type StageScene struct {
	switcher          Switcher                     // 场景切换器引用
	session           *gamemode.Session            // 游戏模式会话
	modeID            string                       // 模式 ID（用于重玩）
	diffID            string                       // 难度 ID（用于重玩）
	frame             int                          // 当前帧计数
	state             stageState                   // 当前游戏状态（进行中/胜利/失败）
	gameMap           *gamemap.GameMap             // 运行时地图
	enemies           *enemy.Pool                  // 敌人对象池
	spawner           *enemy.Spawner               // 波次出怪管理器
	towers            *tower.Pool                  // 塔对象池
	projectiles       *projectile.Pool             // 弹射物对象池
	beams             *combat.BeamPool             // 光束视觉对象池
	econ              economy.Config               // 经济配置
	lives             int                          // 剩余生命值
	gold              int                          // 当前金币
	kills             int                          // 累计击杀数
	towerDefs         []tower.TowerDef             // 可建造的塔类型列表
	selectedDef       int                          // 当前选中的塔类型索引
	selectedTower     *tower.Tower                 // 点击选中的塔（显示信息面板+射程）
	towerRenderer     *render.TowerRenderer        // 塔 SVG 渲染器
	enemyRenderer     *render.EnemyRenderer        // 敌人 SVG 渲染器
	wardenRenderer    *render.WardenRenderer       // 战灵精灵渲染器
	audioMgr          *gameAudio.Manager           // 音效管理器
	wardenUnit        *warden.Warden               // 战灵实体（选择前为 nil）
	wardenOverlay     *hud.WardenSelectOverlay     // 战灵选择覆盖层
	wardenReady       bool                         // 战灵已选择并激活
	eventPool         *event.Pool                  // 事件池
	appliedEvents     []event.Event                // 已应用的事件列表
	killRewardBonus   int                          // 额外击杀金币（事件增益）
	buildDiscount     float64                      // 建造折扣比例（事件增益）
	tutorial          *tutorial.Tutorial           // 新手教程
	progressMgr       *persistence.ProgressManager // 持久化进度管理器
	lastWave          int                          // 上一帧的波次号
	wardenType        string                       // 战灵类型标识（用于重玩传递）
	wardenCfg         *config.WardenConfig         // 战灵配置（用于面板显示）
	gameSpeed         int                          // 游戏速度倍率（1 或 2）
	imode             interactMode                 // 交互状态机
	prePauseMode      interactMode                 // 暂停前的交互模式（恢复用）
	eventPending      []event.Event                // 待选事件列表（modeEvent 时使用）
	eventHoverIdx     int                          // 事件卡片鼠标悬停索引
	buildHoverIdx     int                          // 建塔面板鼠标悬停索引
	gesture           *input.Gesture               // 统一手势识别器
	waveLivesSnapshot int                          // 波开始时的生命快照（用于完美波次检测）
	// 相机（大地图拖拽）
	camX, camY    float64 // 相机偏移（世界坐标）
	dragging      bool    // 是否正在拖拽
	dragStartX    float64 // 拖拽起始屏幕位置
	dragStartY    float64
	dragCamStartX float64 // 拖拽起始相机位置
	dragCamStartY float64
	dragMoved     bool // 拖拽期间是否产生了位移（区分点击和拖拽）
	// 测试模式
	// HUD 面板状态
	wavePanelOpen   bool // 左下角波次面板是否展开
	wardenPanelOpen bool // 右下角战灵面板是否展开
	testMode        bool
	scenarioID      string
	enemyFilter     string
	manualWave      bool
	debugPanelOpen  bool
	debugShowRange  bool
	spawnMode       bool
	spawnType       string
	spawnHoverIdx   int
	initOpts        StageOptions // 保存原始配置（重新开始用）
	postPipeline    *postprocess.Pipeline      // 后处理管线（bloom 等）
	particlePool    *particle.Pool             // GPU 粒子系统
	debugOverlay    *hud.DebugOverlay          // 调试覆盖层（F2 切换）
	perfTracker     *debug.PerfTracker         // 性能追踪器
}

// NewStageScene 创建游戏主场景，默认加载 map_01。
func NewStageScene(sw Switcher) *StageScene {
	return NewStageSceneWithOpts(sw, StageOptions{MapID: "map_01", WardenType: "envoy"})
}

// NewStageSceneWithMap 创建游戏主场景，加载指定地图（默认金灵战灵）。
func NewStageSceneWithMap(sw Switcher, mapID string) *StageScene {
	return NewStageSceneWithOpts(sw, StageOptions{MapID: mapID, WardenType: "envoy"})
}

// NewStageSceneWithOptions 创建游戏主场景，指定地图和战灵类型（向后兼容）。
func NewStageSceneWithOptions(sw Switcher, mapID, wardenType string) *StageScene {
	return NewStageSceneWithOpts(sw, StageOptions{MapID: mapID, WardenType: wardenType})
}

// NewStageSceneWithOpts 创建游戏主场景，接受完整配置选项。
func NewStageSceneWithOpts(sw Switcher, opts StageOptions) *StageScene {
	cfg, err := config.LoadMap(opts.MapID)
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

	// 初始化持久化
	store, _ := persistence.DefaultStorage()
	pm := persistence.NewProgressManager(store)

	// 教程（已完成则不再显示）
	tut := tutorial.DefaultTutorial()
	if pm.Progress().TutorialDone {
		tut.Skip()
	}

	spawner := enemy.NewSpawner(gm, cfg.Waves)

	// 加载敌人原型配置并注入到 spawner
	if archetypes, err := config.LoadEnemyArchetypes(); err == nil {
		spawner.Archetypes = convertArchetypesToSpawnConfigs(archetypes)
	} else {
		log.Printf("敌人原型加载失败，使用默认配置: %v", err)
	}

	// 加载战灵配置（用于面板显示）
	var wardenCfg *config.WardenConfig
	if cfgs, err := config.LoadWardenConfigs(); err == nil {
		if wc, ok := cfgs[opts.WardenType]; ok {
			wardenCfg = &wc
		}
	}

	// 加载难度配置
	diff := gamemode.LoadDifficulty(opts.DifficultyID)

	// 应用难度到 spawner
	spawner.HPScale = diff.HPScale
	spawner.SpeedScale = diff.SpeedScale

	// 初始金币和经济配置
	startGold := diff.StartGold
	econ := economy.DefaultConfig()
	econ.KillReward = int(float64(econ.KillReward) * diff.RewardScale)
	econ.WaveBonus = int(float64(econ.WaveBonus) * diff.RewardScale)

	// 创建游戏模式和会话
	modeID := opts.ModeID
	if modeID == "" {
		modeID = "campaign"
	}
	if opts.TestMode {
		modeID = "test"
	}
	mode := gamemode.GetOrDefault(modeID)
	session := gamemode.NewSession(mode)

	// 战灵：WardenType 为空表示需要在 Stage 内选择
	var wardenUnit *warden.Warden
	wardenReady := false
	if opts.WardenType != "" {
		wardenUnit = warden.NewWarden(1, opts.WardenType, opts.WardenType)
		wardenReady = true
	}

	s := &StageScene{
		switcher:        sw,
		session:         session,
		modeID:          modeID,
		diffID:          opts.DifficultyID,
		gameMap:         gm,
		enemies:         enemy.DefaultPool(),
		spawner:         spawner,
		towers:          tower.DefaultPool(),
		projectiles:     projectile.DefaultPool(),
		beams:           combat.NewBeamPool(),
		econ:            econ,
		towerRenderer:   render.NewTowerRenderer(config.GetAssetFS()),
		enemyRenderer:   render.NewEnemyRenderer(config.GetAssetFS()),
		wardenRenderer:  render.NewWardenRenderer(config.GetAssetFS()),
		audioMgr:        sw.AudioManager(),
		wardenUnit:      wardenUnit,
		wardenOverlay:   hud.NewWardenSelectOverlay(),
		wardenReady:     wardenReady,
		eventPool:       event.NewPool(loader.LoadAllyEvents()),
		tutorial:        tut,
		progressMgr:     pm,
		lives:           20,
		gold:            startGold,
		towerDefs:       loadTowerDefsOrFallback(),
		selectedDef:     0,
		wardenType:      opts.WardenType,
		wardenCfg:       wardenCfg,
		gameSpeed:       1,
		gesture:         newStageGesture(),
		wavePanelOpen:   true,
		wardenPanelOpen: false,
	}

	// 后处理管线（bloom）+ 粒子系统
	s.postPipeline = postprocess.NewPipeline()
	s.particlePool = particle.NewPool()

	// 调试覆盖层 + 性能追踪器
	s.debugOverlay = hud.NewDebugOverlay()
	s.perfTracker = debug.NewPerfTracker()

	// 注入战灵精灵获取函数到覆盖层
	s.wardenOverlay.SpriteFunc = s.wardenRenderer.GetSprite

	// 测试模式覆盖（优先于难度设置）
	if opts.Gold > 0 {
		s.gold = opts.Gold
	}
	if opts.Lives > 0 {
		s.lives = opts.Lives
	}
	if opts.Waves > 0 {
		spawner.MaxWaves = opts.Waves
	} else if opts.Waves == -1 {
		spawner.MaxWaves = 0
		spawner.AllDone = true
	}
	s.testMode = opts.TestMode
	s.scenarioID = opts.ScenarioID
	s.enemyFilter = opts.EnemyFilter
	s.manualWave = opts.ManualWave
	s.initOpts = opts

	// 测试模式：手动开波 + 每波固定 3 个怪 + 高 HP（方便观察）
	if opts.TestMode {
		spawner.ManualWave = true
		spawner.FixedCount = 3
		spawner.HPScale = 50
	}
	// ManualWave 场景覆盖（如沙盒模式也需要手动开波）
	if opts.ManualWave {
		spawner.ManualWave = true
	}

	// 传递 EnemyFilter 到 spawner
	spawner.EnemyFilter = opts.EnemyFilter

	// 波间隔使用模式配置
	spawner.WaveInterval = session.Mode.IntermissionSecs()

	// 未选战灵时：ManualWave 确保第一波不自动开始（由 stage 控制）
	if !wardenReady {
		spawner.ManualWave = true
	}

	// 模式初始化（endless → 设置 maxWaves=9999, timed → 初始化计时器）
	session.Mode.OnInit(s.buildModeCtx())

	// all-static 模式：一次性生成所有原型静止展示
	if opts.EnemyFilter == "all-static" {
		s.spawnAllStatic()
	}

	// 触发教程首步
	tut.OnEvent("gameStart")

	return s
}

// buildModeCtx 构建游戏模式上下文快照。
func (s *StageScene) buildModeCtx() *gamemode.Context {
	return &gamemode.Context{
		Wave:         s.spawner.Wave,
		MaxWaves:     s.spawner.MaxWaves,
		Lives:        s.lives,
		Gold:         s.gold,
		Kills:        s.kills,
		Leaked:       s.session.Stats.Leaked,
		TowersBuilt:  s.session.Stats.TowersBuilt,
		EnemiesAlive: s.enemies.Count,
		ElapsedTime:  s.session.ElapsedTime,
		Spawning:     !s.spawner.IsClear(s.enemies),
		SetLives:     func(v int) { s.lives = v },
		SetGold:      func(v int) { s.gold = v },
		AddGold:      func(v int) { s.gold += v },
		SetMaxWaves:  func(v int) { s.spawner.MaxWaves = v },
	}
}

// spawnAllStatic 生成所有敌人原型，静止排列在地图上（用于全怪展示模式）。
func (s *StageScene) spawnAllStatic() {
	archetypes := s.spawner.Archetypes
	if archetypes == nil {
		return
	}
	// Collect all archetype names sorted
	var names []string
	for name := range archetypes {
		names = append(names, name)
	}
	sort.Strings(names)

	// Grid layout: 8 columns
	cols := 8
	cellSize := float64(s.gameMap.CellSize)
	offsetX := s.gameMap.OffsetX + cellSize
	offsetY := s.gameMap.OffsetY + cellSize

	for i, name := range names {
		col := i % cols
		row := i / cols
		x := offsetX + float64(col)*cellSize*1.5
		y := offsetY + float64(row)*cellSize*1.5

		cfg := archetypes[name]
		baseHP := 100.0 // 展示用固定基准
		baseSpd := 50.0
		e := s.enemies.Spawn(x, y, baseHP, baseSpd, 9999, name, cfg)
		if e != nil {
			e.Speed = 0
			e.BaseSpeed = 0
		}
	}
	// Stop spawner
	s.spawner.AllDone = true
}

// Update 每帧逻辑更新：根据游戏状态分发输入处理和游戏逻辑。
func (s *StageScene) Update() error {
	s.frame++

	switch s.state {
	case statePlaying:
		// 交互状态机驱动
		switch s.imode {
		case modeEvent:
			s.handleEventSelection()
		case modePaused:
			s.handlePausedInput()
		case modeWardenSelect:
			s.handleWardenSelection()
		default:
			s.handleInput()
			s.updatePlaying()
		}
	case stateVictory, stateDefeat:
		// 胜利/失败状态：点击/触摸进入结算场景
		if isTapJustPressed() {
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewResultScene(s.switcher, ResultData{
				MapID:        s.gameMap.Config.ID,
				MapName:      s.gameMap.Config.Name,
				Won:          s.state == stateVictory,
				Kills:        s.kills,
				Waves:        s.spawner.Wave,
				MaxWaves:     s.spawner.MaxWaves,
				Gold:         s.gold,
				Towers:       s.towers.Count,
				WardenType:   s.wardenType,
				ModeID:       s.modeID,
				DifficultyID: s.diffID,
				Score:        s.session.Mode.GetScore(s.buildModeCtx()),
				ElapsedSecs:  s.session.ElapsedTime,
			}))
		}
	}

	// Toast 通知更新
	hud.UpdateToast(dt)

	return nil
}

// handleInput 基于手势识别器 + 交互状态机处理输入。
// Gesture 在 Update 中统一判定 Tap/Drag/Scroll：
//   - Drag: 实时平移相机（按住移动中每帧更新）
//   - Tap:  松开时触发游戏操作（仅未拖拽时）
//   - Scroll: 触控板双指滚动平移相机
func (s *StageScene) handleInput() {
	g := s.gesture
	// 状态机控制拖拽权限
	switch s.imode {
	case modeIdle, modeBuildPlace, modeTowerSel, modeSpawnPlace:
		g.DragEnabled = s.needsCamera()
	default: // modeBuildMenu, modeSpawnMenu, modeEvent, modePaused
		g.DragEnabled = false
	}
	g.Update()

	mx, my := g.CursorPos()
	fmx, fmy := float32(mx), float32(my)

	// ── 拖拽 → 平移相机（实时，每帧） ──
	if g.IsDragging() && s.needsCamera() {
		dx, dy := g.DragDelta()
		s.camX -= dx
		s.camY -= dy
		s.clampCamera()
	}

	// ── 滚轮 ──
	_, sy := g.ScrollDelta()
	// 调试面板打开时，滚轮用于面板滚动
	if sy != 0 && s.debugPanelOpen {
		hud.DebugPanelScroll(sy)
	} else if sy != 0 && s.needsCamera() {
		s.camY -= sy * 3
		s.clampCamera()
	}

	// ── Hover 更新 ──
	if s.imode == modeSpawnMenu {
		s.spawnHoverIdx = hud.SpawnMenuHoverTest(fmx, fmy, len(s.spawnEntries()))
	} else {
		s.spawnHoverIdx = -1
	}

	// ── 键盘快捷键 ──
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch s.imode {
		case modeBuildMenu, modeBuildPlace:
			s.selectedTower = nil
			s.imode = modeIdle
		case modeTowerSel:
			s.selectedTower = nil
			s.imode = modeIdle
		case modeSpawnMenu, modeSpawnPlace:
			s.spawnMode = false
			s.spawnType = ""
			s.imode = modeIdle
		default:
			// modeIdle 等: ESC 打开暂停菜单（而非直接退出对局）
			s.prePauseMode = s.imode
			s.imode = modePaused
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if s.imode == modeBuildMenu {
			s.imode = modeIdle
		} else {
			s.imode = modeBuildMenu
			s.selectedTower = nil
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) && s.imode == modeTowerSel {
		s.tryUpgradeTower()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if !s.spawner.WaveActive && !s.spawner.AllDone {
			s.tryStartWave()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.gameSpeed = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.gameSpeed = 2
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.gameSpeed = 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if s.imode == modePaused {
			s.imode = s.prePauseMode
		} else {
			s.prePauseMode = s.imode
			s.imode = modePaused
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if s.imode == modeTowerSel && s.selectedTower != nil {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
		}
	}

	// F2: 调试覆盖层（性能统计，任何模式可用）
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		s.debugOverlay.Toggle()
	}

	// 测试模式专用快捷键
	if s.testMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyD) {
			s.debugPanelOpen = !s.debugPanelOpen
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyG) {
			s.gold += 500
			hud.ShowToast("+500 金币")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyK) {
			s.enemies.Each(func(e *enemy.Enemy) {
				e.HP = 0
			})
			hud.ShowToast("清除全场敌人")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			s.enemies.Each(func(e *enemy.Enemy) {
				e.HP = 0
			})
			s.spawner.WaveActive = false
			s.spawner.StartNextWave()
			hud.ShowToast("跳到下一波")
		}
	}

	// ── Hover 更新（每帧） ──
	if s.imode == modeBuildMenu {
		s.buildHoverIdx = hud.BuildMenuHoverTest(fmx, fmy, len(s.towerDefs))
	} else {
		s.buildHoverIdx = -1
	}

	// ── Tap → 游戏操作（仅在 Gesture 判定为 Tap 时执行） ──
	if !g.JustTapped() {
		return
	}
	tapX, tapY := g.TapPos()
	ftx, fty := float32(tapX), float32(tapY)
	wtx, wty := s.screenToWorld(tapX, tapY)

	// 调试面板点击（优先级最高）
	if s.testMode && s.debugPanelOpen {
		actions := s.debugActions()
		idx := hud.DebugPanelHitTest(ftx, fty, actions)
		if idx == -2 {
			s.debugPanelOpen = false // 关闭按钮
			return
		}
		if idx >= 0 && idx < len(actions) && actions[idx].Action != nil {
			actions[idx].Action()
			return // consume click
		}
	}

	// 左下角切换按钮（波次面板）
	if hud.ToggleButtonHitTest(ftx, fty, true) {
		s.wavePanelOpen = !s.wavePanelOpen
		return
	}
	// 右下角切换按钮（战灵面板，与建塔菜单互斥）
	if hud.ToggleButtonHitTest(ftx, fty, false) {
		s.wardenPanelOpen = !s.wardenPanelOpen
		if s.wardenPanelOpen {
			// 关闭建塔菜单
			if s.imode == modeBuildMenu || s.imode == modeBuildPlace {
				s.imode = modeIdle
			}
			s.selectedTower = nil
		}
		return
	}

	// TopBar 按钮（屏幕坐标）
	topBtn := hud.TopBarHitTest(ftx, fty)
	switch topBtn {
	case "start":
		if !s.spawner.WaveActive && !s.spawner.AllDone {
			s.tryStartWave()
		}
		return
	case "speed":
		if s.testMode {
			// 测试模式: 1 → 2 → 3 → 10(turbo) → 1
			switch s.gameSpeed {
			case 1:
				s.gameSpeed = 2
			case 2:
				s.gameSpeed = 3
			case 3:
				s.gameSpeed = 10
			default:
				s.gameSpeed = 1
			}
		} else {
			// 普通模式: 1 → 2 → 1
			if s.gameSpeed == 1 {
				s.gameSpeed = 2
			} else {
				s.gameSpeed = 1
			}
		}
		return
	case "menu":
		s.imode = modePaused
		return
	case "build":
		if s.imode == modeBuildMenu {
			s.imode = modeIdle
		} else {
			s.imode = modeBuildMenu
			s.selectedTower = nil
			s.wardenPanelOpen = false // 与战灵面板互斥
		}
		return
	case "spawn":
		if s.imode == modeSpawnMenu || s.imode == modeSpawnPlace {
			s.imode = modeIdle
			s.spawnMode = false
			s.spawnType = ""
		} else {
			s.imode = modeSpawnMenu
			s.spawnMode = true
			s.spawnType = ""
			s.selectedTower = nil
		}
		return
	case "debug":
		s.debugPanelOpen = !s.debugPanelOpen
		return
	}

	// 按交互模式分发 Tap
	switch s.imode {
	case modeIdle:
		clicked := s.towerAtPixel(wtx, wty)
		if clicked != nil {
			s.selectedTower = clicked
			s.imode = modeTowerSel
		} else if s.wardenPanelOpen {
			s.wardenPanelOpen = false // 点击空地收起战灵面板
		}

	case modeBuildMenu:
		idx := hud.BuildMenuHitTest(ftx, fty, len(s.towerDefs))
		if idx == -2 || idx == -1 {
			s.imode = modeIdle // 点击关闭按钮或面板外部 → 关闭
		} else if idx >= 0 {
			s.selectedDef = idx
			s.selectedTower = nil
			s.imode = modeBuildPlace
		}

	case modeBuildPlace:
		placed := s.tryPlaceTower(wtx, wty)
		if placed {
			s.imode = modeIdle // 放完一个回到空闲，需重新选择
		} else if s.towerAtPixel(wtx, wty) != nil {
			hud.ShowToast("此位置已有塔")
		}

	case modeSpawnMenu:
		// 造怪菜单：点击选择敌人类型
		entries := s.spawnEntries()
		if idx := hud.SpawnMenuHitTest(ftx, fty, len(entries)); idx >= 0 {
			s.spawnType = entries[idx].Name
			s.imode = modeSpawnPlace
			label := entries[idx].Name
			if entries[idx].Config != nil && entries[idx].Config.Label != "" {
				label = entries[idx].Config.Label
			}
			hud.ShowToast("点击地图放置: " + label)
		} else {
			// 点击菜单外部 → 关闭
			s.imode = modeIdle
			s.spawnMode = false
			s.spawnType = ""
		}

	case modeSpawnPlace:
		// 造怪放置：点击地图放置静止敌人（baseHP=100, baseSpeed=0 → 静止）
		if cfg, ok := s.spawner.Archetypes[s.spawnType]; ok {
			s.enemies.Spawn(wtx, wty, 100, 0, 0, s.spawnType, cfg)
			label := s.spawnType
			if cfg.Label != "" {
				label = cfg.Label
			}
			hud.ShowToast("已放置: " + label)
		}
		// 放完后留在放置模式，可继续放置同类敌人

	case modeTowerSel:
		if hud.InfoPanelUpgradeHitTest(ftx, fty, s.selectedTower) {
			s.tryUpgradeTower()
		} else if hud.InfoPanelSellHitTest(ftx, fty, s.selectedTower) {
			s.trySellTower(s.selectedTower.X, s.selectedTower.Y)
			s.imode = modeIdle
		} else {
			clicked := s.towerAtPixel(wtx, wty)
			if clicked != nil && clicked != s.selectedTower {
				s.selectedTower = clicked
			} else {
				s.selectedTower = nil
				s.imode = modeIdle
			}
		}
	}
}

// handlePausedInput 暂停菜单输入：按钮点击 + 键盘。
func (s *StageScene) handlePausedInput() {
	s.gesture.Update()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.imode = s.prePauseMode // 恢复暂停前的模式
		return
	}
	if s.gesture.JustTapped() {
		tx, ty := s.gesture.TapPos()
		action := hud.PauseMenuHitTest(float32(tx), float32(ty))
		switch action {
		case hud.PauseResume:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.imode = s.prePauseMode // 恢复暂停前的模式
		case hud.PauseRestart:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, s.initOpts))
		case hud.PauseQuit:
			s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
		}
	}
}

// tryUpgradeTower 为选中的塔购买 10 点永久强度。
func (s *StageScene) tryUpgradeTower() {
	t := s.selectedTower
	if t == nil {
		return
	}
	cost := tower.StrengthBuyCost
	if s.gold < cost {
		return
	}
	spent := t.BuyStrength()
	s.gold -= spent
	// 通过战力系统增加永久强度
	if t.Strength != nil {
		t.Strength.AddPermanent(10)
	}
	s.audioMgr.PlaySafe(gameAudio.SFXUpgrade)
	s.showNotify(fmt.Sprintf("强度+10 (-$%d)", spent))
}

// towerAtPixel 返回像素位置上的塔，无塔返回 nil。
// screenToWorld 将屏幕坐标转换为世界坐标（加上相机偏移）。

// newStageGesture 创建配置好的手势识别器。
func newStageGesture() *input.Gesture {
	g := input.NewGesture()
	g.ToLogical = func(x, y float64) (float64, float64) {
		return x / draw.Scale, y / draw.Scale
	}
	g.IsOnUI = func(x, y float64) bool {
		fx, fy := float32(x), float32(y)
		if hud.TopBarHitTest(fx, fy) != "" {
			return true
		}
		// 左下/右下角切换按钮
		if hud.ToggleButtonHitTest(fx, fy, true) || hud.ToggleButtonHitTest(fx, fy, false) {
			return true
		}
		if y > float64(game.ScreenHeight)-120 {
			return true
		}
		return false
	}
	return g
}

// screenToWorld 将屏幕坐标转换为世界坐标（加上相机偏移）。
func (s *StageScene) screenToWorld(sx, sy float64) (float64, float64) {
	return sx + s.camX, sy + s.camY
}

// clampCamera 将相机偏移夹紧到地图范围内。
func (s *StageScene) clampCamera() {
	mapW := s.gameMap.Width()
	mapH := s.gameMap.Height()
	screenW := float64(game.ScreenWidth)
	screenH := float64(game.ScreenHeight)

	// 地图小于等于屏幕时不允许滚动
	maxX := mapW + s.gameMap.OffsetX*2 - screenW
	maxY := mapH + s.gameMap.OffsetY*2 - screenH
	if maxX < 0 {
		maxX = 0
	}
	if maxY < 0 {
		maxY = 0
	}

	if s.camX < 0 {
		s.camX = 0
	}
	if s.camX > maxX {
		s.camX = maxX
	}
	if s.camY < 0 {
		s.camY = 0
	}
	if s.camY > maxY {
		s.camY = maxY
	}
}

// needsCamera 返回地图是否需要相机（大于屏幕）。
func (s *StageScene) needsCamera() bool {
	mapW := s.gameMap.Width() + s.gameMap.OffsetX*2
	mapH := s.gameMap.Height() + s.gameMap.OffsetY*2
	return mapW > float64(game.ScreenWidth) || mapH > float64(game.ScreenHeight)
}

func (s *StageScene) towerAtPixel(px, py float64) *tower.Tower {
	gm := s.gameMap
	cs := float64(gm.CellSize)
	fx := (px - gm.OffsetX) / cs
	fy := (py - gm.OffsetY) / cs
	if fx < 0 || fy < 0 {
		return nil
	}
	return s.towers.At(int(fy), int(fx))
}

// tryPlaceTower 尝试在像素位置放置当前选中类型的塔。
func (s *StageScene) tryPlaceTower(px, py float64) bool {
	gm := s.gameMap
	cellType := gm.CellAt(px, py)
	if cellType != config.CellBuildable {
		return false
	}
	cs := float64(gm.CellSize)
	col := int((px - gm.OffsetX) / cs)
	row := int((py - gm.OffsetY) / cs)

	if s.towers.At(row, col) != nil {
		return false // 该位置已有塔
	}
	def := s.towerDefs[s.selectedDef]
	cost := def.Cost
	if s.buildDiscount > 0 {
		cost = int(float64(cost) * (1 - s.buildDiscount))
		if cost < 1 {
			cost = 1
		}
	}
	if s.gold < cost {
		return false // 金币不足
	}

	center := gm.CellCenter(row, col)
	placed := s.towers.Place(row, col, center.X, center.Y, def)
	// 初始化战力系统（base/potential 已在 pool.Place 中从 TowerDef 设置）
	if placed != nil {
		placed.Strength = strength.NewStrengthData()
		placed.RecalcStats() // 用强度100计算初始属性
	}
	s.gold -= cost
	render.InvalidateMapCache() // slot occupancy changed
	s.session.OnTowerBuilt()
	s.audioMgr.PlaySafe(gameAudio.SFXBuild)
	s.tutorial.OnEvent("towerBuilt")
	return true
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
	render.InvalidateMapCache() // slot occupancy changed
	s.selectedTower = nil
	s.audioMgr.PlaySafe(gameAudio.SFXTowerSell)
	s.showNotify(fmt.Sprintf("Sold +$%d", refund))
}

// showNotify 显示屏幕中央通知（通过 toast 系统，自动淡出）。
func (s *StageScene) showNotify(msg string) {
	hud.ShowToast(msg)
}

// debugActions 返回调试面板按钮列表（仅测试模式使用）。
func (s *StageScene) debugActions() []hud.DebugAction {
	rangeLabel := "显示射程圈"
	if s.debugShowRange {
		rangeLabel = "隐藏射程圈"
	}
	inSpawn := s.imode == modeSpawnMenu || s.imode == modeSpawnPlace
	spawnLabel := "造怪模式"
	if inSpawn {
		spawnLabel = "退出造怪"
	}

	actions := []hud.DebugAction{
		// ── 强度 ──
		{Label: "强度", IsSection: true},
		{Label: "全场塔 +50 强度", Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.AddPermanent(50)
				}
			})
		}},
		{Label: "全场塔 -50 强度", Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.AddPermanent(-50)
				}
			})
		}},
		{Label: "全场塔强度重置", Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.ResetPermanent()
				}
			})
		}},
	}

	// ── 战灵 ──
	if s.wardenUnit != nil && s.wardenUnit.Active {
		actions = append(actions,
			hud.DebugAction{Label: "战灵", IsSection: true},
			hud.DebugAction{Label: "战灵强度 +100", Action: func() {
				s.wardenUnit.SelfStrength += 100
			}},
			hud.DebugAction{Label: "战灵升级", Action: func() {
				s.wardenUnit.SelfStrength += 50
			}},
		)
	}

	// ── 经济 ──
	actions = append(actions,
		hud.DebugAction{Label: "经济", IsSection: true},
		hud.DebugAction{Label: "+500 金币", Action: func() { s.gold += 500 }},
		hud.DebugAction{Label: "+5000 金币", Action: func() { s.gold += 5000 }},
		hud.DebugAction{Label: "金币归零", Action: func() { s.gold = 0 }},
	)

	// ── 波次 ──
	actions = append(actions,
		hud.DebugAction{Label: "波次", IsSection: true},
		hud.DebugAction{Label: "跳到下一波", Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP = 0 })
			s.spawner.WaveActive = false
			s.spawner.StartNextWave()
		}},
		hud.DebugAction{Label: "清除全场敌人", Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP = 0 })
		}},
		hud.DebugAction{Label: "生成 Boss", Action: func() { s.spawnBoss() }},
	)

	// ── 塔操作 ──
	actions = append(actions,
		hud.DebugAction{Label: "塔操作", IsSection: true},
		hud.DebugAction{Label: spawnLabel, Action: func() {
			if inSpawn {
				s.imode = modeIdle
				s.spawnMode = false
				s.spawnType = ""
			} else {
				s.imode = modeSpawnMenu
				s.spawnMode = true
				s.spawnType = ""
			}
		}},
	)

	// ── 敌方 ──
	actions = append(actions,
		hud.DebugAction{Label: "敌方", IsSection: true},
		hud.DebugAction{Label: "敌方全场减 50% HP", Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP *= 0.5 })
		}},
	)

	// ── 显示 ──
	actions = append(actions,
		hud.DebugAction{Label: "显示", IsSection: true},
		hud.DebugAction{Label: rangeLabel, Action: func() { s.debugShowRange = !s.debugShowRange }},
	)

	// ── 技能 ──
	skillNames := []string{
		"chainLightning", "nukeBomb", "windBlade", "channelLaser",
		"missileBarrage", "judgmentBeam", "chainLightningBolts", "judgmentRain",
		"thunderSmite",
	}
	actions = append(actions, hud.DebugAction{Label: "技能", IsSection: true})
	for _, sn := range skillNames {
		sn := sn // capture
		actions = append(actions, hud.DebugAction{
			Label: "塔+" + sn,
			Action: func() {
				if s.selectedTower == nil {
					hud.ShowToast("先选中一座塔")
					return
				}
				t := s.selectedTower
				if t.Skill == nil {
					t.Skill = &skill.SkillState{}
				}
				skill.AssignSkill(t.Skill, sn, t)
				hud.ShowToast("塔挂载 " + sn)
			},
		})
	}
	for _, sn := range skillNames {
		sn := sn
		actions = append(actions, hud.DebugAction{
			Label:  "灵+" + sn,
			Action: func() { s.assignWardenSkill(sn) },
		})
	}

	// ── 配置审计 ──
	actions = append(actions,
		hud.DebugAction{Label: "配置审计", IsSection: true},
		hud.DebugAction{Label: "能力装备检查", Action: func() {
			audit, err := config.AuditAbilities()
			if err != nil {
				hud.ShowToast("审计失败: " + err.Error())
				return
			}
			msg := fmt.Sprintf("总%d 装备%d 储备%d", audit.Total, len(audit.Equipped), len(audit.NotEquipped))
			if len(audit.Orphaned) > 0 {
				msg += fmt.Sprintf(" 孤儿%d!", len(audit.Orphaned))
			}
			hud.ShowToast(msg)
			// 详细信息打印到控制台
			for _, a := range audit.NotEquipped {
				fmt.Printf("  储备能力: %s\n", a)
			}
			for _, a := range audit.Orphaned {
				fmt.Printf("  ⚠ 孤儿引用: %s\n", a)
			}
		}},
	)

	return actions
}

// spawnEntries 构建排序后的造怪菜单条目列表。
func (s *StageScene) spawnEntries() []hud.SpawnEntry {
	var entries []hud.SpawnEntry
	for name, cfg := range s.spawner.Archetypes {
		entries = append(entries, hud.SpawnEntry{Name: name, Config: cfg})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}

// spawnBoss 在路径起点生成一个 Boss 敌人（仅测试模式）。
func (s *StageScene) spawnBoss() {
	for name, cfg := range s.spawner.Archetypes {
		if cfg.Boss {
			path := s.gameMap.Waypoints
			if len(path) > 0 {
				s.enemies.Spawn(path[0].X, path[0].Y, cfg.HpScale*300, cfg.SpeedScale*25, 0, name, cfg)
			}
			return
		}
	}
}

// dt 固定时间步长（1/60 秒）。
const dt = 1.0 / float64(game.TargetTPS)

// updatePlaying 游戏进行中的核心循环，按管线顺序执行。
func (s *StageScene) updatePlaying() {
	s.perfTracker.BeginUpdate()
	defer s.perfTracker.EndUpdate()

	if s.imode == modePaused {
		return
	}
	// 游戏速度倍率
	gameDT := dt * float64(s.gameSpeed)

	// Screen effects update (hit-stop freezes game logic for this frame).
	if s.postPipeline.Effects.Update(gameDT) {
		// Hit-stop: skip game logic but still update VFX for visual feedback.
		render.UpdateFloatTexts(gameDT)
		render.UpdateImpactVFX(gameDT)
		return
	}

	prevWave := s.spawner.Wave
	render.UpdateVFXTick(gameDT)

	// 音效节流计时器递减

	// 模式每帧 tick（限时模式倒计时等）
	s.session.Tick(gameDT, s.buildModeCtx())

	// 0. 非手动模式：第一波倒计时结束时自动弹出战灵选择
	if !s.wardenReady && s.spawner.Wave == 0 && s.spawner.TimeToNextWave() <= 0 && !s.testMode {
		s.showWardenSelect()
		return
	}

	// 1. 生成敌人
	s.spawner.Update(s.enemies, gameDT)
	if s.spawner.Wave > prevWave {
		s.waveLivesSnapshot = s.lives
		s.session.OnWaveStart(s.spawner.Wave, s.buildModeCtx())
		s.audioMgr.PlaySafe(gameAudio.SFXWaveStart)
		if s.spawner.Wave%5 == 0 {
			s.audioMgr.PlaySafe(gameAudio.SFXBossEnter)
		}
		s.tutorial.OnEvent("waveStarted")
	}

	// 2. 敌人状态效果（减速、流血等）
	pipeline.TickEnemyStatusEffects(s.enemies, gameDT, func(e *enemy.Enemy, dmg float64) {
		render.SpawnDamageText(e.X, e.Y-10, dmg, false)
	})

	// 3. 敌人移动（到达终点扣生命）
	s.enemies.Each(func(e *enemy.Enemy) {
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, gameDT) {
			s.lives--
			s.session.OnEnemyLeaked(s.buildModeCtx())
			s.enemies.Kill(e)
			s.audioMgr.PlaySafe(gameAudio.SFXEnemyLeak)
			s.postPipeline.Effects.TriggerHitFlash(0.15)
		}
	})

	// 4. 战灵行为（未选择前跳过）
	if s.wardenReady && s.wardenUnit != nil {
		s.wardenUnit.Tick(&warden.TickContext{
			Enemies:     s.enemies,
			Towers:      s.towers,
			Projectiles: s.projectiles,
			DT:          gameDT,
			OnKill: func() {
				s.kills++
				s.gold += s.econ.KillGold() + s.killRewardBonus
				s.audioMgr.PlaySafe(gameAudio.SFXEnemyDeath)
			},
			OnFire: func() {
				s.audioMgr.PlayThrottled(gameAudio.SFXWardenFire, 100)
			},
			OnSpecial: func() {
				sfx := wardenSpecialSFX(s.wardenType)
				s.audioMgr.PlayThrottled(sfx, 200)
			},
			OnDamage: func(x, y, dmg float64, crit bool) {
				render.SpawnDamageText(x, y, dmg, crit)
			},
		})
	}

	// 5. 战灵技能 tick
	if s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Skill != nil {
		base := s.wardenUnit.BaseState()
		if base != nil {
			var enemySlice []*enemy.Enemy
			s.enemies.Each(func(e *enemy.Enemy) { enemySlice = append(enemySlice, e) })
			skill.TickEntitySkill(s.wardenUnit.Skill, base, enemySlice, gameDT, s.buildSkillContext())
		}
	}

	// 6. 能力 tick（重置属性 + 光环 buff + 区域效果 + 经济产出）
	// 必须在索敌射击之前执行，确保 Range 等属性是本帧最新值
	abilityGold := pipeline.TickTowerAbilities(s.towers, s.enemies, gameDT)
	s.gold += abilityGold

	// 6.5. 塔技能 tick
	pipeline.TickTowerSkills(s.towers, s.enemies, gameDT, s.buildSkillContext())

	// 6.6. 收集塔光源（动态光照）
	s.postPipeline.Lighting.Clear()
	lightIdx := 0
	s.towers.Each(func(t *tower.Tower) {
		if lightIdx >= postprocess.MaxLights {
			return
		}
		s.postPipeline.Lighting.AddLight(postprocess.PointLight{
			X: t.X, Y: t.Y,
			Color:     towerLightColor(t.AttackStyleID),
			Radius:    t.Range * 0.6,
			Intensity: 0.4,
		})
		lightIdx++
	})

	// 7. 塔索敌射击（按攻击方式分发）
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, s.beams, gameDT, func(style string) {
		s.audioMgr.PlayThrottled(gameAudio.FireSFXForStyle(style), 100)
	}, func(e *enemy.Enemy, damage float64, killed bool, _ string) {
		// 直接攻击方式（laser/beam/spin_aoe等）的伤害飘字
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, damage >= 50)
		}
	})

	// 7. 弹射物移动 + 光束衰减
	s.projectiles.Update(gameDT)
	s.beams.Update(gameDT)

	// 8. 弹射物命中检测（含能力触发）
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers, func(e *enemy.Enemy, damage float64, killed bool, attackStyle string) {
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, damage >= 50)
			e.HitFlash = 0.12
			// 命中特效：蓄力弹用大号，其他用通用小型
			if attackStyle == "charge" {
				render.SpawnChargeImpact(e.X, e.Y)
			} else {
				render.SpawnHitImpact(e.X, e.Y)
			}
		}
		if killed {
			particle.EmitDeathBurst(s.particlePool, e.X, e.Y)
			particle.EmitGoldCollect(s.particlePool, e.X, e.Y)
			if e.Boss {
				s.audioMgr.PlayThrottled(gameAudio.SFXEnemyDeathBoss, 50)
				s.postPipeline.Effects.TriggerHitStop(3)
			} else {
				s.audioMgr.PlayThrottled(gameAudio.SFXEnemyDeath, 50)
			}
		} else {
			// 命中音效：per-sound 节流，优先按敌人状态区分
			switch {
			case e.ShieldHP > 0:
				s.audioMgr.PlayThrottled(gameAudio.SFXHitShield, 60)
			case e.Boss:
				s.audioMgr.PlayThrottled(gameAudio.SFXHitHeavy, 60)
			default:
				s.audioMgr.PlayThrottled(gameAudio.HitSFXForStyle(attackStyle), 60)
			}
		}
	})
	s.kills += kills
	killGold := s.econ.KillGold() + s.killRewardBonus
	s.gold += kills * killGold
	if kills > 0 {
		s.tutorial.OnEvent("enemyKilled")
		for i := 0; i < kills; i++ {
			s.session.OnEnemyKilled(false, s.buildModeCtx()) // TODO: pass actual boss flag per enemy
			if s.wardenReady && s.wardenUnit != nil {
				s.wardenUnit.OnKill()
			}
		}
	}

	// 9. VFX 更新（浮动文本 + 冲击 + 粒子 + 屏幕震动）
	render.UpdateFloatTexts(gameDT)
	render.UpdateImpactVFX(gameDT)
	s.particlePool.Update(gameDT)
	render.UpdateShake(gameDT)

	// 10. 波次完成奖励 + 事件触发
	if s.spawner.Wave > prevWave && prevWave > 0 {
		ctx := s.buildModeCtx()
		result := s.session.OnWaveCleared(prevWave, ctx)
		interest := s.econ.InterestGold(s.gold)
		totalBonus := result.BonusGold + result.PerfectBonus + interest
		s.gold += totalBonus
		if s.wardenReady && s.wardenUnit != nil {
			s.wardenUnit.OnWaveClear()
		}
		if s.lives == s.waveLivesSnapshot && result.PerfectBonus > 0 {
			s.audioMgr.PlaySafe(gameAudio.SFXWaveClearPerfect)
		} else {
			s.audioMgr.PlaySafe(gameAudio.SFXWaveClear)
		}
		msg := result.Message
		if interest > 0 {
			msg += fmt.Sprintf(" +$%d interest", interest)
		}
		s.showNotify(msg)
		s.tutorial.OnEvent("waveCleared")

		// 检查是否是事件奖励波次（仅启用事件的模式）
		if s.session.Mode.EnableEvents() {
			for _, rw := range event.RewardWaves() {
				if prevWave == rw {
					s.triggerEventChoice(prevWave)
					break
				}
			}
		}
	}

	// 11. 胜负判定（委托给游戏模式）
	ctx := s.buildModeCtx()
	if s.session.CheckEndConditions(ctx) && s.state == statePlaying {
		if s.session.Status == gamemode.StatusVictory {
			s.state = stateVictory
			s.audioMgr.PlaySafe(gameAudio.SFXVictory)
			s.progressMgr.RecordGameResult(s.modeID, s.gameMap.Config.ID, s.kills, true)
			if s.tutorial.IsComplete() {
				s.progressMgr.SetTutorialDone()
			}
		} else if s.session.Status == gamemode.StatusDefeat {
			s.state = stateDefeat
			s.audioMgr.PlaySafe(gameAudio.SFXDefeat)
			s.progressMgr.RecordGameResult(s.modeID, s.gameMap.Config.ID, s.kills, false)
		}
	}
}

// triggerEventChoice 在奖励波次触发事件选择弹窗，暂停游戏等待玩家选择。
func (s *StageScene) triggerEventChoice(wave int) {
	picks := s.eventPool.PickTiered(wave)
	if len(picks) == 0 {
		return
	}
	s.eventPending = picks
	s.eventHoverIdx = -1
	s.imode = modeEvent
	s.audioMgr.PlaySafe(gameAudio.SFXChoiceAppear)
}

// ─── 事件选择弹窗 ───

// 事件卡片布局常量。
const (
	eventCardW   float32 = 200 // 卡片宽度
	eventCardH   float32 = 120 // 卡片高度
	eventCardGap float32 = 12  // 卡片间距
	eventCardR   float32 = 10  // 卡片圆角
)

// eventCardRect 计算第 i 张事件卡片的位置（居中布局）。
func eventCardRect(n, i int) (x, y float32) {
	totalW := float32(n)*eventCardW + float32(n-1)*eventCardGap
	startX := (float32(game.ScreenWidth) - totalW) / 2
	x = startX + float32(i)*(eventCardW+eventCardGap)
	y = (float32(game.ScreenHeight) - eventCardH) / 2
	return x, y
}

// handleEventSelection 处理事件选择弹窗的输入：悬停高亮和点击选择。
// 使用手势系统统一处理鼠标/触摸输入，避免拖拽误触。
func (s *StageScene) handleEventSelection() {
	g := s.gesture
	g.DragEnabled = false
	g.Update()

	mx, my := draw.CursorPos()
	fmx, fmy := float32(mx), float32(my)
	n := len(s.eventPending)

	// 更新悬停索引
	s.eventHoverIdx = -1
	for i := 0; i < n; i++ {
		cx, cy := eventCardRect(n, i)
		if fmx >= cx && fmx <= cx+eventCardW && fmy >= cy && fmy <= cy+eventCardH {
			s.eventHoverIdx = i
			break
		}
	}

	// 通过手势系统检测 Tap（桌面+移动端统一）
	if g.JustTapped() {
		tapX, tapY := g.TapPos()
		ftx, fty := float32(tapX), float32(tapY)
		for i := 0; i < n; i++ {
			cx, cy := eventCardRect(n, i)
			if ftx >= cx && ftx <= cx+eventCardW && fty >= cy && fty <= cy+eventCardH {
				chosen := s.eventPending[i]
				event.Apply(&chosen, s)
				s.appliedEvents = append(s.appliedEvents, chosen)
				s.showNotify(fmt.Sprintf("Event: %s", chosen.Label))
				s.eventPending = nil
				s.eventHoverIdx = -1
				s.imode = modeIdle
				s.audioMgr.PlaySafe(gameAudio.SFXChoiceSelect)
				return
			}
		}
	}
}

// drawEventPopup 绘制事件选择弹窗（半透明遮罩 + 标题 + 卡片列表）。
func (s *StageScene) drawEventPopup(screen *ebiten.Image) {
	if s.eventPending == nil {
		return
	}
	n := len(s.eventPending)

	// 半透明遮罩
	draw.RoundRect(screen, 0, 0, float32(game.ScreenWidth), float32(game.ScreenHeight), 0, theme.HUDGameOverlay)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	// 标题
	titleY := float64(game.ScreenHeight)/2 - float64(eventCardH)/2 - 36
	fm.DrawCenteredBoldText(screen, "选择事件", float64(game.ScreenWidth)/2, titleY, theme.FontH1, theme.TextTitle)

	// 事件卡片
	for i, ev := range s.eventPending {
		cx, cy := eventCardRect(n, i)

		// 卡片背景
		if i == s.eventHoverIdx {
			draw.RoundRect(screen, cx, cy, eventCardW, eventCardH, eventCardR, theme.TonePrimary)
		} else {
			draw.RoundRect(screen, cx, cy, eventCardW, eventCardH, eventCardR, theme.PanelBg)
		}

		// 卡片描边
		draw.StrokeRoundRect(screen, cx, cy, eventCardW, eventCardH, eventCardR, 1.5, theme.PanelBorder)

		// 事件名称（卡片上方居中）
		labelX := float64(cx) + float64(eventCardW)/2
		labelY := float64(cy) + 16
		fm.DrawCenteredBoldText(screen, ev.Label, labelX, labelY, theme.FontH2, theme.TextTitle)

		// 事件描述（卡片中部居中）
		descX := float64(cx) + float64(eventCardW)/2
		descY := float64(cy) + 48
		fm.DrawCenteredText(screen, ev.Description, descX, descY, theme.FontBody, theme.TextBody)

		// Tier 标签（卡片底部）
		tierLabel := fmt.Sprintf("Tier %d", ev.Tier)
		tierX := float64(cx) + float64(eventCardW)/2
		tierY := float64(cy) + float64(eventCardH) - 24
		tierColor := theme.TextMuted
		if ev.Tier >= 2 {
			tierColor = theme.StatusSkill
		}
		fm.DrawCenteredText(screen, tierLabel, tierX, tierY, theme.FontCaption, tierColor)
	}
}

// ─── GameState 接口实现（供事件处理器调用）───

func (s *StageScene) AddGold(amount int)             { s.gold += amount }
func (s *StageScene) SetBuildDiscount(ratio float64) { s.buildDiscount = ratio }
func (s *StageScene) SetKillRewardBonus(extra int)   { s.killRewardBonus = extra }

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

// Draw 渲染游戏画面：地图 → 塔 → 敌人 → 弹射物 → 预览 → HUD → 通知 → 胜负覆盖。
// shakeBuffer 屏幕震动用的离屏缓冲（懒初始化）。
var shakeBuffer *ebiten.Image

// worldBuffer 大地图相机偏移用的离屏缓冲（懒初始化）。
var worldBuffer *ebiten.Image

func (s *StageScene) Draw(screen *ebiten.Image) {
	s.perfTracker.BeginDraw()
	defer s.perfTracker.EndDraw()

	// 屏幕震动：先画到缓冲，再偏移 blit 到 screen
	dx, dy := render.ShakeOffset()
	target := screen
	if dx != 0 || dy != 0 {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		if shakeBuffer == nil || shakeBuffer.Bounds().Dx() != w {
			shakeBuffer = ebiten.NewImage(w, h)
		}
		shakeBuffer.Clear()
		target = shakeBuffer
	}

	s.drawScene(target)

	if target != screen {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(dx, dy)
		screen.DrawImage(target, opts)
	}
}

func (s *StageScene) drawScene(screen *ebiten.Image) {
	useCamera := s.needsCamera()

	// 后处理管线：世界元素渲染到 sceneTarget，bloom 后输出到 screen。
	physW, physH := screen.Bounds().Dx(), screen.Bounds().Dy()
	sceneTarget := s.postPipeline.SceneBuffer(physW, physH)

	// 确定世界元素的绘制目标：大地图时使用离屏缓冲，小地图画到 sceneTarget。
	worldTarget := sceneTarget
	if useCamera {
		// worldBuffer 按地图完整尺寸 × draw.Scale 创建（覆盖整个世界）
		logicalW := s.gameMap.Width() + s.gameMap.OffsetX*2
		logicalH := s.gameMap.Height() + s.gameMap.OffsetY*2
		// 至少覆盖屏幕大小
		screenLogW := float64(game.ScreenWidth)
		screenLogH := float64(game.ScreenHeight)
		if logicalW < screenLogW {
			logicalW = screenLogW
		}
		if logicalH < screenLogH {
			logicalH = screenLogH
		}
		bufW := int(logicalW * draw.Scale)
		bufH := int(logicalH * draw.Scale)
		if worldBuffer == nil || worldBuffer.Bounds().Dx() != bufW || worldBuffer.Bounds().Dy() != bufH {
			worldBuffer = ebiten.NewImage(bufW, bufH)
		}
		worldBuffer.Clear()
		worldTarget = worldBuffer
	}

	// ── 世界元素（受相机偏移影响）──

	// 地图（渐变背景覆盖全屏，无需 Fill）
	inBuildMode := s.imode == modeBuildMenu || s.imode == modeBuildPlace
	render.DrawMap(worldTarget, s.gameMap, render.GlobalFont(), float64(s.frame)/60.0, func(row, col int) bool {
		return s.towers.At(row, col) != nil
	}, inBuildMode)

	// 塔（优先 SVG 渲染，回退到彩色方块）
	animTime := float64(s.frame) / 60.0
	s.towerRenderer.DrawTowers(worldTarget, s.towers, s.selectedTower, animTime)

	// 被 buff 的塔显示强化特效（五角星芒）
	s.towers.Each(func(t *tower.Tower) {
		if len(t.Buffs) > 0 {
			render.DrawTowerBuffEffect(worldTarget, t, animTime)
		}
	})

	// 塔技能 VFX + CD 进度条
	s.towers.Each(func(t *tower.Tower) {
		if t.Skill != nil {
			render.DrawSkillVFX(worldTarget, t.Skill)
			render.DrawSkillBar(worldTarget, t.X, t.Y, t.Skill, t)
		}
	})

	// 调试射程圈（测试模式下显示所有塔的射程）
	if s.debugShowRange {
		s.towers.Each(func(t *tower.Tower) {
			draw.CircleOutline(worldTarget, float32(t.X), float32(t.Y), float32(t.Range), 1, color.RGBA{R: 60, G: 120, B: 200, A: 80})
		})
	}

	// 敌人（优先 SVG 渲染）
	s.enemyRenderer.DrawEnemies(worldTarget, s.enemies, animTime)

	// 弹射物
	render.DrawProjectiles(worldTarget, s.projectiles)
	render.DrawBeams(worldTarget, s.beams)

	// 冲击特效（蓄力弹命中）
	render.DrawImpactVFX(worldTarget)

	// 粒子系统
	s.particlePool.Draw(worldTarget)

	// 浮动文本（伤害数字等）
	render.DrawFloatTexts(worldTarget)

	// 战灵（选择后才绘制）
	if s.wardenReady && s.wardenUnit != nil {
		s.wardenRenderer.DrawWarden(worldTarget, s.wardenUnit)
		// 战灵技能 VFX + CD 进度条
		if s.wardenUnit.Skill != nil {
			render.DrawSkillVFX(worldTarget, s.wardenUnit.Skill)
			if base := s.wardenUnit.BaseState(); base != nil {
				render.DrawSkillBar(worldTarget, base.X, base.Y, s.wardenUnit.Skill, base)
			}
		}
		// 战灵面板展开时显示攻击距离圈
		if s.wardenPanelOpen {
			if base := s.wardenUnit.BaseState(); base != nil && base.Range > 0 {
				draw.DashedCircle(worldTarget, float32(base.X), float32(base.Y),
					float32(base.Range), 1, 6, 4, color.RGBA{R: 180, G: 140, B: 255, A: 100})
			}
		}
	}

	// 放塔预览（鼠标在可建造位置时显示）
	// 放塔预览：仅在放塔模式下，鼠标悬停可建位时显示射程圈
	if s.state == statePlaying && s.imode == modeBuildPlace {
		pmx, pmy := s.gesture.CursorPos()
		wmx, wmy := s.screenToWorld(pmx, pmy)
		cellType := s.gameMap.CellAt(wmx, wmy)
		if cellType == config.CellBuildable {
			cs := float64(s.gameMap.CellSize)
			col := int((wmx - s.gameMap.OffsetX) / cs)
			row := int((wmy - s.gameMap.OffsetY) / cs)
			center := s.gameMap.CellCenter(row, col)
			def := s.towerDefs[s.selectedDef]
			valid := s.towers.At(row, col) == nil && s.gold >= def.Cost
			render.DrawTowerRangePreview(worldTarget, float32(center.X), float32(center.Y), def.Range, valid)
		}
	}

	// ── 将世界缓冲 blit 到 sceneTarget（带相机偏移） ──
	if useCamera {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-s.camX*draw.Scale, -s.camY*draw.Scale)
		sceneTarget.DrawImage(worldBuffer, opts)
	}

	// ── 后处理（bloom）→ 输出到 screen ──
	s.postPipeline.Apply(screen)

	// ── HUD 元素（不受相机偏移影响，直接画到 screen）──

	// HUD：顶部状态栏
	hud.DrawTopBar(screen, hud.TopBarData{
		Gold:          s.gold,
		Lives:         s.lives,
		Wave:          s.spawner.Wave,
		MaxWaves:      s.spawner.MaxWaves,
		Kills:         s.kills,
		Enemies:       s.enemies.Count,
		Speed:         s.gameSpeed,
		WaveCountdown: s.spawner.TimeToNextWave(),
		TestMode:      s.testMode,
		SpawnMode:     s.imode == modeSpawnMenu || s.imode == modeSpawnPlace,
		DebugOpen:     s.debugPanelOpen,
	})

	// HUD：底部建塔菜单
	hud.DrawBuildMenu(screen, hud.BuildMenuData{
		TowerDefs:   s.towerDefs,
		SelectedIdx: s.selectedDef,
		Gold:        s.gold,
		HoverIdx:    s.buildHoverIdx,
		SpriteFunc:  s.towerRenderer.GetSprite,
		Visible:     s.imode == modeBuildMenu,
	})

	// HUD：底部中央面板（塔信息 和 战灵信息 互斥）
	if s.selectedTower != nil {
		// 塔选中时显示塔信息面板（底部中央）
		sellValue := s.econ.SellRefund(s.selectedTower.Cost)
		hud.DrawInfoPanel(screen, s.selectedTower, sellValue)
		// Hover 在面板上时显示升级详情浮窗
		mx, my := draw.CursorPos()
		hud.DrawInfoPanelHoverTooltip(screen, s.selectedTower, float32(mx), float32(my))
	} else if s.wardenPanelOpen && s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		// 无塔选中且战灵面板展开时显示战灵面板（底部中央）
		hud.DrawWardenPanel(screen, s.buildWardenPanelData())
	}

	// 左下角：波次面板（可收起）
	if s.wavePanelOpen {
		hud.DrawWavePanel(screen, hud.WavePanelData{
			WaveNum:    s.spawner.Wave,
			MaxWaves:   s.spawner.MaxWaves,
			EnemyCount: s.enemies.Count,
		})
	}
	// 左下角收起按钮
	hud.DrawToggleButton(screen, true, s.wavePanelOpen, ">")

	// 右下角收起按钮（战灵，选择后才显示）
	if s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		hud.DrawToggleButton(screen, false, s.wardenPanelOpen, "⚡")
	}

	// 教程提示（顶部居中）
	if msg := s.tutorial.CurrentMessage(); msg != "" {
		if fm := render.GlobalFont(); fm != nil {
			fm.DrawCenteredText(screen, msg, float64(game.ScreenWidth)/2, 50, theme.FontH2, theme.TextBody)
		}
	}

	// 调试面板（测试模式）
	if s.testMode && s.debugPanelOpen {
		hud.DrawDebugPanel(screen, hud.DebugPanelData{Actions: s.debugActions()})
	}

	// 调试覆盖层：实体统计 + 性能统计（F2 切换）
	s.debugOverlay.DrawHUD(screen,
		s.towers.Count, s.enemies.Count,
		s.beams.Count(), s.projectiles.Count)
	s.debugOverlay.DrawPerf(screen, s.perfTracker)

	// 造怪选择菜单（测试模式 FSM）
	if s.imode == modeSpawnMenu {
		hud.DrawSpawnMenu(screen, hud.SpawnMenuData{
			Entries:    s.spawnEntries(),
			HoverIdx:   s.spawnHoverIdx,
			SpriteFunc: s.enemyRenderer.GetSprite,
		})
	}

	// 暂停菜单覆盖层
	if s.imode == modePaused {
		hud.DrawPauseMenu(screen)
	}

	// Toast 通知
	hud.DrawToast(screen)

	// 战灵选择覆盖层
	if s.wardenOverlay != nil {
		s.wardenOverlay.Draw(screen)
	}

	// 事件选择弹窗
	s.drawEventPopup(screen)

	// 胜负覆盖层
	if s.state == stateVictory || s.state == stateDefeat {
		// 半透明遮罩
		draw.RoundRect(screen, 0, 0, float32(game.ScreenWidth), float32(game.ScreenHeight), 0, theme.HUDGameOverlay)
		if fm := render.GlobalFont(); fm != nil {
			if s.state == stateVictory {
				fm.DrawCenteredText(screen, "VICTORY!", float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, theme.FontGameOver, theme.HUDVictoryColor)
			} else {
				fm.DrawCenteredText(screen, "DEFEAT!", float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, theme.FontGameOver, theme.HUDDefeatColor)
			}
			fm.DrawCenteredText(screen, fmt.Sprintf("击杀: %d  点击继续", s.kills), float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2+30, theme.FontH2, theme.TextMuted)
		}
	}
}

// buildWardenPanelData 根据当前战灵状态和配置构建面板显示数据。
func (s *StageScene) buildWardenPanelData() hud.WardenPanelData {
	w := s.wardenUnit
	d := hud.WardenPanelData{
		Type:         w.Type,
		Strength:     w.PerceivedStrength,
		PeakStrength: w.PeakStrength,
	}

	cfg := s.wardenCfg
	if cfg == nil {
		d.Name = w.Name
		return d
	}
	d.Name = cfg.Name

	// 运行时占位符参数（含强度缩放后的实际值）
	params := w.DescParams()

	// 有效伤害（优先用运行时值）
	if base := w.BaseState(); base != nil && base.Damage > 0 {
		d.Damage = fmt.Sprintf("%.0f", base.Damage)
	} else if cfg.Damage > 0 {
		d.Damage = fmt.Sprintf("%.0f", cfg.Damage)
	}
	if cfg.AttackInterval > 0 {
		d.Interval = fmt.Sprintf("%.1fs", cfg.AttackInterval)
	}

	// 行为描述（替换占位符为实际值）
	d.AttackDesc = replaceDescParams(cfg.AttackDesc, params)
	if cfg.SpecialDesc != "" {
		d.SpecialDesc = replaceDescParams(cfg.SpecialDesc, params)
	}

	// 成长信息
	if cfg.GrowthOnKill > 0 || cfg.GrowthOnWaveClear > 0 {
		parts := ""
		if cfg.GrowthOnKill > 0 {
			parts += fmt.Sprintf("击杀+%.0f", cfg.GrowthOnKill)
		}
		if cfg.GrowthOnWaveClear > 0 {
			if parts != "" {
				parts += " "
			}
			parts += fmt.Sprintf("通波+%.0f", cfg.GrowthOnWaveClear)
		}
		d.GrowthDesc = parts
	}

	return d
}

// wardenSpecialSFX 根据战灵类型返回特殊能力音效名。
func wardenSpecialSFX(typ string) string {
	switch typ {
	case "prince":
		return gameAudio.SFXWardenSpecialFire
	case "core":
		return gameAudio.SFXWardenSpecialMech
	case "chain":
		return gameAudio.SFXWardenSpecialChain
	case "skystrike":
		return gameAudio.SFXWardenSpecialWater
	case "envoy":
		return gameAudio.SFXWardenSpecialGold
	default:
		return gameAudio.SFXWardenFire
	}
}

// assignWardenSkill 为战灵挂载技能（调试用）。
func (s *StageScene) assignWardenSkill(name string) {
	if s.wardenUnit == nil {
		hud.ShowToast("无战灵")
		return
	}
	base := s.wardenUnit.BaseState()
	if base == nil {
		return
	}
	if s.wardenUnit.Skill == nil {
		s.wardenUnit.Skill = &skill.SkillState{}
	}
	skill.AssignSkill(s.wardenUnit.Skill, name, base)
	hud.ShowToast("战灵挂载 " + name)
}

// buildSkillContext 构建技能执行上下文。
func (s *StageScene) buildSkillContext() *skill.SkillContext {
	return &skill.SkillContext{
		Projectiles: s.projectiles,
		Beams:       s.beams,
		OnHit: func(e *enemy.Enemy, dmg float64, killed bool) {
			render.SpawnDamageText(e.X, e.Y-10, dmg, dmg >= 50)
			if killed {
				s.kills++
				s.gold += s.econ.KillGold() + s.killRewardBonus
				s.audioMgr.PlaySafe(gameAudio.SFXEnemyDeath)
			}
		},
		OnActivate: func(skillKey string) {
			if sfx := gameAudio.SkillSFX(skillKey); sfx != "" {
				s.audioMgr.PlaySafe(sfx)
			}
		},
		PlaySFX: func(name string) {
			s.audioMgr.PlayThrottled(name, 300)
		},
	}
}

// replaceDescParams 将描述字符串中的 {key} 占位符替换为实际值。
func replaceDescParams(desc string, params map[string]string) string {
	if params == nil {
		return desc
	}
	result := desc
	for k, v := range params {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

// convertArchetypesToSpawnConfigs 将 config.EnemyArchetype 转换为 enemy.SpawnConfig。
// 使 enemy 包不依赖 config 包。
func convertArchetypesToSpawnConfigs(archetypes map[string]*config.EnemyArchetype) map[string]*enemy.SpawnConfig {
	result := make(map[string]*enemy.SpawnConfig, len(archetypes))
	for key, a := range archetypes {
		result[key] = &enemy.SpawnConfig{
			Label:       a.Label,
			HpScale:     a.HPScale,
			SpeedScale:  a.SpeedScale,
			Radius:      a.Radius,
			Boss:        a.Boss,
			ShieldScale: a.ShieldScale,
		}
	}
	return result
}

// ── 战灵选择逻辑 ────────────────────────────────────

// tryStartWave 尝试开波。若战灵未选择则先弹出战灵选择面板。
func (s *StageScene) tryStartWave() {
	if !s.wardenReady {
		s.showWardenSelect()
		return
	}
	s.spawner.StartNextWave()
	s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
}

// showWardenSelect 弹出战灵选择覆盖层。
func (s *StageScene) showWardenSelect() {
	s.wardenOverlay.Show(func(key string) {
		s.activateWarden(key)
		// 选完后立即开第一波
		s.spawner.StartNextWave()
		s.audioMgr.PlaySafe(gameAudio.SFXUIClick)
	})
	s.imode = modeWardenSelect
}

// activateWarden 激活战灵。
func (s *StageScene) activateWarden(key string) {
	s.wardenType = key
	s.wardenReady = true
	s.imode = modeIdle

	if key == "" {
		// "不选" — wardenUnit 保持 nil
		return
	}
	s.wardenUnit = warden.NewWarden(1, key, key)

	// 加载战灵配置：驱动 Init 参数 + 面板显示
	if cfgs, err := config.LoadWardenConfigs(); err == nil {
		if wc, ok := cfgs[key]; ok {
			s.wardenCfg = &wc
			// JSON 驱动覆盖 Init 硬编码的基础属性
			if base := s.wardenUnit.BaseState(); base != nil {
				if wc.Damage > 0 {
					base.Damage = wc.Damage
				}
				if wc.AttackInterval > 0 {
					base.AttackInterval = wc.AttackInterval
				}
				if wc.Range > 0 {
					base.Range = wc.Range
				}
				if wc.MoveSpeed > 0 {
					base.MoveSpeed = wc.MoveSpeed
				}
			}
			// 覆盖成长参数
			if wc.GrowthOnKill > 0 {
				s.wardenUnit.GrowthOnKill = wc.GrowthOnKill
			}
			if wc.GrowthOnWaveClear > 0 {
				s.wardenUnit.GrowthOnWaveClear = wc.GrowthOnWaveClear
			}
		}
	}

	// 非手动模式下，恢复自动开波
	if !s.testMode && !s.manualWave {
		s.spawner.ManualWave = false
	}
}

// handleWardenSelection 战灵选择覆盖层的交互处理。
func (s *StageScene) handleWardenSelection() {
	if s.wardenOverlay == nil || !s.wardenOverlay.Active {
		s.imode = modeIdle
		return
	}
	mx, my := draw.CursorPos()
	s.wardenOverlay.Update(mx, my, isTapJustPressed())
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

// towerLightColor maps a tower attack style to a light color for dynamic lighting.
func towerLightColor(style string) color.RGBA {
	switch style {
	case "laser", "wideBeam":
		return color.RGBA{R: 255, G: 80, B: 80, A: 255} // red
	case "scatter":
		return color.RGBA{R: 100, G: 180, B: 255, A: 255} // ice blue
	case "charge":
		return color.RGBA{R: 255, G: 255, B: 100, A: 255} // electric yellow
	case "spin_aoe":
		return color.RGBA{R: 255, G: 120, B: 30, A: 255} // fire orange
	case "aura_dot":
		return color.RGBA{R: 150, G: 255, B: 150, A: 255} // poison green
	default:
		return color.RGBA{R: 255, G: 240, B: 220, A: 255} // warm white
	}
}
