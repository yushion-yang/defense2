// stage.go — 游戏主场景。
// 管理塔防核心循环：生成敌人、移动、塔战斗、弹射物、经济、胜负判定。
package scene

import (
	"encoding/json"
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
	"defense2/internal/core/gamemode"
	"defense2/internal/core/item"
	"defense2/internal/core/persistence"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	tel "defense2/internal/core/telemetry"
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
	switcher       Switcher                     // 场景切换器引用
	bus            *event.Bus                   // 事件总线（从 Switcher 获取）
	busSubscribed  bool                         // Bus 订阅是否已完成（延迟到首次 Update）
	session        *gamemode.Session            // 游戏模式会话
	modeID         string                       // 模式 ID（用于重玩）
	diffID         string                       // 难度 ID（用于重玩）
	frame          int                          // 当前帧计数
	state          stageState                   // 当前游戏状态（进行中/胜利/失败）
	gameMap        *gamemap.GameMap             // 运行时地图
	enemies        *enemy.Pool                  // 敌人对象池
	spawner        *enemy.Spawner               // 波次出怪管理器
	towers         *tower.Pool                  // 塔对象池
	projectiles    *projectile.Pool             // 弹射物对象池
	beams          *combat.BeamPool             // 光束视觉对象池
	econ           economy.Config               // 经济配置
	lives          int                          // 剩余生命值
	gold           int                          // 当前金币
	kills          int                          // 累计击杀数
	towerDefs      []tower.TowerDef             // 可建造的塔类型列表
	selectedDef    int                          // 当前选中的塔类型索引
	selectedTower  *tower.Tower                 // 点击选中的塔（显示信息面板+射程）
	towerRenderer  *render.TowerRenderer        // 塔 SVG 渲染器
	enemyRenderer  *render.EnemyRenderer        // 敌人 SVG 渲染器
	wardenRenderer *render.WardenRenderer       // 战灵精灵渲染器
	audioMgr       *gameAudio.Manager           // 音效管理器
	wardenUnit     *warden.Warden               // 战灵实体（选择前为 nil）
	wardenOverlay  *hud.WardenSelectOverlay     // 战灵选择覆盖层
	wardenReady    bool                         // 战灵已选择并激活
	tutorial       *tutorial.Tutorial           // 新手教程
	progressMgr    *persistence.ProgressManager // 持久化进度管理器
	lastWave       int                          // 上一帧的波次号
	wardenType     string                       // 战灵类型标识（用于重玩传递）
	wardenCfg      *config.WardenConfig         // 战灵配置（用于面板显示）
	// 道具系统
	inventory         *item.Inventory // 道具背包
	dragItemKind      item.Kind       // 当前拖拽的道具类型
	dragItemActive    bool            // 是否正在拖拽道具
	dragHoverTower    *tower.Tower    // 拖拽道具时悬停的目标塔
	itemPanelOpen     bool            // 道具面板是否打开
	gameSpeed         int             // 游戏速度倍率（1 或 2）
	imode             interactMode    // 交互状态机
	prePauseMode      interactMode    // 暂停前的交互模式（恢复用）
	buildHoverIdx     int             // 建塔面板鼠标悬停索引
	gesture           *input.Gesture  // 统一手势识别器
	waveLivesSnapshot int             // 波开始时的生命快照（用于完美波次检测）
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
	wavePanelOpen     bool               // 左下角波次面板是否展开
	wavePanelState    hud.WavePanelState // 抽屉动画状态
	wardenPanelOpen   bool               // 右下角战灵面板是否展开
	testMode          bool
	scenarioID        string
	enemyFilter       string
	manualWave        bool
	debugPanelOpen    bool
	debugShowRange    bool
	spawnMode         bool
	spawnMoving       bool   // true=造动怪（放在路径上行走），false=造静怪
	saveNaming        bool           // 场景命名输入中
	saveNameBuf       string         // 命名缓冲区
	hoveredEnemy      *enemy.Enemy   // 测试模式：鼠标悬浮的敌人
	spawnType         string
	spawnHoverIdx     int
	initOpts          StageOptions          // 保存原始配置（重新开始用）
	postPipeline      *postprocess.Pipeline // 后处理管线（bloom 等）
	particlePool      *particle.Pool        // GPU 粒子系统
	debugOverlay      *hud.DebugOverlay     // 调试覆盖层（F2 切换）
	perfTracker       *debug.PerfTracker    // 性能追踪器
	qualityAdaptive   *game.QualityAdaptive // 自适应画质调节器
	waveAnnounce      *hud.WaveAnnounce     // 波次开始公告动画
	ambientTimer      float64               // 环境粒子发射计时器（每秒一次）
	multiKillCount    int                   // 连续击杀计数
	multiKillTimer    float64               // 连杀窗口倒计时（1.5s 无击杀后重置）
	choicePanel       *hud.ChoicePanel      // 能力选择覆盖层
	autoPlayer        AutoPlayer            // 自动对局驱动（nil=手动模式）
	screenshotPending bool                  // F12 截图请求标志
	achieveTracker    *achievement.Tracker  // 成就追踪器
	gameStats         GameStats             // 详细游戏统计
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

	// 加载怪物能力配置表
	config.LoadEnemyAbilities()

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

	// bossRush 模式：每波都出 Boss
	if modeID == "bossRush" {
		spawner.BossEveryWave = true
	}

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
		towerDefs:       filterUnlockedTowers(loadTowerDefsOrFallback(), pm),
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

	// Restore saved scenario towers (custom JSON scenarios)
	if opts.ScenarioID != "" {
		if scenarios, err := config.LoadScenarios(); err == nil {
			for _, sd := range scenarios {
				if sd.ID == opts.ScenarioID {
					s.restoreScenario(sd)
					break
				}
			}
		}
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
// rewardScale 为敌人原型奖励倍率（如 tank=1.35, runner=0.72），0 或 1 表示无缩放。
func (s *StageScene) emitKill(isBoss bool, killerID string, rewardScale float64) {
	gold := s.econ.KillGold()
	if rewardScale > 0 && rewardScale != 1 {
		gold = int(float64(gold) * rewardScale)
	}
	s.bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{
		IsBoss:    isBoss,
		KillerID:  killerID,
		GoldValue: gold,
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

// finalizeGameStats 结算时最终化游戏统计数据。
// 从 session 和塔池填充剩余字段（击杀数、波次、Boss、最强塔等）。
func (s *StageScene) finalizeGameStats() GameStats {
	gs := s.gameStats
	gs.TotalKills = s.kills
	gs.TotalWaves = s.spawner.Wave
	gs.MaxWave = s.spawner.MaxWaves
	gs.BossKills = s.session.Stats.BossKills
	gs.TimePlayed = s.session.ElapsedTime

	// 查找击杀最多的塔
	s.towers.Each(func(t *tower.Tower) {
		if t.Kills > gs.BestTowerKills {
			gs.BestTowerKills = t.Kills
			gs.BestTowerKey = t.Key
			gs.BestTowerName = t.Label
		}
	})
	return gs
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

	// 场景命名输入（拦截所有其他输入）
	if s.saveNaming {
		s.updateSaveNaming()
		return nil
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
			stats := s.finalizeGameStats()
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
				Stats:        stats,
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
	if s.gold < cost {
		return false // 金币不足
	}

	center := gm.CellCenter(row, col)
	placed := s.towers.Place(row, col, center.X, center.Y, def)
	// 初始化战力系统（base/potential 已在 pool.Place 中从 TowerDef 设置）
	if placed != nil {
		// Strength 已在 pool.Place 中初始化，无需重复创建
		placed.BuildAnim = 0.3
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
	// ── 造怪 ──
	if inSpawn {
		actions = append(actions,
			hud.DebugAction{Label: "造怪", IsSection: true},
			hud.DebugAction{Label: "退出造怪", Action: func() {
				s.imode = modeIdle
				s.spawnMode = false
				s.spawnType = ""
			}},
		)
	} else {
		actions = append(actions,
			hud.DebugAction{Label: "造怪", IsSection: true},
			hud.DebugAction{Label: "造静怪", Action: func() {
				s.imode = modeSpawnMenu
				s.spawnMode = true
				s.spawnMoving = false
				s.spawnType = ""
			}},
			hud.DebugAction{Label: "造动怪", Action: func() {
				s.imode = modeSpawnMenu
				s.spawnMode = true
				s.spawnMoving = true
				s.spawnType = ""
			}},
		)
	}

	// ── 塔操作 ──
	actions = append(actions,
		hud.DebugAction{Label: "塔操作", IsSection: true},
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
	// ── 场景快照 ──
	actions = append(actions,
		hud.DebugAction{Label: "场景快照", IsSection: true},
		hud.DebugAction{Label: "Save Scenario", Action: func() {
			s.saveNaming = true
			s.saveNameBuf = ""
		}},
	)

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

// updateSaveNaming handles text input for scenario naming overlay.
func (s *StageScene) updateSaveNaming() {
	// Append typed characters
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if len(s.saveNameBuf) < 40 {
			s.saveNameBuf += string(ch)
		}
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.saveNameBuf) > 0 {
		// Remove last rune (handles multi-byte UTF-8)
		runes := []rune(s.saveNameBuf)
		s.saveNameBuf = string(runes[:len(runes)-1])
	}

	// Enter = confirm
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		name := strings.TrimSpace(s.saveNameBuf)
		if name == "" {
			name = fmt.Sprintf("Snapshot %s", time.Now().Format("15:04:05"))
		}
		s.saveNaming = false
		s.saveScenario(name)
	}

	// Escape = cancel
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.saveNaming = false
		hud.ShowToast("Save cancelled")
	}
}

// drawSaveNaming renders the scenario naming input overlay.
func (s *StageScene) drawSaveNaming(screen *ebiten.Image) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// Dim background
	draw.FilledRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{A: 160}, false)

	// Dialog box
	boxW, boxH := float32(400), float32(120)
	boxX := float32(sw)/2 - boxW/2
	boxY := float32(sh)/2 - boxH/2
	draw.RoundRect(screen, boxX, boxY, boxW, boxH, 12, color.RGBA{R: 30, G: 35, B: 50, A: 245})
	draw.StrokeRoundRect(screen, boxX, boxY, boxW, boxH, 12, 1.5, color.RGBA{R: 80, G: 120, B: 200, A: 200})

	// Title
	fm.DrawCenteredText(screen, "Save Scenario", sw/2, float64(boxY)+16, 16, color.RGBA{R: 220, G: 230, B: 255, A: 255})

	// Input field background
	fieldX := boxX + 20
	fieldY := boxY + 50
	fieldW := boxW - 40
	fieldH := float32(30)
	draw.RoundRect(screen, fieldX, fieldY, fieldW, fieldH, 6, color.RGBA{R: 15, G: 18, B: 30, A: 255})
	draw.StrokeRoundRect(screen, fieldX, fieldY, fieldW, fieldH, 6, 1, color.RGBA{R: 60, G: 80, B: 140, A: 200})

	// Text content with blinking cursor
	display := s.saveNameBuf
	if int(time.Now().UnixMilli()/500)%2 == 0 {
		display += "|"
	}
	if display == "|" {
		// Show placeholder when empty
		fm.DrawText(screen, "Enter scenario name...", float64(fieldX)+8, float64(fieldY)+7, 13, color.RGBA{R: 80, G: 90, B: 110, A: 200})
	} else {
		fm.DrawText(screen, display, float64(fieldX)+8, float64(fieldY)+7, 13, color.RGBA{R: 200, G: 210, B: 230, A: 255})
	}

	// Hint
	fm.DrawCenteredText(screen, "Enter: Save  |  Esc: Cancel", sw/2, float64(boxY+boxH)-14, 10, color.RGBA{R: 100, G: 110, B: 140, A: 200})
}

// enemyTooltipLine 带颜色的 tooltip 行。
type enemyTooltipLine struct {
	text string
	clr  color.RGBA
}

var (
	ttWhite  = color.RGBA{R: 220, G: 230, B: 245, A: 255}
	ttDim    = color.RGBA{R: 140, G: 150, B: 170, A: 220}
	ttHeader = color.RGBA{R: 100, G: 160, B: 220, A: 255}
	ttIce    = color.RGBA{R: 100, G: 180, B: 255, A: 255}
	ttYellow = color.RGBA{R: 255, G: 220, B: 80, A: 255}
	ttRed    = color.RGBA{R: 255, G: 100, B: 80, A: 255}
	ttGreen  = color.RGBA{R: 100, G: 220, B: 80, A: 255}
	ttPurple = color.RGBA{R: 220, G: 140, B: 255, A: 255}
	ttGray   = color.RGBA{R: 160, G: 170, B: 190, A: 200}
	ttOrange = color.RGBA{R: 255, G: 180, B: 60, A: 255}
	ttCyan   = color.RGBA{R: 80, G: 220, B: 220, A: 255}
)

// drawEnemyTooltip 绘制敌人属性浮窗（测试模式悬浮检测）。
func (s *StageScene) drawEnemyTooltip(screen *ebiten.Image, e *enemy.Enemy) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	L := func(c color.RGBA, f string, a ...any) enemyTooltipLine {
		return enemyTooltipLine{text: fmt.Sprintf(f, a...), clr: c}
	}

	var lines []enemyTooltipLine

	// ── 标识 ──
	tag := ""
	if e.Boss {
		tag = " Boss"
	}
	if e.Behavior != "" {
		tag += " [" + e.Behavior + "]"
	}
	lines = append(lines, L(ttHeader, "%s%s", e.Archetype, tag))
	lines = append(lines, L(ttDim, "编号:%d  坐标:(%.0f,%.0f)  半径:%.0f  存活:%.1f秒", e.ID, e.X, e.Y, e.Radius, e.Age))

	// ── 生命 ──
	lines = append(lines, L(ttHeader, "--- 生命 ---"))
	lines = append(lines, L(ttWhite, "血量: %.0f / %.0f  (%.1f%%)", e.HP, e.MaxHP, e.HP/e.MaxHP*100))
	lines = append(lines, L(ttDim, "显示血量: %.0f  奖励: %d金 (×%.2f)", e.DisplayHP, e.Reward, e.RewardScale))

	// ── 移动 ──
	lines = append(lines, L(ttHeader, "--- 移动 ---"))
	lines = append(lines, L(ttWhite, "速度: %.1f / %.1f  路径: %d/%d", e.Speed, e.BaseSpeed, e.PathIndex, len(e.Path)))
	if e.SpeedBuff > 0 {
		lines = append(lines, L(ttOrange, "  光环加速: +%.0f%%", e.SpeedBuff*100))
	}
	if e.DashActiveT > 0 {
		lines = append(lines, L(ttOrange, "  冲刺中: +%.0f%% 剩余%.1f秒", e.DashSpeedBoost*100, e.DashActiveT))
	} else if e.DashCooldownT > 0 {
		lines = append(lines, L(ttDim, "  冲刺冷却: %.1f秒", e.DashCooldownT))
	}
	if e.BerserkThreshold > 0 {
		state := "待触发"
		if e.BerserkTriggered {
			state = "已激活"
		}
		lines = append(lines, L(ttOrange, "  狂暴: <%.0f%%血 → ×%.1f速 [%s]", e.BerserkThreshold*100, e.BerserkSpeedScale, state))
	}
	if e.TeleportInterval > 0 {
		lines = append(lines, L(ttCyan, "  传送: 每%.0f秒跳%d段 (冷却:%.1f秒)", e.TeleportInterval, e.TeleportSkip, e.TeleportTimer))
	}

	// ── 控制效果 ──
	hasCC := e.SlowTimer > 0 || e.StunTimer > 0 || e.RootTimer > 0
	if hasCC || e.Tenacity > 0 || e.IsControlImmune || e.IsStunImmune || e.IsSlowImmune || e.IsRootImmune || e.ControlImmuneTimer > 0 {
		lines = append(lines, L(ttHeader, "--- 控制 ---"))
	}
	if e.SlowTimer > 0 {
		lines = append(lines, L(ttIce, "  减速: ×%.0f%%速度  剩余%.1f秒", e.SlowFactor*100, e.SlowTimer))
	}
	if e.StunTimer > 0 {
		lines = append(lines, L(ttYellow, "  眩晕: 剩余%.1f秒", e.StunTimer))
	}
	if e.RootTimer > 0 {
		lines = append(lines, L(ttIce, "  定身: 剩余%.1f秒", e.RootTimer))
	}
	if e.Tenacity > 0 {
		lines = append(lines, L(ttDim, "  韧性: %.0f%%", e.Tenacity*100))
	}
	if e.ControlImmuneTimer > 0 {
		lines = append(lines, L(ttGray, "  控制免疫: 剩余%.1f秒", e.ControlImmuneTimer))
	}
	if e.IsControlImmune {
		lines = append(lines, L(ttGray, "  全控制免疫(永久)"))
	}
	if e.IsStunImmune {
		lines = append(lines, L(ttGray, "  眩晕免疫"))
	}
	if e.IsSlowImmune {
		lines = append(lines, L(ttGray, "  减速免疫"))
	}
	if e.IsRootImmune {
		lines = append(lines, L(ttGray, "  定身免疫"))
	}

	// ── 持续伤害 ──
	hasDot := e.BleedTimer > 0 || e.PoisonTimer > 0 || e.BurnTimer > 0 || e.ZoneDmgAccum > 0
	if hasDot {
		lines = append(lines, L(ttHeader, "--- 持续伤害 ---"))
	}
	if e.BleedTimer > 0 {
		lines = append(lines, L(ttRed, "  流血: %.1f/秒  剩余%.1f秒", e.BleedDPS, e.BleedTimer))
	}
	if e.PoisonTimer > 0 {
		lines = append(lines, L(ttGreen, "  中毒: %.1f/秒  剩余%.1f秒", e.PoisonDPS, e.PoisonTimer))
	}
	if e.BurnTimer > 0 {
		lines = append(lines, L(ttRed, "  灼烧: %.1f/秒  剩余%.1f秒", e.BurnDPS, e.BurnTimer))
	}
	if e.ZoneDmgAccum > 0 {
		lines = append(lines, L(ttPurple, "  区域伤害: %.1f待结算", e.ZoneDmgAccum))
	}

	// ── 减益状态 ──
	hasDebuff := e.DamageAmplify > 0 || e.Silenced || e.Stealthed || e.AbilitySilenced
	if hasDebuff {
		lines = append(lines, L(ttHeader, "--- 减益 ---"))
	}
	if e.DamageAmplify > 0 {
		lines = append(lines, L(ttPurple, "  虚弱: 受伤+%.0f%%  剩余%.1f秒", e.DamageAmplify*100, e.DamageAmplifyTimer))
	}
	if e.Silenced {
		lines = append(lines, L(ttGray, "  沉默(伤害上限失效)"))
	}
	if e.AbilitySilenced {
		lines = append(lines, L(ttGray, "  能力沉默(主动能力禁用)"))
	}
	if e.Stealthed {
		lines = append(lines, L(ttDim, "  隐身: 剩余%.1f秒", e.StealthTimer))
	}

	// ── 防御 ──
	hasDef := e.DamageCap > 0 || e.DamageCapPercent > 0 || e.DamageReduceRatio > 0 ||
		e.ProjectileBlockChance > 0 || e.ArmorFlat > 0 || e.EvasionChance > 0 ||
		e.IsInvincible || e.IsDamageImmune || e.IsUntargetable
	if hasDef {
		lines = append(lines, L(ttHeader, "--- 防御 ---"))
	}
	if e.ProjectileBlockChance > 0 {
		lines = append(lines, L(ttGray, "  弹幕盾: %.0f%%格挡弹射物", e.ProjectileBlockChance*100))
	}
	if e.ArmorFlat > 0 {
		lines = append(lines, L(ttGray, "  装甲: 每次减免%.0f伤害", e.ArmorFlat))
	}
	if e.EvasionChance > 0 {
		lines = append(lines, L(ttGray, "  闪避: %.0f%%概率", e.EvasionChance*100))
	}
	if e.DamageCap > 0 {
		lines = append(lines, L(ttGray, "  坚韧: 单次上限%.0f", e.DamageCap))
	}
	if e.DamageCapPercent > 0 {
		lines = append(lines, L(ttGray, "  坚韧: 单次上限%.0f%%血量", e.DamageCapPercent*100))
	}
	if e.DamageReduceRatio > 0 {
		lines = append(lines, L(ttGray, "  减伤: %.0f%%", e.DamageReduceRatio*100))
	}
	if e.PhaseActive {
		lines = append(lines, L(ttYellow, "  相位免伤中: 剩余%.1f秒", e.PhaseTimer))
	} else if e.PhaseCooldown > 0 {
		lines = append(lines, L(ttDim, "  相位冷却: %.1f秒", e.PhaseTimer))
	}
	if e.IsInvincible && !e.PhaseActive {
		lines = append(lines, L(ttYellow, "  无敌"))
	}
	if e.IsDamageImmune {
		lines = append(lines, L(ttYellow, "  伤害免疫"))
	}
	if e.IsUntargetable && !e.PhaseActive {
		lines = append(lines, L(ttYellow, "  不可选中"))
	}

	// ── 能力 ──
	hasAbil := e.RegenPerSec > 0 || e.HealPower > 0 || e.SplitCount > 0 || e.AuraRange > 0 ||
		e.DeathSpawnCount > 0 || e.StrDrainRatio > 0 || e.PurgeInterval > 0
	if hasAbil {
		lines = append(lines, L(ttHeader, "--- 能力 ---"))
	}
	if e.RegenPerSec > 0 {
		lines = append(lines, L(ttGreen, "  回血: %.1f/秒", e.RegenPerSec))
	}
	if e.HealPower > 0 {
		lines = append(lines, L(ttGreen, "  治疗光环: %.0f治疗量 半径%.0f 每%.1f秒 (冷却:%.1f秒)", e.HealPower, e.HealRadius, e.HealInterval, e.HealCooldown))
	}
	if e.AuraRange > 0 {
		lines = append(lines, L(ttOrange, "  加速光环: +%.0f%%速度 半径%.0f", e.AuraSpeedUp*100, e.AuraRange))
	}
	if e.SplitCount > 0 {
		lines = append(lines, L(ttCyan, "  死亡分裂: %d子体 %.0f%%血量 ×%.1f速", e.SplitCount, e.SplitHPRatio*100, e.SplitSpeedScale))
	}
	if e.DeathSpawnCount > 0 {
		lines = append(lines, L(ttCyan, "  死亡召唤: %d个%s", e.DeathSpawnCount, e.DeathSpawnArch))
	}
	if e.StrDrainRatio > 0 {
		lines = append(lines, L(ttPurple, "  削强: -%.0f%%强度 每%.0f秒 持续%.0f秒 (冷却:%.1f秒)", e.StrDrainRatio*100, e.StrDrainInterval, e.StrDrainDuration, e.StrDrainTimer))
	}
	if e.PurgeInterval > 0 {
		lines = append(lines, L(ttCyan, "  净化: 每%.0f秒清除全debuff+免疫%.0f秒 (计时:%.1f秒)", e.PurgeInterval, e.PurgeImmuneDur, e.PurgeTimer))
	}

	// ── 绘制 ──
	mx, my := draw.CursorPos()
	const (
		fontSize = 10.0
		lineH    = 13.0
		padX     = 8.0
		padY     = 5.0
		offsetX  = 18.0
		offsetY  = 8.0
	)
	boxW := float32(260)
	boxH := float32(float64(len(lines))*lineH + padY*2)
	bx := float32(mx + offsetX)
	by := float32(my + offsetY)

	// Clamp to screen
	sw := float32(game.ScreenWidth)
	sh := float32(game.ScreenHeight)
	if bx+boxW > sw {
		bx = float32(mx) - boxW - 4
	}
	if by+boxH > sh {
		by = sh - boxH - 4
	}
	if by < 0 {
		by = 4
	}

	draw.RoundRect(screen, bx, by, boxW, boxH, 6, color.RGBA{R: 10, G: 14, B: 24, A: 235})
	draw.StrokeRoundRect(screen, bx, by, boxW, boxH, 6, 1, color.RGBA{R: 60, G: 80, B: 120, A: 180})

	for i, line := range lines {
		fm.DrawText(screen, line.text, float64(bx)+padX, float64(by)+padY+float64(i)*lineH, fontSize, line.clr)
	}
}

// saveScenario exports the current tower layout + scene config to a JSON file.
func (s *StageScene) saveScenario(name string) {
	var towers []config.TowerSnapshot
	s.towers.Each(func(t *tower.Tower) {
		var permStr float64
		if t.Strength != nil {
			permStr = t.Strength.Permanent
		}
		towers = append(towers, config.TowerSnapshot{
			Row:             t.Row,
			Col:             t.Col,
			Key:             t.Key,
			AbilitySlots:    t.AbilitySlots,
			DamageTier:      t.DamageTier,
			SpeedTier:       t.SpeedTier,
			RangeTier:       t.RangeTier,
			BaseDamage:      t.BaseDamage,
			PotentialDamage: t.PotentialDamage,
			BaseSpeed:       t.BaseSpeed,
			PotentialSpeed:  t.PotentialSpeed,
			BaseRange:       t.BaseRange,
			PotentialRange:  t.PotentialRange,
			Strength:        permStr,
		})
	})

	// Capture enemies
	var enemies []config.EnemySnapshot
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.DyingTimer > 0 {
			return // skip dying enemies
		}
		enemies = append(enemies, config.EnemySnapshot{
			Archetype: e.Archetype,
			X:         e.X,
			Y:         e.Y,
			PathIndex: e.PathIndex,
			HP:        e.HP,
			MaxHP:     e.MaxHP,
			IsDummy:   e.IsDummy,
		})
	})

	// Generate file-safe ID from name: spaces → underscore, keep unicode letters/digits
	id := strings.Map(func(r rune) rune {
		if r == ' ' {
			return '_'
		}
		// Keep letters (including CJK), digits, dash, underscore
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r > 0x7F {
			return r
		}
		return -1 // drop other special chars
	}, name)
	if id == "" {
		id = fmt.Sprintf("scenario-%s", time.Now().Format("20060102-150405"))
	}

	sd := config.ScenarioData{
		ID:          id,
		Name:        name,
		Description: fmt.Sprintf("%d towers, %d enemies on %s", len(towers), len(enemies), s.initOpts.MapID),
		MapID:       s.initOpts.MapID,
		Gold:        s.initOpts.Gold,
		Lives:       s.initOpts.Lives,
		Waves:       s.initOpts.Waves,
		EnemyFilter: s.initOpts.EnemyFilter,
		ManualWave:  s.initOpts.ManualWave,
		Towers:      towers,
		Enemies:     enemies,
	}

	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}

	path := filepath.Join("config", "scenarios", sd.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		hud.ShowToast("Save failed: " + err.Error())
		return
	}

	hud.ShowToast(fmt.Sprintf("Saved: %s", path))
	fmt.Printf("Scenario saved: %s\n", path)
}

// restoreScenario places towers from a saved scenario snapshot.
// tickStrengthDrain 每帧管理削强敌人→塔的连接、施加/移除减益。
// 多个削强怪优先连接不同的塔。
func (s *StageScene) tickStrengthDrain() {
	// 收集已被占用的塔位
	occupied := map[[2]int]bool{}
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.StrDrainActiveT > 0 && e.StrDrainTargetRC != ([2]int{}) {
			occupied[e.StrDrainTargetRC] = true
		}
	})

	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.StrDrainRatio <= 0 {
			return
		}
		key := fmt.Sprintf("strDrain_%d", e.ID)

		if e.StrDrainActiveT > 0 && e.StrDrainTargetRC == ([2]int{}) {
			// 刚进入激活状态，找塔建立连接（优先未被占用的）
			var bestTower, fallback *tower.Tower
			bestDist, fallDist := 9999.0, 9999.0
			s.towers.Each(func(t *tower.Tower) {
				d := math.Hypot(t.X-e.X, t.Y-e.Y)
				rc := [2]int{t.Row, t.Col}
				if !occupied[rc] {
					if d < bestDist {
						bestDist = d
						bestTower = t
					}
				} else if d < fallDist {
					fallDist = d
					fallback = t
				}
			})
			target := bestTower
			if target == nil {
				target = fallback // 全部被占用时退而求其次
			}
			if target != nil {
				e.StrDrainTargetRC = [2]int{target.Row, target.Col}
				occupied[e.StrDrainTargetRC] = true
			} else {
				e.StrDrainActiveT = 0
				return
			}
		}

		if e.StrDrainActiveT > 0 {
			// 连接中：施加减益
			t := s.towers.At(e.StrDrainTargetRC[0], e.StrDrainTargetRC[1])
			if t == nil || !t.Active {
				// 塔被卖了 → 断开
				e.StrDrainActiveT = 0
				e.StrDrainTargetRC = [2]int{}
				e.StrDrainTimer = e.StrDrainInterval
				return
			}
			if t.Strength != nil {
				t.Strength.SetEnemySub(key, t.Strength.Base*e.StrDrainRatio)
				t.RecalcStats()
			}
		} else {
			// 不在激活状态：清除之前的减益（SetEnemySub 传 0 会自动删除）
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.SetEnemySub(key, 0)
				}
			})
		}
	})
}

// drawStrengthDrainLinks 绘制削强连接线 + 流动粒子（塔→怪物方向，表示被吸取）。
func (s *StageScene) drawStrengthDrainLinks(screen *ebiten.Image) {
	animTime := float64(s.frame) / 60.0
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.StrDrainActiveT <= 0 {
			return
		}
		t := s.towers.At(e.StrDrainTargetRC[0], e.StrDrainTargetRC[1])
		if t == nil || !t.Active {
			return
		}
		tx, ty := float32(t.X), float32(t.Y)
		ex, ey := float32(e.X), float32(e.Y)

		// 底层连接线（半透明）
		draw.ThickLine(screen, ex, ey, tx, ty, 1.5, color.RGBA{R: 140, G: 40, B: 180, A: 60})

		// 流动粒子：从塔→怪物方向，3 个粒子均匀分布沿线移动
		const particleCount = 3
		speed := 1.2 // 粒子移动速度
		for i := 0; i < particleCount; i++ {
			// 每个粒子偏移不同相位
			phase := math.Mod(animTime*speed+float64(i)/particleCount, 1.0)
			// phase 0=塔位置, 1=怪物位置
			px := float32(float64(tx) + float64(ex-tx)*phase)
			py := float32(float64(ty) + float64(ey-ty)*phase)
			// 粒子大小和亮度随位置变化（靠近怪物时更亮更大）
			size := float32(2 + phase*2)
			alpha := uint8(100 + phase*155)
			draw.FilledCircle(screen, px, py, size, color.RGBA{R: 200, G: 80, B: 255, A: alpha})
		}
	})
}

func (s *StageScene) restoreScenario(sd *config.ScenarioData) {
	defMap := make(map[string]tower.TowerDef)
	for _, d := range s.towerDefs {
		defMap[d.Key] = d
	}
	for _, snap := range sd.Towers {
		def, ok := defMap[snap.Key]
		if !ok {
			fmt.Printf("restoreScenario: unknown tower key %q, skip\n", snap.Key)
			continue
		}
		center := s.gameMap.CellCenter(snap.Row, snap.Col)
		t := s.towers.PlaceFromSnapshot(snap.Row, snap.Col, center.X, center.Y, def, snap)
		if t == nil {
			fmt.Printf("restoreScenario: pool full, cannot place %s at (%d,%d)\n", snap.Key, snap.Row, snap.Col)
			continue
		}
		// Restore abilities (order matters: attack mode first changes style/sprite)
		for _, abilityType := range snap.AbilitySlots {
			if abilityType != "" {
				t.AddAbility(abilityType)
			}
		}
		// Restore permanent strength
		if snap.Strength != 0 && t.Strength != nil {
			t.Strength.AddPermanent(snap.Strength)
		}
		t.RecalcStats()
		s.gold -= def.Cost
	}

	// Restore enemies
	for _, snap := range sd.Enemies {
		cfg, ok := s.spawner.Archetypes[snap.Archetype]
		if !ok {
			fmt.Printf("restoreScenario: unknown enemy archetype %q, skip\n", snap.Archetype)
			continue
		}
		// Use snapshot HP as base (bypass wave scaling), scale=1
		baseHP := snap.MaxHP
		if baseHP <= 0 {
			baseHP = 100 * cfg.HpScale // fallback
		}
		unitCfg := *cfg            // copy to avoid mutating original
		unitCfg.HpScale = 1        // HP already baked in
		unitCfg.SpeedScale = 1     // use archetype base speed directly
		e := s.enemies.Spawn(snap.X, snap.Y, baseHP, 50*cfg.SpeedScale, snap.PathIndex, snap.Archetype, &unitCfg)
		if e == nil {
			fmt.Printf("restoreScenario: enemy pool full, cannot spawn %s\n", snap.Archetype)
			continue
		}
		// Override HP if snapshot captured partial health
		if snap.HP > 0 && snap.HP < e.MaxHP {
			e.HP = snap.HP
			e.DisplayHP = snap.HP
		}
		// Assign path from map
		e.Path = s.gameMap.PickPath()
		// 木桩怪标记
		if snap.IsDummy {
			e.IsDummy = true
		}
	}
}

// spawnEntries 构建排序后的造怪菜单条目列表。
func (s *StageScene) spawnEntries() []hud.SpawnEntry {
	var entries []hud.SpawnEntry
	// 按 JSON 定义顺序排列
	for _, name := range config.EnemyArchetypeOrder() {
		cfg, ok := s.spawner.Archetypes[name]
		if !ok {
			continue
		}
		entries = append(entries, hud.SpawnEntry{
			Name: name, Label: cfg.Label,
			HpScale: cfg.HpScale, SpeedScale: cfg.SpeedScale,
			Radius: cfg.Radius,
			Reward: cfg.Reward, Boss: cfg.Boss,
		})
	}
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

	// 0. 非手动模式：第一波倒计时结束时自动弹出战灵选择（仅 idle 时触发，避免打断其他操作）
	if !s.wardenReady && s.spawner.Wave == 0 && s.spawner.TimeToNextWave() <= 0 && !s.testMode && s.imode == modeIdle {
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

	// 2.5. 敌人行为 tick（狂暴/回血/传送）
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		enemy.UpdateBerserk(e)
		enemy.UpdateRegeneration(e, gameDT)
		enemy.UpdateTeleport(e, gameDT)
	})
	// 群体行为（需要遍历所有敌人的交叉操作）
	var activeEnemies []*enemy.Enemy
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.Active && !e.IsDying() {
			activeEnemies = append(activeEnemies, e)
		}
	})
	enemy.UpdateHealing(activeEnemies, gameDT)
	enemy.UpdateBufferAura(activeEnemies, gameDT)
	// TODO: Boss 行为将通过能力系统装配

	// 3. 敌人移动（到达终点扣生命）
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return // dying enemies don't move
		}
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, gameDT) {
			s.lives--
			if s.lives < 0 {
				s.lives = 0
			}
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
	// 削强能力：每帧管理敌人→塔连接
	// 削强在 TickTowerAbilities 之后执行（避免被 ClearTransient 清掉）

	// 4. 战灵行为（未选择前跳过）
	if s.wardenReady && s.wardenUnit != nil {
		s.wardenUnit.Tick(&warden.TickContext{
			Enemies:     s.enemies,
			Towers:      s.towers,
			Projectiles: s.projectiles,
			DT:          gameDT,
			OnKill: func(e *enemy.Enemy) {
				s.audioMgr.PlaySafeAt(gameAudio.SFXEnemyDeath, gameAudio.VolKill)
				s.emitKill(e.Boss, "warden", e.RewardScale)
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
	chainActive := s.wardenType == "envoy" && s.wardenReady
	abilityGold := pipeline.TickTowerAbilities(s.towers, s.enemies, gameDT, chainActive)
	s.gold += abilityGold
	s.gameStats.GoldEarned += abilityGold

	// 6.1 削强：必须在 TickTowerAbilities（ClearTransient）之后，确保 EnemySub 不被清掉
	s.tickStrengthDrain()

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
			if e.HitFlash < 0.06 && e.Age > 0.1 { // 出生 0.1s 内不闪白
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
			s.emitKill(e.Boss, "projectile", e.RewardScale) // 统一击杀事件：kills/gold/session/tutorial/warden
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
			newUnlocks := s.progressMgr.RecordGameResult(s.modeID, s.gameMap.Config.ID, s.kills, true)
			if s.tutorial.IsComplete() {
				s.progressMgr.SetTutorialDone()
			}
			s.checkVictoryAchievements()
			// 显示新解锁提示
			for _, name := range newUnlocks {
				hud.ShowToast("解锁: " + name)
			}
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

func (s *StageScene) AddGold(amount int) { s.gold += amount }

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

	// 场景命名输入覆盖层（画在最上层）
	if s.saveNaming {
		s.drawSaveNaming(screen)
	}

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

	// 削强连接线
	s.drawStrengthDrainLinks(worldTarget)

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
		vm := BuildInfoPanelVM(s.selectedTower, sellValue, s.wavesCleared, s.testMode)
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

	// 测试模式：敌人属性浮窗
	if s.testMode && s.hoveredEnemy != nil {
		s.drawEnemyTooltip(screen, s.hoveredEnemy)
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
			TypeIcon:  towerTypeIcon(def.Key),
			Sprite:    s.towerRenderer.GetSprite("sentinel"),
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

// convertArchetypesToSpawnConfigs 将 config.EnemyArchetype 转换为 enemy.SpawnConfig。
// 能力驱动：从 abilities 配置中读取行为参数，取代旧的硬编码字段。
func convertArchetypesToSpawnConfigs(archetypes map[string]*config.EnemyArchetype) map[string]*enemy.SpawnConfig {
	result := make(map[string]*enemy.SpawnConfig, len(archetypes))
	for key, a := range archetypes {
		sc := &enemy.SpawnConfig{
			Label:        a.Label,
			HpScale:      a.HPScale,
			SpeedScale:   a.SpeedScale,
			Radius:       a.Radius,
			Boss:         a.Boss,
			RewardScale: a.RewardScale,
			// 分裂默认值（被 deathSplit 能力覆盖时使用）
			SplitScale:      0.3,
			SplitHPRatio:    0.3,
			SplitSpeedScale: 1.4,
		}
		// 从能力配置装配行为
		for _, ref := range a.Abilities {
			def := config.ResolveEnemyAbility(ref)
			if def == nil {
				fmt.Printf("convertArchetypes: unknown enemy ability %q for %s\n", ref.Type, key)
				continue
			}
			applyEnemyAbilityToSpawnConfig(sc, def)
		}
		result[key] = sc
	}
	return result
}

// applyEnemyAbilityToSpawnConfig 将一个怪物能力应用到 SpawnConfig。
func applyEnemyAbilityToSpawnConfig(sc *enemy.SpawnConfig, def *config.EnemyAbilityDef) {
	switch def.Type {
	// ── defense ──
	case "projectileBlock":
		sc.ProjectileBlockChance = def.Base
	case "armorPlating":
		sc.ArmorFlat = def.Base
	case "evasion":
		sc.EvasionChance = def.Base
	case "damageCap":
		sc.DamageCap = def.Base
	case "damageCapPercent":
		sc.DamageCapPercent = def.Base

	// ── resist ──
	case "ccImmune":
		sc.CCImmune = true
	case "slowImmune":
		sc.SlowImmune = true
	case "purge":
		sc.PurgeInterval = def.Base  // base=间隔秒数
		sc.PurgeImmuneDur = def.Param // param=免疫时间

	// ── movement ──
	case "stealth":
		sc.StealthDuration = def.Base
		sc.Behavior = "stealth"
	case "dashOnHit":
		sc.DashSpeedBoost = def.Base   // base=速度提升比例
		sc.DashDuration = def.Param    // param=持续时间
		sc.DashCooldown = 5            // 固定冷却5s
	case "phaseShift":
		sc.PhaseDuration = def.Base    // base=免伤时间
		sc.PhaseCooldown = def.Param   // param=冷却时间
	case "teleport":
		sc.TeleportInterval = def.Base
		sc.TeleportSkip = int(def.Param)

	// ── offense ──
	case "strengthDrain":
		sc.StrDrainRatio = def.Base    // base=减益比例
		sc.StrDrainInterval = def.Param // param=间隔
		sc.StrDrainDuration = 6        // 固定6s

	// ── support ──
	case "healAura":
		sc.HealScale = def.Base
		sc.HealRadius = def.Param
		sc.HealInterval = 3            // 固定3s
		sc.Behavior = "healer"
	case "speedAura":
		sc.AuraSpeedUp = def.Base
		sc.AuraRange = def.Param
		sc.Behavior = "buffer"

	// ── death ──
	case "deathSplit":
		sc.SplitCount = int(def.Base)
		sc.SplitHPRatio = def.Param
		sc.Behavior = "splitter"
	case "deathSpawn":
		sc.DeathSpawnCount = int(def.Base)
		sc.DeathSpawnArch = "normal"
	}
}

// ── 战灵选择逻辑 ────────────────────────────────────

// tryStartWave 尝试开波。若战灵未选择则先弹出战灵选择面板。
// onWaveTransition 处理波次变化（prevWave → 当前 Wave）。
// 包括：WaveStarted 事件 + WaveCleared 奖励/事件。
// 由 spawner.Update 自动开波和 tryStartWave 手动开波两条路径统一调用。
func (s *StageScene) onWaveTransition(prevWave int) {
	// 前一波清完奖励（prevWave=0 时无前波）
	// 注意：必须在更新 waveLivesSnapshot 之前检查完美波次
	if prevWave > 0 {
		ctx := s.buildModeCtx()
		result := s.session.OnWaveCleared(prevWave, ctx)
		totalBonus := result.BonusGold + result.PerfectBonus
		s.gold += totalBonus
		s.gameStats.GoldEarned += totalBonus
		perfect := s.lives == s.waveLivesSnapshot && result.PerfectBonus > 0
		if perfect {
			render.SpawnText(float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-50,
				"完美!", color.RGBA{255, 215, 0, 255}, 20, 2.0)
		}
		s.showNotify(result.Message)

		s.bus.Emit(event.EvtWaveCleared, event.WaveClearedPayload{
			Wave: prevWave, Perfect: perfect,
		})
	}

	// 新波开始：更新快照 + 发事件
	s.waveLivesSnapshot = s.lives
	s.bus.Emit(event.EvtWaveStarted, event.WaveStartedPayload{
		Wave: s.spawner.Wave, IsBoss: s.spawner.Wave%5 == 0,
	})
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

// loadTowerDefsOrFallback 从 JSON 配置加载塔定义，失败时返回空列表。
func loadTowerDefsOrFallback() []tower.TowerDef {
	defs, err := loader.LoadTowerDefs()
	if err != nil {
		log.Printf("塔配置加载失败: %v", err)
		return nil
	}
	return defs
}

// filterUnlockedTowers 过滤只保留已解锁的塔定义。
func filterUnlockedTowers(defs []tower.TowerDef, pm *persistence.ProgressManager) []tower.TowerDef {
	result := make([]tower.TowerDef, 0, len(defs))
	for _, d := range defs {
		if pm.IsTowerUnlocked(d.Key) {
			result = append(result, d)
		}
	}
	if len(result) == 0 {
		// 保底：至少有 basic 塔
		return defs[:1]
	}
	return result
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
		WavesCleared:    s.wavesCleared,
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
			IsHit:    e.HitFlash > 0,
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
			Abilities:   t.Abilities,
			AttackStyle: string(t.AttackStyleID),
			HasTarget:   t.Target != nil,
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

	case APActionAddAbility:
		t := s.towers.At(a.Row, a.Col)
		if t != nil {
			t.AddAbility(a.AbilityName)
		}
	}
}

// runAutoPlayFrame 在 updatePlaying 末尾调用，驱动自动对局逻辑。
func (s *StageScene) runAutoPlayFrame() {
	if s.autoPlayer == nil {
		return
	}
	// 仅 idle 模式下执行自动决策，避免打断暂停/升级/道具等交互
	if s.imode != modeIdle {
		return
	}
	snap := s.buildAutoPlaySnapshot()
	actions := s.autoPlayer.OnUpdate(snap)
	for _, a := range actions {
		s.executeAutoPlayAction(a)
	}
}
