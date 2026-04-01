// stage.go — 游戏主场景。
// 管理塔防核心循环：生成敌人、移动、塔战斗、弹射物、经济、胜负判定。
package scene

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "defense2/internal/core/tower/abilities" // 通过 init() 注册塔能力
	_ "defense2/internal/core/warden/types"    // 通过 init() 注册战灵类型

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/achievement"
	"defense2/internal/core/combat"
	"defense2/internal/core/debug"
	"defense2/internal/core/economy"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/item"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/core/gamemode"
	"defense2/internal/core/persistence"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
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

// 类型定义 (stageState/interactMode/StageOptions) 已移至 stage_types.go。

// StageScene 游戏主场景，包含所有运行时游戏状态。
type StageScene struct {
	switcher          Switcher                     // 场景切换器引用
	bus               *event.Bus                   // 事件总线（从 Switcher 获取）
	busSubscribed     bool                         // Bus 订阅是否已完成（延迟到首次 Update）
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
	killRewardBonus   int                          // 额外击杀金币（事件增益）
	buildDiscount     float64                      // 建造折扣比例（事件增益）
	tutorial          *tutorial.Tutorial           // 新手教程
	progressMgr       *persistence.ProgressManager // 持久化进度管理器
	lastWave          int                          // 上一帧的波次号
	wardenType        string                       // 战灵类型标识（用于重玩传递）
	wardenCfg         *config.WardenConfig         // 战灵配置（用于面板显示）
	// 道具系统
	inventory      *item.Inventory // 道具背包
	dragItemKind   item.Kind       // 当前拖拽的道具类型
	dragItemActive bool            // 是否正在拖拽道具
	dragHoverTower *tower.Tower    // 拖拽道具时悬停的目标塔
	itemPanelOpen  bool            // 道具面板是否打开
	gameSpeed         int                          // 游戏速度倍率（1 或 2）
	imode             interactMode                 // 交互状态机
	prePauseMode      interactMode                 // 暂停前的交互模式（恢复用）
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
	// 能力升级
	wavesCleared int // 已清除波次数（用于能力解锁）
	// 测试模式
	// HUD 面板状态
	wavePanelOpen   bool             // 左下角波次面板是否展开
	wavePanelState  hud.WavePanelState // 抽屉动画状态
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
	qualityAdaptive *game.QualityAdaptive     // 自适应画质调节器
	waveAnnounce    *hud.WaveAnnounce          // 波次开始公告动画
	ambientTimer    float64                    // 环境粒子发射计时器（每秒一次）
	multiKillCount  int                        // 连续击杀计数
	multiKillTimer  float64                    // 连杀窗口倒计时（1.5s 无击杀后重置）
	choicePanel       *hud.ChoicePanel            // 能力选择覆盖层
	autoPlayer        AutoPlayer                 // 自动对局驱动（nil=手动模式）
	screenshotPending bool                       // F12 截图请求标志
	achieveTracker    *achievement.Tracker       // 成就追踪器
	gameStats         GameStats                  // 详细游戏统计
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
	achTracker := achievement.NewTracker(store)

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
		if base := wardenUnit.BaseState(); base != nil {
			base.MapWidth = gm.PixelWidth()
			base.MapHeight = gm.PixelHeight()
		}
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
		achieveTracker:  achTracker,
	}

	// 后处理管线（bloom）+ 粒子系统
	s.postPipeline = postprocess.NewPipeline()
	s.particlePool = particle.NewPool()

	// 调试覆盖层 + 性能追踪器 + 自适应画质
	s.debugOverlay = hud.NewDebugOverlay()
	s.perfTracker = debug.NewPerfTracker()
	s.qualityAdaptive = game.NewQualityAdaptive()
	s.waveAnnounce = hud.NewWaveAnnounce()
	s.choicePanel = hud.NewChoicePanel()
	s.particlePool.MaxActive = game.Settings().MaxParticles
	s.inventory = item.NewInventory(5)

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

	// 事件总线引用（订阅延迟到首次 Update，避免被场景切换淡出的 bus.Clear 清掉）
	s.bus = sw.EventBus()

	return s
}

// subscribeBus 注册所有事件总线订阅。
// 注册顺序即执行顺序，关键路径（如 gold 变更）应在统计类订阅之前。
func (s *StageScene) subscribeBus() {
	bus := s.bus

	// ── 塔事件 ──────────────────────────────────
	event.OnTyped(bus, event.EvtTowerBuilt, func(p event.TowerBuiltPayload) {
		s.session.OnTowerBuilt()
		s.audioMgr.PlaySafeAt(gameAudio.SFXBuild, gameAudio.VolBuild)
		s.tutorial.OnEvent("towerBuilt")
		// 成就: 累计建塔 + 单局塔种类
		s.achieveTracker.IncrTowersBuilt()
		if s.achieveTracker.TotalTowersBuilt >= 10 {
			if s.achieveTracker.Unlock("builder_10") {
				hud.ShowToast("成就解锁: 塔防新手")
			}
		}
		s.achieveTracker.SessionTowerTypes[p.TowerKey] = true
		if len(s.achieveTracker.SessionTowerTypes) >= 5 {
			if s.achieveTracker.Unlock("all_towers") {
				hud.ShowToast("成就解锁: 全能战士")
			}
		}
	})
	event.OnTyped(bus, event.EvtTowerUpgraded, func(_ event.TowerUpgradedPayload) {
		s.audioMgr.PlaySafeAt(gameAudio.SFXUpgrade, gameAudio.VolBuild)
		s.tutorial.Trigger("upgrade")
	})
	event.OnTyped(bus, event.EvtTowerSold, func(_ event.TowerSoldPayload) {
		s.audioMgr.PlaySafeAt(gameAudio.SFXTowerSell, gameAudio.VolBuild)
	})

	// ── 波次事件 ─────────────────────────────────
	event.OnTyped(bus, event.EvtWaveStarted, func(p event.WaveStartedPayload) {
		s.session.OnWaveStart(p.Wave, s.buildModeCtx())
		s.audioMgr.PlaySafeAt(gameAudio.SFXWaveStart, gameAudio.VolWave)
		if p.IsBoss {
			s.audioMgr.PlaySafeAt(gameAudio.SFXBossEnter, gameAudio.VolWave)
			// BGM: Boss 波切换到 Boss 音乐
			s.audioMgr.PlayBGM(gameAudio.BGMBoss)
		}
		s.waveAnnounce.Trigger(p.Wave, s.spawner.MaxWaves, p.IsBoss)
		s.tutorial.OnEvent("waveStarted")
	})
	event.OnTyped(bus, event.EvtWaveCleared, func(p event.WaveClearedPayload) {
		prevUnlocked := tower.UnlockedSlots(s.wavesCleared)
		s.wavesCleared++
		newUnlocked := tower.UnlockedSlots(s.wavesCleared)
		// 为所有塔 roll 新解锁能力位的选项
		newPending := 0
		s.towers.Each(func(t *tower.Tower) {
			before := tower.PendingCount(t)
			tower.RollAndCachePendingChoices(t, s.wavesCleared)
			after := tower.PendingCount(t)
			if after > before {
				newPending++
			}
		})
		if newUnlocked > prevUnlocked && newPending > 0 {
			hud.ShowToast(fmt.Sprintf("新能力位解锁! %d座塔可选择能力", newPending))
		}
		if s.wardenReady && s.wardenUnit != nil {
			s.wardenUnit.OnWaveClear()
		}
		if p.Perfect {
			s.audioMgr.PlaySafeAt(gameAudio.SFXWaveClearPerfect, gameAudio.VolWave)
		} else {
			s.audioMgr.PlaySafeAt(gameAudio.SFXWaveClear, gameAudio.VolWave)
		}
		// BGM: 波次清除后恢复战斗音乐（Boss 波结束时从 BGMBoss 切回）
		s.audioMgr.PlayBGM(gameAudio.BGMBattle)
		s.tutorial.OnEvent("waveCleared")
		// 成就: Endless 模式 50 波
		if s.modeID == "endless" && p.Wave >= 50 {
			if s.achieveTracker.Unlock("endless_50") {
				hud.ShowToast("成就解锁: 不灭传说")
			}
		}
	})

	// ── 敌人事件 ─────────────────────────────────
	event.OnTyped(bus, event.EvtEnemyLeaked, func(_ event.EnemyLeakedPayload) {
		s.session.OnEnemyLeaked(s.buildModeCtx())
		s.gameStats.LeaksTotal++
		s.audioMgr.PlaySafeAt(gameAudio.SFXEnemyLeak, gameAudio.VolWave)
	})
	event.OnTyped(bus, event.EvtEnemyKilled, func(p event.EnemyKilledPayload) {
		s.kills++
		s.gold += p.GoldValue
		s.gameStats.GoldEarned += p.GoldValue
		s.session.OnEnemyKilled(p.IsBoss, s.buildModeCtx())
		s.tutorial.OnEvent("enemyKilled")
		if s.wardenReady && s.wardenUnit != nil {
			s.wardenUnit.OnKill()
		}
		// 成就: 击杀数 + Boss + 金币
		s.achieveTracker.SessionKills++
		if s.achieveTracker.SessionKills >= 100 {
			if s.achieveTracker.Unlock("centurion") {
				hud.ShowToast("成就解锁: 百杀")
			}
		}
		if p.IsBoss {
			if s.achieveTracker.Unlock("first_boss") {
				hud.ShowToast("成就解锁: 首个Boss")
			}
		}
		if s.gold > s.achieveTracker.SessionMaxGold {
			s.achieveTracker.SessionMaxGold = s.gold
		}
		if s.achieveTracker.SessionMaxGold >= 1000 {
			if s.achieveTracker.Unlock("rich") {
				hud.ShowToast("成就解锁: 富甲一方")
			}
		}
	})
}

// emitKill 统一发出击杀事件（弹射物/战灵/技能共用）。
func (s *StageScene) emitKill(isBoss bool, killerID string) {
	s.bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{
		IsBoss:    isBoss,
		KillerID:  killerID,
		GoldValue: s.econ.KillGold() + s.killRewardBonus,
	})
}

// checkVictoryAchievements checks and unlocks all victory-related achievements.
func (s *StageScene) checkVictoryAchievements() {
	t := s.achieveTracker

	// first_win — any victory
	if t.Unlock("first_win") {
		hud.ShowToast("成就解锁: 初次胜利")
	}

	// Star rating (same logic as result.go calcStars)
	stars := 1
	if s.spawner.MaxWaves > 0 && s.spawner.Wave >= s.spawner.MaxWaves {
		stars = 3
	} else if s.spawner.MaxWaves > 0 && float64(s.spawner.Wave) >= float64(s.spawner.MaxWaves)*0.8 {
		stars = 2
	}

	// perfect_star — any map 3 stars
	if stars == 3 {
		if t.Unlock("perfect_star") {
			hud.ShowToast("成就解锁: 完美主义")
		}
	}

	// no_leak_hard — Hard difficulty, zero leaks
	if s.diffID == "hard" && s.session.Stats.Leaked == 0 {
		if t.Unlock("no_leak_hard") {
			hud.ShowToast("成就解锁: 零泄漏")
		}
	}

	// speedrun — victory within 10 minutes
	if s.session.ElapsedTime <= 600 {
		if t.Unlock("speedrun") {
			hud.ShowToast("成就解锁: 速通")
		}
	}

	// extreme_master — any Extreme victory
	if s.diffID == "extreme" {
		if t.Unlock("extreme_master") {
			hud.ShowToast("成就解锁: 大师")
		}
		// extreme_perfect — Extreme + 3 stars
		if stars == 3 {
			if t.Unlock("extreme_perfect") {
				hud.ShowToast("成就解锁: 完美大师")
			}
		}
	}
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
	// 延迟订阅 Bus：避免在构造函数中订阅后被场景切换淡出的 bus.Clear() 清掉
	if !s.busSubscribed {
		s.busSubscribed = true
		s.subscribeBus()
		// BGM: 进入战斗场景播放战斗音乐
		s.audioMgr.PlayBGM(gameAudio.BGMBattle)
	}
	s.frame++

	// F12 / 截图按钮：任意状态可用（不受交互模式限制）
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		s.screenshotPending = true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := draw.CursorPos()
		if hud.TopBarHitTest(float32(mx), float32(my)) == "screenshot" {
			s.screenshotPending = true
		}
	}

	switch s.state {
	case statePlaying:
		// 交互状态机驱动
		switch s.imode {
		case modePaused:
			s.handlePausedInput()
		case modeWardenSelect:
			if s.autoPlayer != nil {
				// 自动对局：自动选择战灵
				snap := s.buildAutoPlaySnapshot()
				actions := s.autoPlayer.OnUpdate(snap)
				for _, a := range actions {
					if a.Type == APActionSelectWarden {
						s.executeAutoPlayAction(a)
						break
					}
				}
			} else {
				s.handleWardenSelection()
			}
		default:
			s.handleInput()
			s.updatePlaying()
		}
	case stateVictory, stateDefeat:
		if s.autoPlayer != nil {
			// 自动对局：通知结束，等 Done() 返回 true 后再退出（让 Draw 截到结果画面）
			s.autoPlayer.OnGameEnd(s.buildAutoPlaySnapshot(), s.state == stateVictory)
			if s.autoPlayer.Done() {
				return ebiten.Termination
			}
			return nil
		}
		// 胜利/失败状态：点击/触摸进入结算场景
		if isTapJustPressed() {
			s.audioMgr.PlaySafeAt(gameAudio.SFXUIClick, gameAudio.VolUI)
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

	// 教程自动推进计时器
	s.tutorial.Update(dt)
	if s.tutorial.Done {
		s.progressMgr.SetTutorialDone()
	}

	// 自适应画质：根据帧耗时动态调整画质等级
	totalMs := s.perfTracker.AvgUpdateMs + s.perfTracker.AvgDrawMs
	s.qualityAdaptive.Tick(totalMs)
	s.particlePool.MaxActive = game.Settings().MaxParticles

	return nil
}


// handleInput 等输入方法已移至 stage_input.go。


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
		placed.BuildAnim = 0.3 // build-in animation
		tower.RollAndCachePendingChoices(placed, s.wavesCleared)
	}
	s.gold -= cost
	s.gameStats.GoldSpent += cost
	s.gameStats.TowersBuilt++
	render.InvalidateMapCache() // slot occupancy changed
	s.bus.Emit(event.EvtTowerBuilt, event.TowerBuiltPayload{TowerKey: def.Key, Cost: cost})
	return true
}

// trySellTower 尝试出售像素位置上的塔。
// 不立即删除，而是启动出售动画，动画结束后由 updatePlaying 移除。
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
	if t == nil || t.Selling {
		return
	}
	refund := s.econ.SellRefund(t.Cost)
	s.gold += refund
	s.gameStats.TowersSold++
	// Start sell animation instead of immediate removal
	t.SellAnim = 0.25
	t.Selling = true
	particle.EmitGoldCollect(s.particlePool, t.X, t.Y)
	render.SpawnGoldText(t.X, t.Y-10, refund)
	s.selectedTower = nil
	s.bus.Emit(event.EvtTowerSold, event.TowerSoldPayload{TowerKey: t.Key, Refund: refund})
	s.showNotify(fmt.Sprintf("已卖出 +$%d", refund))
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
			prevWave := s.spawner.Wave
			s.spawner.StartNextWave()
			if s.spawner.Wave > prevWave {
				s.onWaveTransition(prevWave)
			}
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
		entries = append(entries, hud.SpawnEntry{
			Name: name, Label: cfg.Label,
			HpScale: cfg.HpScale, SpeedScale: cfg.SpeedScale,
			Radius: cfg.Radius,
			Reward: cfg.Reward, Boss: cfg.Boss,
		})
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

	// Day/night cycle: warm(early) → cool(late) based on wave progress
	if s.spawner.MaxWaves > 0 {
		progress := float64(s.spawner.Wave) / float64(s.spawner.MaxWaves) // 0→1
		fx := s.postPipeline.Effects
		// Early (warm amber): R=1.0 G=0.9 B=0.7
		// Late (cool blue):   R=0.6 G=0.7 B=1.0
		fx.DayNightR = 1.0 - 0.4*progress
		fx.DayNightG = 0.9 - 0.2*progress
		fx.DayNightB = 0.7 + 0.3*progress
		fx.DayNightA = 0.06 // subtle — 6% blend
	}

	prevWave := s.spawner.Wave

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
		s.onWaveTransition(prevWave)
	}

	// 波次公告动画更新
	s.waveAnnounce.Update(gameDT)
	s.wavePanelState.Update(gameDT, s.wavePanelOpen)

	// 2. 敌人状态效果（减速、流血等）
	pipeline.TickEnemyStatusEffects(s.enemies, gameDT, func(e *enemy.Enemy, dmg float64) {
		render.SpawnDamageText(e.X, e.Y-10, dmg, false)
	})

	// 3. 敌人移动（到达终点扣生命）
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return // dying enemies don't move
		}
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, gameDT) {
			s.lives--
			s.enemies.KillImmediate(e) // leaked enemies vanish instantly, no dying anim
			fx := s.postPipeline.Effects
			fx.HitTintR, fx.HitTintG, fx.HitTintB = 1.0, 0.1, 0.1 // red flash on leak
			fx.TriggerHitFlash(0.15)
			s.bus.Emit(event.EvtEnemyLeaked, event.EnemyLeakedPayload{})
		}
	})

	// 3.5. Tick dying enemies (shrink+fade animation countdown)
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			e.DyingTimer -= gameDT
			if e.DyingTimer <= 0 {
				s.enemies.FinishDying(e)
			}
		}
	})

	// 3.6. 敌人行为 tick（治疗/隐身/旗手光环/回血）
	behaviorEvents := enemy.TickBehaviors(s.enemies, gameDT)
	if len(behaviorEvents.Heals) > 0 {
		s.audioMgr.PlayThrottledAt(gameAudio.SFXMedicHeal, 500, gameAudio.VolHit)
	}
	for _, rev := range behaviorEvents.Reveals {
		_ = rev
		s.audioMgr.PlaySafeAt(gameAudio.SFXStealthReveal, gameAudio.VolKill)
	}
	if behaviorEvents.Regens > 0 {
		s.audioMgr.PlayThrottledAt(gameAudio.SFXRegenTick, 2000, gameAudio.VolHit*0.5)
	}

	// 4. 战灵行为（未选择前跳过）
	if s.wardenReady && s.wardenUnit != nil {
		s.wardenUnit.Tick(&warden.TickContext{
			Enemies:     s.enemies,
			Towers:      s.towers,
			Projectiles: s.projectiles,
			DT:          gameDT,
			OnKill: func(e *enemy.Enemy) {
				s.audioMgr.PlaySafeAt(gameAudio.SFXEnemyDeath, gameAudio.VolKill)
				s.emitKill(e.Boss, "warden")
			},
			OnFire: func() {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXWardenFire, 100, gameAudio.VolWarden)
			},
			OnSpecial: func() {
				sfx := wardenSpecialSFX(s.wardenType)
				s.audioMgr.PlayThrottledAt(sfx, 200, gameAudio.VolWarden)
			},
			OnDamage: func(x, y, dmg float64, crit bool) {
				render.SpawnDamageText(x, y, dmg, crit)
			},
		})
	}

	// 5.5. 塔建造/出售动画 tick
	s.towers.Each(func(t *tower.Tower) {
		if t.BuildAnim > 0 {
			t.BuildAnim -= gameDT
			if t.BuildAnim < 0 {
				t.BuildAnim = 0
			}
		}
		if t.SellAnim > 0 {
			t.SellAnim -= gameDT
			if t.SellAnim <= 0 {
				s.towers.Remove(t)
				render.InvalidateMapCache()
			}
		}
	})

	// 5.6. 持续粒子特效：火焰塔/冰冻塔在有目标时发射元素粒子
	if s.frame%6 == 0 && game.Settings().MaxParticles > 100 {
		s.towers.Each(func(t *tower.Tower) {
			if t.Selling || t.Target == nil {
				return
			}
			switch t.AttackStyleID {
			case tower.StyleSpinAoE:
				particle.EmitFireParticles(s.particlePool, t.X, t.Y-4, 1)
			case tower.StyleScatter:
				particle.EmitIceParticles(s.particlePool, t.X, t.Y-4, 1)
			}
		})
	}

	// 6. 能力 tick（重置属性 + 光环 buff + 区域效果 + 经济产出）
	// 必须在索敌射击之前执行，确保 Range 等属性是本帧最新值
	abilityGold := pipeline.TickTowerAbilities(s.towers, s.enemies, gameDT)
	s.gold += abilityGold
	s.gameStats.GoldEarned += abilityGold

	// 6.6. 收集光源（优先级：路径端点 > Boss > 战灵 > 塔）
	s.postPipeline.Lighting.Clear()
	animTime := float64(s.frame) / 60.0
	lightCap := game.Settings().MaxLights
	lightCount := 0

	// Priority 1: path entrance (red breathing) + exit (blue breathing)
	if len(s.gameMap.Waypoints) > 0 && lightCount < lightCap {
		first := s.gameMap.Waypoints[0]
		last := s.gameMap.Waypoints[len(s.gameMap.Waypoints)-1]
		breathIntensity := 0.3 + 0.1*math.Sin(animTime*2)
		s.postPipeline.Lighting.AddLight(postprocess.PointLight{
			X: first.X, Y: first.Y,
			Color:     color.RGBA{R: 255, G: 60, B: 60, A: 255},
			Radius:    60,
			Intensity: breathIntensity,
		})
		lightCount++
		if lightCount < lightCap {
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{
				X: last.X, Y: last.Y,
				Color:     color.RGBA{R: 60, G: 120, B: 255, A: 255},
				Radius:    60,
				Intensity: breathIntensity,
			})
			lightCount++
		}
	}

	// Priority 2: boss enemy lights (first boss found)
	if lightCount < lightCap {
		s.enemies.Each(func(e *enemy.Enemy) {
			if lightCount >= lightCap {
				return
			}
			if e.Boss && !e.IsDying() {
				s.postPipeline.Lighting.AddLight(postprocess.PointLight{
					X: e.X, Y: e.Y,
					Color:     color.RGBA{R: 255, G: 160, B: 40, A: 255},
					Radius:    50,
					Intensity: 0.5,
				})
				lightCount++
			}
		})
	}

	// Priority 3: warden light
	if lightCount < lightCap && s.wardenReady && s.wardenUnit != nil {
		if base := s.wardenUnit.BaseState(); base != nil {
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{
				X: base.X, Y: base.Y,
				Color:     color.RGBA{R: 180, G: 140, B: 255, A: 255},
				Radius:    70,
				Intensity: 0.4,
			})
			lightCount++
		}
	}

	// Priority 4: towers fill remaining slots
	s.towers.Each(func(t *tower.Tower) {
		if lightCount >= postprocess.MaxLights || lightCount >= lightCap {
			return
		}
		s.postPipeline.Lighting.AddLight(postprocess.PointLight{
			X: t.X, Y: t.Y,
			Color:     towerLightColor(t.AttackStyleID),
			Radius:    t.Range * 0.6,
			Intensity: 0.4,
		})
		lightCount++
	})

	// CC 效果音效回调（塔战斗 + 弹射物共用）
	onCC := func(x, y float64, ccType string) {
		switch ccType {
		case "slow":
			s.audioMgr.PlayThrottledAt(gameAudio.SFXSlowApply, 120, gameAudio.VolHit)
		case "freeze":
			s.audioMgr.PlayThrottledAt(gameAudio.SFXFreezeHit, 120, gameAudio.VolHit)
		case "stun":
			s.audioMgr.PlayThrottledAt(gameAudio.SFXStunImpact, 150, gameAudio.VolHit)
		case "burn":
			s.audioMgr.PlayThrottledAt(gameAudio.SFXBurnIgnite, 200, gameAudio.VolHit)
		case "root":
			s.audioMgr.PlayThrottledAt(gameAudio.SFXRootApply, 150, gameAudio.VolHit)
		}
	}

	// 7. 塔索敌射击（按攻击方式分发）
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, s.beams, gameDT, func(t *tower.Tower, style string) {
		s.audioMgr.PlayThrottledAt(gameAudio.FireSFXForStyle(style), 100, gameAudio.VolFire)
		// Muzzle flash particles toward target (or spin angle for AoE)
		if t.Target != nil {
			angle := math.Atan2(t.Target.Y-t.Y, t.Target.X-t.X)
			particle.EmitMuzzleFlash(s.particlePool, t.X, t.Y, angle)
		} else if style == tower.StyleSpinAoE {
			particle.EmitMuzzleFlash(s.particlePool, t.X, t.Y, t.SpinAngle)
		}
	}, func(e *enemy.Enemy, damage float64, killed bool, _ string, crit bool) {
		// 直接攻击方式（laser/beam/spin_aoe等）的伤害飘字
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, crit)
		}
		if crit {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXCritHit, 150, gameAudio.VolHit)
		}
	}, onCC)

	// 7. 弹射物移动 + 光束衰减
	s.projectiles.Update(gameDT)
	s.beams.Update(gameDT)

	// 8. 弹射物命中检测（含能力触发）
	kills := pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers, func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool) {
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, crit)
			if e.HitFlash < 0.06 {
				e.HitFlash = 0.12
			}
			// 元素类型化命中特效
			render.SpawnTypedImpact(e.X, e.Y, attackStyle)
			// 元素粒子
			switch attackStyle {
			case "scatter":
				particle.EmitIceParticles(s.particlePool, e.X, e.Y, 2)
				s.postPipeline.Effects.TriggerRipple(e.X, e.Y, 6.0)
			case "spin_aoe":
				particle.EmitFireParticles(s.particlePool, e.X, e.Y, 2)
			case "charge":
				particle.EmitElectricSparks(s.particlePool, e.X, e.Y, 4)
			}
		}
		if killed {
			particle.EmitDeathBurst(s.particlePool, e.X, e.Y)
			particle.EmitGoldCollect(s.particlePool, e.X, e.Y)
			// 分裂体死亡音效（子体已由 Pool.Kill 自动生成）
			if e.Behavior == "splitter" && e.SplitCount > 0 {
				s.audioMgr.PlaySafeAt(gameAudio.SFXSplitPop, gameAudio.VolKill)
			}
			// Multi-kill tracker
			s.multiKillCount++
			s.multiKillTimer = 1.5
			if s.multiKillCount > s.gameStats.MaxKillStreak {
				s.gameStats.MaxKillStreak = s.multiKillCount
			}
			if s.multiKillCount > s.achieveTracker.SessionMaxStreak {
				s.achieveTracker.SessionMaxStreak = s.multiKillCount
			}
			if s.multiKillCount >= 20 {
				if s.achieveTracker.Unlock("killstreak_20") {
					hud.ShowToast("成就解锁: 连杀达人")
				}
			}
			if s.multiKillCount == 5 {
				render.SpawnText(float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-30,
					"连杀 x5", color.RGBA{255, 200, 50, 255}, 16, 1.5)
			} else if s.multiKillCount == 10 {
				render.SpawnText(float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-30,
					"超级连杀 x10", color.RGBA{255, 100, 50, 255}, 18, 2.0)
			}
			if e.Boss {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXEnemyDeathBoss, 50, gameAudio.VolKill)
				s.postPipeline.Effects.TriggerHitStop(3)
			} else {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXEnemyDeath, 50, gameAudio.VolKill)
			}
			s.emitKill(e.Boss, "projectile") // 统一击杀事件：kills/gold/session/tutorial/warden
		} else {
			// 命中音效：per-sound 节流，优先按敌人状态区分
			switch {
			case e.Boss:
				s.audioMgr.PlayThrottledAt(gameAudio.SFXHitHeavy, 60, gameAudio.VolHit)
			default:
				s.audioMgr.PlayThrottledAt(gameAudio.HitSFXForStyle(attackStyle), 60, gameAudio.VolHit)
			}
		}
		if crit {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXCritHit, 150, gameAudio.VolHit)
		}
	}, onCC)
	// 击杀统计/金币/session/tutorial/warden 由 emitKill → Bus 订阅者统一处理
	_ = kills

	// 8.5. Bleed drip particles for bleeding enemies
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		if e.BleedTimer > 0 && rand.Float64() < 0.15 { // ~9 particles/sec at 60fps
			particle.EmitBleedDrip(s.particlePool, e.X, e.Y, e.Radius)
		}
	})

	// 9. VFX 更新（浮动文本 + 冲击 + 粒子 + 屏幕震动）
	render.UpdateFloatTexts(gameDT)
	render.UpdateImpactVFX(gameDT)
	s.particlePool.Update(gameDT)
	render.UpdateShake(gameDT)

	// Multi-kill timer decay
	if s.multiKillTimer > 0 {
		s.multiKillTimer -= gameDT
		if s.multiKillTimer <= 0 {
			s.multiKillCount = 0
		}
	}

	// Ambient environment particles (skip on Low quality)
	if game.Settings().MaxParticles >= 1024 {
		s.ambientTimer += gameDT
		if s.ambientTimer >= 1.0 {
			s.ambientTimer -= 1.0
			particle.EmitAmbient(s.particlePool, float64(game.ScreenWidth), float64(game.ScreenHeight))
		}
	}

	// 10. 波次完成奖励 + 事件触发（由 onWaveTransition 统一处理，
	// 此处仅处理 spawner.Update 触发的波次变化；手动开波的变化在 tryStartWave 中处理）

	// 10.5. 后置安全网：清除本帧内被 abilities/skills/combat 击杀但尚未 Kill 的敌人
	// 前置安全网(step 2)只能处理上一帧残留，本帧新产生的 HP<=0 敌人需要在胜负判定前处理
	pipeline.TickEnemyStatusEffects(s.enemies, 0, nil) // dt=0 不触发 DoT，仅做 HP<=0 检查

	// 11. 胜负判定（委托给游戏模式）
	ctx := s.buildModeCtx()
	if s.session.CheckEndConditions(ctx) && s.state == statePlaying {
		if s.session.Status == gamemode.StatusVictory {
			s.state = stateVictory
			s.audioMgr.StopBGM()
			s.audioMgr.PlaySafeAt(gameAudio.SFXVictory, gameAudio.VolWave)
			s.progressMgr.RecordGameResult(s.modeID, s.gameMap.Config.ID, s.kills, true)
			if s.tutorial.IsComplete() {
				s.progressMgr.SetTutorialDone()
			}
			s.checkVictoryAchievements()
		} else if s.session.Status == gamemode.StatusDefeat {
			s.state = stateDefeat
			s.audioMgr.StopBGM()
			s.audioMgr.PlaySafeAt(gameAudio.SFXDefeat, gameAudio.VolWave)
			s.progressMgr.RecordGameResult(s.modeID, s.gameMap.Config.ID, s.kills, false)
		}
		// 清除覆盖层状态，防止 ChoicePanel/暂停菜单遮挡结算画面
		s.imode = modeIdle
		s.selectedTower = nil
		if s.choicePanel != nil {
			s.choicePanel.Close()
		}
	}

	// AutoPlay 决策钩子
	s.runAutoPlayFrame()
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

// readScreenPixels 在 Draw goroutine 中同步读取屏幕像素，返回 NRGBA image。
// 必须在 Draw 内同步调用，因为 Ebitengine 的 screen 在 Draw 返回后立即被清空/重用。
func readScreenPixels(screen *ebiten.Image) *image.NRGBA {
	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 1 || h <= 1 {
		return nil // 无头模式窗口太小，跳过截图
	}
	pixels := make([]byte, w*h*4)
	screen.ReadPixels(pixels)
	// 检查像素数据是否全零（GPU 尚未渲染）
	allZero := true
	for i := 0; i < len(pixels) && i < 1024; i++ {
		if pixels[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		log.Printf("screenshot skipped: pixel data is all zero (%dx%d)", w, h)
		return nil
	}
	return &image.NRGBA{
		Pix:    pixels,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}
}

// screenshotWG 追踪所有异步截图 goroutine，确保进程退出前全部完成。
var screenshotWG sync.WaitGroup

// WaitScreenshots 等待所有异步截图完成。在进程退出前调用。
func WaitScreenshots() {
	screenshotWG.Wait()
}

// saveImageAsync 异步编码并保存 PNG（不涉及 GPU 操作，可安全在 goroutine 中执行）。
func saveImageAsync(img *image.NRGBA, path string) {
	screenshotWG.Add(1)
	go func() {
		defer screenshotWG.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("screenshot panic (skipped): %v", r)
			}
		}()
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0o755)
		}
		f, err := os.Create(path)
		if err != nil {
			log.Printf("screenshot create error: %v", err)
			return
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			os.Remove(path) // 删除残缺 PNG
			log.Printf("screenshot encode error: %v", err)
			return
		}
		f.Close()
	}()
}

// worldBuffer 大地图相机偏移用的离屏缓冲（懒初始化）。
var worldBuffer *ebiten.Image

func (s *StageScene) Draw(screen *ebiten.Image) {
	s.perfTracker.BeginDraw()
	defer s.perfTracker.EndDraw()

	// AutoPlay 无头优化：没有截图请求时跳过全部渲染，GPU 开销≈0
	if s.autoPlayer != nil {
		fname := s.autoPlayer.ScreenshotRequested()
		if fname == "" {
			return // 跳过渲染
		}
		// 有截图请求：执行一次完整渲染 → 同步读像素 → 异步保存 PNG
		s.drawFullScene(screen)
		if img := readScreenPixels(screen); img != nil {
			saveImageAsync(img, fname)
		}
		return
	}

	s.drawFullScene(screen)

	// F12 截图：渲染完成后读取像素并异步保存
	if s.screenshotPending {
		s.screenshotPending = false
		if img := readScreenPixels(screen); img != nil {
			fname := filepath.Join("docs", "autotest", "pic",
				fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405")))
			saveImageAsync(img, fname)
			hud.ShowToast("截图已保存")
			log.Printf("screenshot saved: %s", fname)
		}
	}
}

// drawFullScene 执行完整的场景渲染（含屏幕震动）。
func (s *StageScene) drawFullScene(screen *ebiten.Image) {
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
	animTime := float64(s.frame) / 60.0
	inBuildMode := s.imode == modeBuildMenu || s.imode == modeBuildPlace
	render.DrawMap(worldTarget, s.gameMap, render.GlobalFont(), animTime, func(row, col int) bool {
		return s.towers.At(row, col) != nil
	}, inBuildMode)
	render.DrawParallaxBG(worldTarget, animTime, s.gameMap.Width()+s.gameMap.OffsetX*2, s.gameMap.Height()+s.gameMap.OffsetY*2)

	// 塔（优先 SVG 渲染，回退到彩色方块）
	s.towerRenderer.DrawTowers(worldTarget, s.towers, s.selectedTower, animTime)

	// 被 buff 的塔显示强化特效（五角星芒）
	s.towers.Each(func(t *tower.Tower) {
		if len(t.Buffs) > 0 {
			render.DrawTowerBuffEffect(worldTarget, t, animTime)
		}
	})

	// 能力升级指示器（塔上方脉冲金色菱形）
	s.drawUpgradeIndicators(worldTarget, animTime)

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
		s.wardenRenderer.DrawWarden(worldTarget, s.wardenUnit, animTime)
		// 战灵面板展开时显示攻击距离圈
		if s.wardenPanelOpen {
			if base := s.wardenUnit.BaseState(); base != nil && base.Range > 0 {
				draw.DashedCircle(worldTarget, float32(base.X), float32(base.Y),
					float32(base.Range), 1, 6, 4, color.RGBA{R: 180, G: 140, B: 255, A: 100})
			}
		}
	}

	// 道具拖拽目标高亮
	if s.dragItemActive && s.dragHoverTower != nil {
		t := s.dragHoverTower
		draw.CircleOutline(worldTarget, float32(t.X), float32(t.Y), 22, 2,
			color.RGBA{R: 100, G: 255, B: 100, A: 200})
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

	// HUD：底部动作栏
	hud.DrawActionBar(screen, hud.ActionBarData{
		BuildActive: s.imode == modeBuildMenu || s.imode == modeBuildPlace,
		ItemActive:  s.imode == modeItemPanel || s.imode == modeItemDrag,
		ItemTotal:   s.inventory.TotalCount(),
	})

	// HUD：底部建塔菜单
	hud.DrawBuildMenu(screen, s.buildBuildMenuData())

	// HUD：道具面板
	hud.DrawItemPanel(screen, s.buildItemPanelData())

	// HUD：底部中央面板（塔信息 和 战灵信息 互斥）
	if s.selectedTower != nil {
		// 塔选中时显示塔信息面板（底部中央）
		sellValue := s.econ.SellRefund(s.selectedTower.Cost)
		vm := BuildInfoPanelVM(s.selectedTower, sellValue, s.wavesCleared)
		hud.DrawInfoPanel(screen, vm)
		// Hover 在面板上时显示升级详情浮窗
		mx, my := draw.CursorPos()
		hud.DrawInfoPanelHoverTooltip(screen, s.selectedTower != nil, float32(mx), float32(my))
	} else if s.wardenPanelOpen && s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		// 无塔选中且战灵面板展开时显示战灵面板（底部中央）
		hud.DrawWardenPanel(screen, s.buildWardenPanelData())
	}

	// 左下角：抽屉式波次面板（含 tab handle）
	hud.DrawWavePanel(screen, s.buildWavePanelData(), &s.wavePanelState)

	// 右下角收起按钮（战灵，选择后才显示）
	if s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		hud.DrawToggleButton(screen, false, s.wardenPanelOpen, "⚡")
	}

	// 教程覆盖层
	if step := s.tutorial.CurrentStep(); step != nil {
		hud.DrawTutorialOverlay(screen, hud.TutorialVM{
			Visible:        true,
			Message:        step.Message,
			Step:           s.tutorial.StepIndex() + 1,
			Total:          s.tutorial.StepCount(),
			ClickToAdvance: step.Event == "",
		})
	}

	// 调试面板（测试模式）
	if s.testMode && s.debugPanelOpen {
		hud.DrawDebugPanel(screen, hud.DebugPanelData{Actions: s.debugActions()})
	}

	// 调试覆盖层：实体统计 + 性能统计（F2 切换）
	s.debugOverlay.DrawHUD(screen,
		s.towers.Count, s.enemies.Count,
		s.beams.Count(), s.projectiles.Count)
	s.debugOverlay.DrawPerf(screen, hud.PerfVM{
		FPS: s.perfTracker.FPS, AvgUpdateMs: s.perfTracker.AvgUpdateMs,
		AvgDrawMs: s.perfTracker.AvgDrawMs, P99UpdateMs: s.perfTracker.P99UpdateMs,
		P99DrawMs: s.perfTracker.P99DrawMs, GCCount: s.perfTracker.GCCount,
		HeapMB: s.perfTracker.HeapMB,
	})

	// 小地图
	hud.DrawMinimap(screen, s.buildMinimapVM())

	// 波次公告动画（slide-in/hold/slide-out）
	s.waveAnnounce.Draw(screen)

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

	// 道具拖拽指示器
	if s.dragItemActive {
		mx, my := s.gesture.CursorPos()
		def := item.Defs[s.dragItemKind]
		hud.DrawDragItem(screen, float32(mx), float32(my), def.Color, def.Name, int(s.dragItemKind))
	}

	// Toast 通知
	hud.DrawToast(screen)

	// 战灵选择覆盖层
	if s.wardenOverlay != nil {
		s.wardenOverlay.Draw(screen)
	}

	// 能力选择覆盖层
	if s.choicePanel != nil {
		s.choicePanel.Draw(screen)
	}

	// 胜负覆盖层
	if s.state == stateVictory || s.state == stateDefeat {
		// 半透明遮罩
		draw.RoundRect(screen, 0, 0, float32(game.ScreenWidth), float32(game.ScreenHeight), 0, theme.HUDGameOverlay)
		if fm := render.GlobalFont(); fm != nil {
			if s.state == stateVictory {
				fm.DrawCenteredText(screen, "胜利!", float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, 52, theme.HUDVictoryColor)
			} else {
				fm.DrawCenteredText(screen, "失败!", float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, 52, theme.HUDDefeatColor)
			}
			fm.DrawCenteredText(screen, fmt.Sprintf("击杀: %d  点击继续", s.kills), float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2+30, theme.FontH2, theme.TextMuted)
		}
	}
}

// buildBuildMenuData 构建建造菜单展示数据。
func (s *StageScene) buildBuildMenuData() hud.BuildMenuData {
	buildableCount := len(s.towerDefs)
	cards := make([]hud.BuildCardVM, 0, buildableCount+10)

	// Buildable tower cards
	for _, def := range s.towerDefs {
		roleTag, roleClr := towerRoleTags(def)
		cards = append(cards, hud.BuildCardVM{
			Key: def.Key, Label: def.Label, Cost: def.Cost,
			Damage: def.Damage, AttackSpeed: def.AttackSpeed, Range: def.Range,
			RoleTag: roleTag, RoleColor: roleClr,
			TypeIcon: towerTypeIcon(def.Key),
			Sprite:   s.towerRenderer.GetSprite("sentinel"),
			Buildable: true,
		})
	}

	// Attack style variant cards (display only)
	attackAbils := tower.AbilitiesForCategory(config.AbilityCatAttack)
	sort.Slice(attackAbils, func(i, j int) bool {
		return attackAbils[i].Type < attackAbils[j].Type
	})
	for _, ab := range attackAbils {
		sprKey := tower.AbilitySpriteKey(ab.Type)
		cards = append(cards, hud.BuildCardVM{
			Key:         ab.Type,
			Label:       tower.SpriteLabelFor(sprKey),
			RoleTag:     ab.Label,
			RoleColor:   color.RGBA{R: 140, G: 160, B: 200, A: 180},
			TypeIcon:    ab.Icon,
			Sprite:      s.towerRenderer.GetSprite(sprKey),
			Buildable:   false,
			AbilityDesc: attackStyleDesc(ab.Type),
		})
	}

	return hud.BuildMenuData{
		Cards:          cards,
		BuildableCount: buildableCount,
		SelectedIdx:    s.selectedDef,
		Gold:           s.gold,
		HoverIdx:       s.buildHoverIdx,
		Visible:        s.imode == modeBuildMenu,
	}
}

// buildMenuTotalCards returns total card count (buildable + attack variants) for layout.
func (s *StageScene) buildMenuTotalCards() int {
	return len(s.towerDefs) + len(tower.AbilitiesForCategory(config.AbilityCatAttack))
}

func (s *StageScene) buildItemPanelData() hud.ItemPanelData {
	if s.imode != modeItemPanel && s.imode != modeItemDrag {
		return hud.ItemPanelData{Visible: false}
	}
	return hud.ItemPanelData{Cards: s.buildItemPanelCards(), Visible: true}
}

func (s *StageScene) buildItemPanelCards() []hud.ItemCardVM {
	cards := make([]hud.ItemCardVM, item.KindCount)
	for _, k := range item.AllKinds {
		cards[k] = hud.ItemCardVM{
			Name:  item.Defs[k].Name,
			Count: s.inventory.Count(k),
			Color: item.Defs[k].Color,
			Kind:  int(k),
		}
	}
	return cards
}

// attackStyleDesc 返回攻击方式的纯功能描述（不含数值）。
func attackStyleDesc(abilType string) string {
	m := map[string]string{
		"enhance":     "一次性全面提升基础属性",
		"scatter":     "发射多颗弹丸，锥形散布",
		"wideBeam":    "宽光束穿透所有敌人",
		"spinAoe":     "旋转范围伤害，内圈额外加伤",
		"bounce":      "弹射多个敌人",
		"splash":      "命中后对周围敌人造成溅射伤害",
		"multiTarget": "同时攻击多个目标",
		"radial":      "360度发射穿透弹，1.2倍射程",
	}
	if d, ok := m[abilType]; ok {
		return d
	}
	return ""
}

// towerRoleTags 返回塔的角色标签和颜色。
func towerRoleTags(def tower.TowerDef) (string, color.RGBA) {
	for _, ab := range def.Abilities {
		switch ab {
		case "onHitSlow":
			return "控制·减速", color.RGBA{R: 80, G: 180, B: 220, A: 255}
		case "stun":
			return "输出·眩晕", color.RGBA{R: 180, G: 120, B: 220, A: 255}
		case "bounce":
			return "输出·连锁", color.RGBA{R: 220, G: 180, B: 80, A: 255}
		case "splash":
			return "输出·溅射", color.RGBA{R: 220, G: 120, B: 80, A: 255}
		case "bleedDot", "burn":
			return "输出·持续", color.RGBA{R: 220, G: 80, B: 80, A: 255}
		case "executionBonus", "percentHpDamage":
			return "输出·斩杀", color.RGBA{R: 180, G: 60, B: 60, A: 255}
		case "damageUpAura", "attackSpeedAura":
			return "辅助·光环", color.RGBA{R: 80, G: 200, B: 120, A: 255}
		case "poisonZone", "silenceZone":
			return "控制·区域", color.RGBA{R: 100, G: 160, B: 200, A: 255}
		case "goldOnKill", "goldPassive":
			return "经济", color.RGBA{R: 220, G: 200, B: 80, A: 255}
		}
	}
	return "输出", color.RGBA{R: 200, G: 200, B: 200, A: 200}
}

// towerTypeIcon 返回塔类型图标名。
func towerTypeIcon(key string) string {
	switch key {
	case "freeze":
		return "tower-freeze"
	case "electric":
		return "tower-electric"
	case "hunter":
		return "tower-hunter"
	case "laser":
		return "tower-laser"
	default:
		return ""
	}
}

// drawUpgradeIndicators 在有待选能力的塔上方绘制脉冲金色菱形指示器。
func (s *StageScene) drawUpgradeIndicators(target *ebiten.Image, animTime float64) {
	pulse := float32(0.6 + 0.4*math.Sin(animTime*5))  // alpha 脉冲
	scale := float32(1.0 + 0.15*math.Sin(animTime*5)) // 尺寸脉冲
	s.towers.Each(func(t *tower.Tower) {
		if t.HasPendingUpgrade(s.wavesCleared) {
			a := uint8(230 * pulse)
			r := float32(7) * scale
			// 外层辉光
			draw.Diamond(target, float32(t.X), float32(t.Y-24), r+2, 1.0,
				color.RGBA{R: 250, G: 200, B: 50, A: a / 3})
			// 内层实体
			draw.Diamond(target, float32(t.X), float32(t.Y-24), r, 1.8,
				color.RGBA{R: 250, G: 200, B: 50, A: a})
		}
	})
}

// buildMinimapVM 构建小地图展示数据。
func (s *StageScene) buildMinimapVM() hud.MinimapVM {
	vm := hud.MinimapVM{}
	// 路径点
	for _, wp := range s.gameMap.Waypoints {
		vm.PathPoints = append(vm.PathPoints, hud.MinimapPoint{X: wp.X, Y: wp.Y})
	}
	// 塔
	s.towers.Each(func(t *tower.Tower) {
		vm.Towers = append(vm.Towers, hud.MinimapPoint{X: t.X, Y: t.Y})
	})
	// 敌人
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		vm.Enemies = append(vm.Enemies, hud.MinimapDot{X: e.X, Y: e.Y, IsBoss: e.Boss})
	})
	// 战灵
	if s.wardenReady && s.wardenUnit != nil {
		if wb := s.wardenUnit.BaseState(); wb != nil {
			vm.WardenX, vm.WardenY = wb.X, wb.Y
		}
	}
	return vm
}

// buildWardenPanelData 根据当前战灵状态和配置构建面板显示数据。
func (s *StageScene) buildWavePanelData() hud.WavePanelData {
	d := hud.WavePanelData{
		WaveNum:    s.spawner.Wave,
		MaxWaves:   s.spawner.MaxWaves,
		EnemyCount: s.enemies.Count,
		AllDone:    s.spawner.AllDone,
	}
	entries, count, boss := s.spawner.NextWavePreview()
	d.NextWaveCount = count
	d.NextWaveBoss = boss
	for _, e := range entries {
		d.NextWaveTypes = append(d.NextWaveTypes, hud.WaveTypeEntry{
			Label: e.Label,
			Count: e.Count,
		})
	}
	return d
}

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

// archetypeBehavior 从原型名推导行为类型。
// 原型配置中有对应字段的优先使用配置值，否则按名称映射。
var archetypeBehavior = map[string]string{
	"healer":      "healer",
	"medic":       "healer",
	"stealth":     "stealth",
	"splitter":    "splitter",
	"buffer":      "buffer",
	"regenerator": "regenerator",
	"troll":       "regenerator",
}

// convertArchetypesToSpawnConfigs 将 config.EnemyArchetype 转换为 enemy.SpawnConfig。
// 使 enemy 包不依赖 config 包。
func convertArchetypesToSpawnConfigs(archetypes map[string]*config.EnemyArchetype) map[string]*enemy.SpawnConfig {
	result := make(map[string]*enemy.SpawnConfig, len(archetypes))
	for key, a := range archetypes {
		sc := &enemy.SpawnConfig{
			Label:           a.Label,
			HpScale:         a.HPScale,
			SpeedScale:      a.SpeedScale,
			Radius:          a.Radius,
			Boss:            a.Boss,
			StealthDuration: a.StealthDuration,
			SplitCount:      a.SplitCount,
			SplitScale:      0.3, // 默认子体血量 30%
			HealScale:       a.HealScale,
			HealRadius:      a.HealRadius,
			HealInterval:    a.HealInterval,
			AuraRange:       a.AuraRange,
			AuraSpeedUp:     a.AuraSpeedUp,
		}
		// 根据原型名推导行为类型
		if b, ok := archetypeBehavior[key]; ok {
			sc.Behavior = b
		}
		result[key] = sc
	}
	return result
}

// ── 战灵选择逻辑 ────────────────────────────────────

// tryStartWave 尝试开波。若战灵未选择则先弹出战灵选择面板。
// onWaveTransition 处理波次变化（prevWave → 当前 Wave）。
// 包括：WaveStarted 事件 + WaveCleared 奖励/事件。
// 由 spawner.Update 自动开波和 tryStartWave 手动开波两条路径统一调用。
func (s *StageScene) onWaveTransition(prevWave int) {
	// 新波开始事件
	s.waveLivesSnapshot = s.lives
	s.bus.Emit(event.EvtWaveStarted, event.WaveStartedPayload{
		Wave: s.spawner.Wave, IsBoss: s.spawner.Wave%5 == 0,
	})

	// 前一波清完奖励（prevWave=0 时无前波）
	if prevWave > 0 {
		ctx := s.buildModeCtx()
		result := s.session.OnWaveCleared(prevWave, ctx)
		interest := s.econ.InterestGold(s.gold)
		totalBonus := result.BonusGold + result.PerfectBonus + interest
		s.gold += totalBonus
		s.gameStats.GoldEarned += totalBonus
		perfect := s.lives == s.waveLivesSnapshot && result.PerfectBonus > 0
		if perfect {
			render.SpawnText(float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-50,
				"完美!", color.RGBA{255, 215, 0, 255}, 20, 2.0)
		}
		msg := result.Message
		if interest > 0 {
			msg += fmt.Sprintf(" +$%d 利息", interest)
		}
		s.showNotify(msg)

		s.bus.Emit(event.EvtWaveCleared, event.WaveClearedPayload{
			Wave: prevWave, Perfect: perfect,
		})
	}
}

func (s *StageScene) tryStartWave() {
	if !s.wardenReady {
		s.showWardenSelect()
		return
	}
	prevWave := s.spawner.Wave
	s.spawner.StartNextWave()
	s.audioMgr.PlaySafeAt(gameAudio.SFXUIClick, gameAudio.VolUI)
	// 手动开波时 Wave 在 handleInput 阶段递增，updatePlaying 的 prevWave
	// 已经是新值，导致 EvtWaveCleared 不触发。这里补发。
	if s.spawner.Wave > prevWave && prevWave > 0 {
		s.onWaveTransition(prevWave)
	}
}

// showWardenSelect 弹出战灵选择覆盖层。
func (s *StageScene) showWardenSelect() {
	s.wardenOverlay.Show(GetWardenOptions(s.progressMgr), func(key string) {
		s.activateWarden(key)
		// 选完后立即开第一波
		prevWave := s.spawner.Wave
		s.spawner.StartNextWave()
		s.audioMgr.PlaySafeAt(gameAudio.SFXUIClick, gameAudio.VolUI)
		if s.spawner.Wave > prevWave {
			s.onWaveTransition(prevWave)
		}
	})
	s.selectedTower = nil // 进入战灵选择时清除塔选中状态
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
	// 设置地图边界，使战灵可达全域
	if base := s.wardenUnit.BaseState(); base != nil {
		base.MapWidth = s.gameMap.PixelWidth()
		base.MapHeight = s.gameMap.PixelHeight()
	}

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

// handleWardenSelection 已移至 stage_input.go。

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

// ─── AutoPlay 集成 ───

// SetAutoPlayer 注入自动对局驱动器。设为 nil 恢复手动模式。
func (s *StageScene) SetAutoPlayer(ap AutoPlayer) {
	s.autoPlayer = ap
}

// HasPendingScreenshot 检查 autoPlayer 是否有待截图请求（turbo 循环用）。
func (s *StageScene) HasPendingScreenshot() bool {
	return s.autoPlayer != nil && s.autoPlayer.HasPendingScreenshot()
}

// buildAutoPlaySnapshot 构建当前游戏状态快照。
func (s *StageScene) buildAutoPlaySnapshot() AutoPlaySnapshot {
	snap := AutoPlaySnapshot{
		Tick:            s.frame,
		Gold:            s.gold,
		Lives:           s.lives,
		Wave:            s.spawner.Wave,
		MaxWaves:        s.spawner.MaxWaves,
		WaveActive:      s.spawner.WaveActive,
		WardenReady:     s.wardenReady,
		InteractMode:    int(s.imode),
		GameOver:        s.state != statePlaying,
		Victory:         s.state == stateVictory,
		TotalKills:      s.kills,
		EnemyPoolCount:  s.enemies.Count,
		ProjectileCount: s.projectiles.Count,
		TowerCount:      s.towers.Count,
		MapPixelW:       s.gameMap.PixelWidth(),
		MapPixelH:       s.gameMap.PixelHeight(),
		GameSpeed:       s.gameSpeed,
		Telemetry:       tel.T.Snapshot(),
	}
	// 遥测：记录交互模式
	modeNames := []string{"idle", "buildMenu", "buildPlace", "towerSel", "spawnMenu", "spawnPlace", "event", "paused", "wardenSelect"}
	if int(s.imode) < len(modeNames) {
		tel.T.Record("imode", modeNames[s.imode])
	}

	// 战灵位置
	if s.wardenReady && s.wardenUnit != nil {
		if base := s.wardenUnit.BaseState(); base != nil {
			snap.WardenX = base.X
			snap.WardenY = base.Y
		}
	}

	// 敌人快照
	s.enemies.Each(func(e *enemy.Enemy) {
		snap.Enemies = append(snap.Enemies, AutoPlayEnemy{
			ID: e.ID, X: e.X, Y: e.Y,
			HP: e.HP, MaxHP: e.MaxHP, Speed: e.Speed,
			Archetype: e.Archetype, Boss: e.Boss,
			Active: e.Active, Dying: e.IsDying(),
			IsSlowed: e.SlowTimer > 0, IsStunned: e.StunTimer > 0,
			IsBurning: e.BurnTimer > 0, IsBleeding: e.BleedTimer > 0,
			IsRooted: e.RootTimer > 0,
		})
	})

	// 已建塔快照
	s.towers.Each(func(t *tower.Tower) {
		str := 0
		if t.Strength != nil {
			str = int(t.Strength.Permanent)
		}
		snap.Towers = append(snap.Towers, AutoPlayTower{
			Key: t.Key, Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y, Damage: t.Damage,
			Range: t.Range, Cost: t.Cost, Strength: str,
			Abilities: t.Abilities,
			AttackStyle: string(t.AttackStyleID),
			HasTarget: t.Target != nil,
		})
	})

	// 可用建造位置
	gm := s.gameMap
	for row := 0; row < gm.Config.Rows; row++ {
		for col := 0; col < gm.Config.Cols; col++ {
			if gm.Config.Grid[row][col] == config.CellBuildable && s.towers.At(row, col) == nil {
				center := gm.CellCenter(row, col)
				snap.BuildCells = append(snap.BuildCells, AutoPlayCell{
					Row: row, Col: col, X: center.X, Y: center.Y,
				})
			}
		}
	}

	// 可用塔类型
	for i, d := range s.towerDefs {
		snap.TowerDefs = append(snap.TowerDefs, AutoPlayTowerDef{
			Key: d.Key, Cost: d.Cost, Range: d.Range,
			Damage: d.Damage, Index: i,
		})
	}

	return snap
}

// executeAutoPlayAction 执行一个自动操作指令。
func (s *StageScene) executeAutoPlayAction(a AutoPlayAction) {
	switch a.Type {
	case APActionBuild:
		// 设置选中塔类型
		for i, d := range s.towerDefs {
			if d.Key == a.TowerKey {
				s.selectedDef = i
				break
			}
		}
		center := s.gameMap.CellCenter(a.Row, a.Col)
		s.tryPlaceTower(center.X, center.Y)

	case APActionUpgrade:
		t := s.towers.At(a.Row, a.Col)
		if t != nil {
			s.selectedTower = t
			s.tryUpgradeTower()
		}

	case APActionSell:
		center := s.gameMap.CellCenter(a.Row, a.Col)
		s.trySellTower(center.X, center.Y)

	case APActionStartWave:
		if s.wardenReady {
			prevWave := s.spawner.Wave
			s.spawner.StartNextWave()
			if s.spawner.Wave > prevWave {
				s.onWaveTransition(prevWave)
			}
		}

	case APActionSelectWarden:
		s.activateWarden(a.WardenKey)
	}
}

// runAutoPlayFrame 在 updatePlaying 末尾调用，驱动自动对局逻辑。
func (s *StageScene) runAutoPlayFrame() {
	if s.autoPlayer == nil {
		return
	}
	snap := s.buildAutoPlaySnapshot()
	actions := s.autoPlayer.OnUpdate(snap)
	for _, a := range actions {
		s.executeAutoPlayAction(a)
	}
}
