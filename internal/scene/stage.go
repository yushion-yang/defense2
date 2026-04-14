// stage.go — 游戏主战斗场景（~3600行，本项目最核心的文件）。
//
// 职责：管理完整的塔防战斗循环，从初始化到胜负判定。
//
// 核心架构：
//   - StageScene struct 持有所有运行时游戏状态（敌人池/塔池/弹射物池/经济/战灵/道具等）
//   - Update() 按 stageState 分发：statePlaying → updatePlaying(), stateVictory/stateDefeat → 等待点击进入结算
//   - updatePlaying() 按固定顺序执行 23 步 Tick Pipeline（顺序关键，不可重排）
//   - Draw() 分两大空间：世界空间(受相机偏移) → 后处理 → HUD 屏幕空间(固定位置)
//
// 关键设计决策：
//   - Event Bus 订阅延迟到首次 Update()，避免被场景切换淡出的 bus.Clear() 清掉
//   - modeCtx 闭包只设一次(initModeCtx)，每帧只更新值字段，实现零分配
//   - 空间网格碰撞检测将弹射物碰撞从 O(P×E)=262K 降至 O(P×k)~4K
//   - AutoPlayer 接口支持无头模式自动对局（跳过全部渲染）
//
// 文件拆分：输入处理 → stage_input.go，类型定义 → stage_types.go，信息面板 VM → stage_info_vm.go
package scene

import (
	"cmp"
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
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "defense2/internal/core/tower/abilities" // blank import: 通过 init() 注册 32 种塔能力到全局注册表
	_ "defense2/internal/core/warden/types"    // blank import: 通过 init() 注册 5 种战灵类型到全局注册表

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
	"defense2/internal/core/mascot"
	"defense2/internal/core/persistence"
	"defense2/internal/core/physics"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/core/timescale"
	"defense2/internal/core/tower"
	"defense2/internal/core/tutorial"
	"defense2/internal/core/warden"
	"defense2/internal/i18n"
	"defense2/internal/input"
	"defense2/internal/loader"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/easing"
	"defense2/internal/render/hud"
	"defense2/internal/render/particle"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// 类型定义 (stageState/interactMode/StageOptions) 已移至 stage_types.go。

// itemDrop 单个掉落物的状态（地面发光→飞行→入库三阶段）。
type itemDrop struct {
	active                     bool
	kind                       item.Kind
	worldX, worldY             float64 // 地面位置（世界坐标）
	groundTimer                float64 // 地面倒计时
	flying                     bool
	screenStartX, screenStartY float64 // 飞行起点（屏幕坐标快照）
	targetX, targetY           float64 // 飞行终点（道具按钮中心）
	flyTimer, flyDur           float64
}

// StageScene 游戏主场景，持有一局游戏的所有运行时状态。
// 字段按逻辑分组，生命周期与一局游戏绑定（场景切换时整体重建）。
type StageScene struct {
	// ── 基础设施 ──
	switcher      Switcher           // 场景切换器引用（切换场景/获取 AudioMgr/EventBus）
	bus           *event.Bus         // 事件总线（从 Switcher 获取，延迟订阅）
	busSubscribed bool               // Bus 订阅是否已完成（延迟到首次 Update，避免被 bus.Clear 清掉）
	session       *gamemode.Session        // 游戏模式会话（跟踪模式状态、胜负条件、计时器）
	ruleset       gamemode.TowerRuleset   // 塔建造规则策略（从 session.Ruleset() 缓存，避免每帧虚调用）
	modeCtx       gamemode.Context        // 缓存的模式上下文（闭包 initModeCtx 设一次，每帧只更新值字段，零分配）
	modeID        string             // 模式 ID（campaign/endless/test 等，用于重玩和持久化）
	diffID        string             // 难度 ID（easy/normal/hard/extreme，用于重玩和持久化）
	frame         int                // 当前帧计数（用于动画计时和周期性任务）
	state         stageState         // 当前游戏状态（statePlaying/stateVictory/stateDefeat）
	initOpts      StageOptions       // 保存原始创建配置（重新开始用）
	audioMgr      *gameAudio.Manager // 音效管理器（从 Switcher 获取，全局共享）

	// ── 核心实体池 ──
	gameMap     *gamemap.GameMap // 运行时地图（路径、格子、建造位）
	enemies     *enemy.Pool      // 敌人对象池（固定容量 200，ActiveList 优化遍历）
	spawner     *enemy.Spawner   // 波次出怪管理器（控制出怪节奏、原型选择、Boss 波）
	towers      *tower.Pool      // 塔对象池（二维网格索引，At(row,col) O(1)查找）
	projectiles *projectile.Pool // 弹射物对象池
	beams       *combat.BeamPool // 光束视觉对象池（宽光束/激光的淡出效果）

	// ── 经济与资源 ──
	econ     economy.Config // 经济配置（击杀奖励、出售回收率等）
	lives    int            // 剩余生命值（敌人到达终点扣1，归零判负）
	maxLives int            // 初始生命值上限（吉祥物条件评估用）
	gold     int            // 当前金币（建塔消耗，击杀获取）
	kills    int            // 累计击杀数

	// ── 塔建造 ──
	towerDefs     []tower.TowerDef // 可建造的塔类型列表（已过滤解锁）
	selectedDef   int              // 当前选中的塔类型索引（建塔面板）
	selectedTower *tower.Tower     // 点击选中的塔（显示信息面板+射程圈，nil=无选中）

	// ── 渲染器 ──
	towerRenderer  *render.TowerRenderer  // 塔 SVG 渲染器（缓存 sprite 图集）
	enemyRenderer  *render.EnemyRenderer  // 敌人 SVG 渲染器（缓存 sprite 图集）
	wardenRenderer *render.WardenRenderer // 战灵精灵渲染器

	// ── 战灵系统 ──
	wardenUnit       *warden.Warden           // 战灵实体（选择前为 nil，选"不选"后保持 nil）
	wardenOverlay    *hud.WardenSelectOverlay // 战灵选择覆盖层（第一波倒计时结束时弹出）
	wardenReady      bool                     // 战灵已选择并激活（选"不选"也算 ready）
	wardenType       string                   // 战灵类型标识（prince/core/chain/skystrike/envoy，用于重玩传递）
	wardenCfg        *config.WardenConfig     // 战灵配置（用于 HUD 面板显示技能描述）
	lastCountdownSec int                      // 上一帧的倒计时整秒数（去重播放 countdownTick 音效）

	// ── 教程与持久化 ──
	tutorial      *tutorial.Tutorial           // 新手教程（已完成则不再显示）
	tutorialSaved bool                         // 教程完成已持久化（避免每帧重复写入）
	progressMgr   *persistence.ProgressManager // 持久化进度管理器（地图解锁、星级记录等）
	lastWave      int                          // 上一帧的波次号（波次变化检测用）

	// ── 道具系统 ──
	inventory      *item.Inventory // 道具背包（6 种道具 × 数量）
	dragItemKind   item.Kind       // 当前拖拽的道具类型
	dragItemActive bool            // 是否正在拖拽道具
	dragHoverTower *tower.Tower    // 拖拽道具时悬停的目标塔
	itemPanelOpen  bool            // 道具面板是否打开

	// ── 道具掉落系统 ──
	itemDrops      [8]itemDrop // 掉落物环形缓冲（同时最多 8 个活跃掉落物）
	itemDropCur    int         // 环形缓冲写游标
	dropCycleCount int         // 当前周期已掉落数（每 N 波重置，配合保底机制）

	// ── 交互状态机（11 个 mode）──
	buildBtnState ui.ButtonState        // 造塔按钮微交互动画
	itemBtnState  ui.ButtonState        // 道具按钮微交互动画
	gameSpeed     int                   // 游戏速度倍率（1 或 2）
	timeScale     *timescale.Controller // 慢动作时间缩放控制器（Boss 击杀时触发）
	imode         interactMode          // 当前交互模式（modeIdle/modeBuildMenu/modePaused 等）
	prePauseMode  interactMode          // 暂停前的交互模式（恢复暂停时回到此模式）
	buildHoverIdx int                   // 建塔面板鼠标悬停索引（-1=无）
	itemHoverIdx  int                   // 道具面板鼠标悬停索引（-1=无）
	gesture       *input.Gesture        // 统一手势识别器（桌面+触摸）

	// ── 波次管理 ──
	waveLivesSnapshot int // 波开始时的生命快照（波结束时比较，无损失=完美波次）
	wavesCleared      int // 已清除波次数（驱动能力槽位解锁）

	// ── 图鉴数据收集 ──
	killsByArchetype map[string]int // 本局各原型击杀统计（持久化到图鉴）
	abilitiesPicked  []string       // 本局选择的能力列表（持久化到图鉴）

	// ── 相机（大地图拖拽平移）──
	camX, camY    float64 // 相机偏移（世界坐标，小地图时为 0）
	dragging      bool    // 是否正在拖拽（鼠标/触摸拖拽地图）
	dragStartX    float64 // 拖拽起始屏幕位置
	dragStartY    float64
	dragCamStartX float64 // 拖拽起始相机位置
	dragCamStartY float64
	dragMoved     bool // 拖拽期间是否产生了位移（区分点击和拖拽）

	// ── HUD 面板状态 ──
	wavePanelOpen   bool               // 左下角波次面板是否展开
	wavePanelState  hud.WavePanelState // 波次面板抽屉动画状态
	wardenPanelOpen bool               // 右下角战灵面板是否展开

	// ── 测试模式专用 ──
	testMode       bool
	scenarioID     string
	enemyFilter    string
	manualWave     bool
	debugPanelOpen bool
	debugShowRange bool
	spawnMode      bool
	spawnMoving    bool         // true=造动怪（放在路径上行走），false=造静怪
	saveNaming     bool         // 场景命名输入中（拦截所有其他输入）
	saveNameBuf    string       // 命名缓冲区
	hoveredEnemy   *enemy.Enemy // 测试模式：鼠标悬浮的敌人（显示详细信息面板）
	spawnType      string
	spawnHoverIdx  int

	// ── 吉祥物击杀 VFX ──
	mascotKillVFXActive   bool
	mascotKillVFXX        float64
	mascotKillVFXY        float64
	mascotKillVFXTimer    float64
	mascotKillVFXDuration float64

	// ── 渲染与后处理 ──
	postPipeline    *postprocess.Pipeline // 后处理管线（bloom/desaturation/lighting/ripple 等）
	particlePool    *particle.Pool        // 粒子系统（死亡/金币/环境/出生/元素等特效）
	debugOverlay    *hud.DebugOverlay     // 调试覆盖层（F2 切换，显示实体数+性能）
	perfTracker     *debug.PerfTracker    // 性能追踪器（FPS/P99/GC/HeapMB）
	qualityAdaptive *game.QualityAdaptive // 自适应画质调节器（根据帧耗时动态调整 High/Medium/Low）
	waveAnnounce    *hud.WaveAnnounce     // 波次开始公告动画（slide-in/hold/slide-out）
	ambientTimer    float64               // 环境粒子发射计时器（每秒一次）

	// ── 连杀与反馈 ──
	multiKillCount int     // 连续击杀计数（1.5s 内无击杀则重置）
	multiKillTimer float64 // 连杀窗口倒计时
	regenTextCD    float64 // 回血浮字节流冷却（避免每帧刷屏，1 秒间隔）

	// ── 覆盖层 ──
	choicePanel *hud.ChoicePanel // 能力选择覆盖层（每 2 波解锁一个，3 选 1）

	// ── AutoPlay 自动对局 ──
	autoPlayer        AutoPlayer // 自动对局驱动（nil=手动模式，非 nil=跳过渲染）
	screenshotPending bool       // F12 截图请求标志

	// ── 成就与统计 ──
	achieveTracker *achievement.Tracker // 成就追踪器（持久化，跨局累计）
	gameStats      GameStats            // 详细游戏统计（结算时传递给 ResultScene）

	// ── 性能优化 ──
	collisionGrid *physics.SpatialGrid // 空间网格碰撞索引（64px cell，弹射物命中检测用）
	entityPosBuf  []physics.EntityPos  // 实体位置缓冲（每帧重建，避免分配）

	// ── 缓存 ──
	cachedMapInfo *AutoPlayMapInfo // autoplay 地图信息缓存（首帧构建后复用）
}

// NewStageSceneWithOpts 创建游戏主场景，接受完整配置选项。
//
// 初始化流程（按依赖顺序）：
//  1. 清理全局残留状态（浮字/VFX/震动/视口/Toast/hover）
//  2. 加载地图配置 → 创建 GameMap
//  3. 初始化持久化存储 + 成就追踪器 + 教程
//  4. 创建 Spawner + 加载敌人原型/能力配置
//  5. 加载战灵配置 + 难度配置 → 应用到 Spawner
//  6. 创建游戏模式 Session
//  7. 如果 opts.WardenType 已指定则直接创建战灵（跳过选择）
//  8. 组装 StageScene struct
//  9. 初始化后处理管线 + 粒子 + 调试工具 + 道具背包
//
// 10. 应用测试模式覆盖 + 场景恢复 + 相机居中
func NewStageSceneWithOpts(sw Switcher, opts StageOptions) *StageScene {
	// ── 清理上一局残留的全局状态 ──
	render.ClearFloatTexts()
	render.ClearImpactVFX()
	render.ClearSplashVFX()
	render.ResetShake()
	render.SetShakeEnabled(true)
	render.ResetViewport()
	hud.ClearToast()
	draw.ResetHover()
	tel.T.Reset()

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

	// 新场景需要重绘地图缓存（旧缓存烘焙了上一局的 towerAt 结果）
	render.InvalidateMapCache()

	// 初始化持久化
	store, err := persistence.DefaultStorage()
	if err != nil {
		store = persistence.NewMemoryStorage()
	}
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

	// 加载战灵配置（用于面板显示），优先从全局缓存读取
	var wardenCfg *config.WardenConfig
	if wc := config.GlobalWardenConfig(opts.WardenType); wc != nil {
		cfg := *wc
		wardenCfg = &cfg
	} else if cfgs, err := config.LoadWardenConfigs(); err == nil {
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
		switcher:         sw,
		session:          session,
		ruleset:          session.Ruleset(),
		modeID:           modeID,
		diffID:           opts.DifficultyID,
		gameMap:          gm,
		enemies:          enemy.DefaultPool(),
		spawner:          spawner,
		towers:           tower.DefaultPool(),
		projectiles:      projectile.DefaultPool(),
		beams:            combat.NewBeamPool(),
		econ:             econ,
		towerRenderer:    render.NewTowerRenderer(config.GetAssetFS()),
		enemyRenderer:    render.NewEnemyRenderer(config.GetAssetFS()),
		wardenRenderer:   render.NewWardenRenderer(config.GetAssetFS()),
		audioMgr:         sw.AudioManager(),
		wardenUnit:       wardenUnit,
		wardenOverlay:    hud.NewWardenSelectOverlay(),
		wardenReady:      wardenReady,
		tutorial:         tut,
		progressMgr:      pm,
		lives:            diff.StartingLives,
		maxLives:         diff.StartingLives,
		gold:             startGold,
		towerDefs:        filterUnlockedTowers(loadTowerDefsOrFallback(), pm),
		selectedDef:      0,
		wardenType:       opts.WardenType,
		wardenCfg:        wardenCfg,
		gameSpeed:        1,
		timeScale:        timescale.New(),
		gesture:          newStageGesture(),
		wavePanelOpen:    true,
		wardenPanelOpen:  false,
		achieveTracker:   achTracker,
		collisionGrid:    physics.NewSpatialGrid(float64(gm.PixelWidth()), float64(gm.PixelHeight())),
		entityPosBuf:     make([]physics.EntityPos, 0, game.MaxEnemies),
		killsByArchetype: make(map[string]int),
	}

	// 初始化模式上下文闭包（只设一次，避免每帧分配）
	s.initModeCtx()

	// 死亡召唤音效回调
	s.enemies.OnDeathSpawn = func(_ *enemy.Enemy, _ int) {
		s.audioMgr.PlayThrottledAt("bossSummonMinions", 500, gameAudio.VolWave)
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
	s.choicePanel.SetAssetFS(config.GetAssetFS())
	s.particlePool.MaxActive = game.Settings().MaxParticles
	s.inventory = item.NewInventoryFromConfig()

	// 注入战灵精灵获取函数到覆盖层
	s.wardenOverlay.SpriteFunc = s.wardenRenderer.GetSprite

	// 测试模式覆盖（优先于难度设置）
	if opts.Gold > 0 {
		s.gold = opts.Gold
	}
	if opts.Lives > 0 {
		s.lives = opts.Lives
		s.maxLives = opts.Lives
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

	// 大地图：初始相机居中
	if totalW := gm.PixelWidth() + gm.OffsetX*2; totalW > float64(game.ScreenWidth) {
		s.camX = (totalW - float64(game.ScreenWidth)) / 2
	}
	if totalH := gm.PixelHeight() + gm.OffsetY*2; totalH > float64(game.ScreenHeight) {
		s.camY = (totalH - float64(game.ScreenHeight)) / 2
	}
	s.clampCamera()

	return s
}

// MascotSnapshot returns a pure-value snapshot of the current stage state
// for the mascot condition evaluation system.
func (s *StageScene) MascotSnapshot() mascot.StageSnapshot {
	snap := mascot.StageSnapshot{
		Wave:        s.spawner.Wave,
		MaxWaves:    s.spawner.MaxWaves,
		Lives:       s.lives,
		MaxLives:    s.maxLives,
		Gold:        s.gold,
		EnemyCount:  s.enemies.Count,
		TowerCount:  s.towers.Count,
		Kills:       s.kills,
		MultiKill:   s.multiKillCount,
		ElapsedSecs: s.session.ElapsedTime,
		IsBossWave:  s.spawner.IsBossWave(),
		WaveActive:  s.spawner.WaveActive,

		// Game state
		Paused:  s.imode == modePaused,
		Victory: s.state == stateVictory,
		Defeat:  s.state == stateDefeat,

		// Performance
		FPS:        s.perfTracker.FPS,
		AvgFrameMs: s.perfTracker.AvgUpdateMs + s.perfTracker.AvgDrawMs,
		HeapMB:     s.perfTracker.HeapMB,
		GCPauseUs:  s.perfTracker.GCPauseUs,

		// UI Context
		InteractMode: int(s.imode),
		QualityLevel: int(game.CurrentQuality),
	}

	// Tower selection info
	if s.imode == modeTowerSel && s.selectedTower != nil {
		snap.SelectedTowerLabel = s.selectedTower.Label
		snap.SelectedTowerStyle = s.selectedTower.AttackStyleID
	}

	// Ability choice options
	if s.imode == modeUpgrade && s.choicePanel != nil && s.choicePanel.Active {
		labels := make([]string, len(s.choicePanel.Options))
		for i, opt := range s.choicePanel.Options {
			labels[i] = opt.Label
		}
		snap.ChoiceAbilities = labels
	}

	// Warden type
	if s.wardenUnit != nil {
		snap.WardenType = s.wardenUnit.Type
	}

	return snap
}

// ExecuteMascotAction executes a mascot battle assistance action.
// Returns true if the action was successfully performed.
func (s *StageScene) ExecuteMascotAction(action *mascot.MascotAction) bool {
	if action.Type != mascot.ActionKillWeakEnemy {
		return false
	}
	// Find the weakest non-boss, non-dying active enemy.
	var weakest *enemy.Enemy
	s.enemies.Each(func(e *enemy.Enemy) {
		if !e.Active || e.Boss || e.DyingTimer > 0 {
			return
		}
		if weakest == nil || e.HP < weakest.HP {
			weakest = e
		}
	})
	if weakest == nil {
		return false
	}
	// Start kill VFX at target position before killing.
	s.mascotKillVFXActive = true
	s.mascotKillVFXX = weakest.X
	s.mascotKillVFXY = weakest.Y
	s.mascotKillVFXTimer = 0
	s.mascotKillVFXDuration = 0.5
	s.enemies.Kill(weakest)
	return true
}

// subscribeBus 注册所有事件总线订阅。
// 注册顺序即执行顺序，关键路径（如 gold 变更）应在统计类订阅之前。
// 事件类型：TowerBuilt/TowerUpgraded/TowerSold/WaveStarted/WaveCleared/EnemyLeaked/EnemyKilled。
// 各订阅者负责：音效播放、成就检测、教程推进、战灵通知、道具掉落周期重置。
func (s *StageScene) subscribeBus() {
	bus := s.bus

	// ── 塔事件 ──────────────────────────────────
	event.OnTyped(bus, event.EvtTowerBuilt, func(p event.TowerBuiltPayload) {
		s.session.OnTowerBuilt()
		s.audioMgr.PlaySafeAt(gameAudio.SFXBuild, gameAudio.VolBuild)
		s.tutorial.OnEvent("towerBuilt")
		// 成就: 累计建塔 + 单局塔种类
		s.achieveTracker.IncrTowersBuilt()
		if s.achieveTracker.TotalTowersBuilt >= achievement.ThresholdOf("builder_10") {
			s.unlockAchievement("builder_10", i18n.T("game.achieve.builder_10"))
		}
		s.achieveTracker.SessionTowerTypes[p.TowerKey] = true
		if len(s.achieveTracker.SessionTowerTypes) >= achievement.ThresholdOf("all_towers") {
			s.unlockAchievement("all_towers", i18n.T("game.achieve.all_towers"))
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
			// Screen shake for boss entrance drama
			render.TriggerShake(4.0, 0.5)
		}
		s.waveAnnounce.Trigger(p.Wave, s.spawner.MaxWaves, p.IsBoss)
		s.tutorial.OnEvent("waveStarted")
	})
	event.OnTyped(bus, event.EvtWaveCleared, func(p event.WaveClearedPayload) {
		prevUnlocked := tower.UnlockedSlots(s.wavesCleared)
		s.wavesCleared++
		newUnlocked := tower.UnlockedSlots(s.wavesCleared)
		// 按模式规则决定是否自动为所有塔 roll 新的待选能力
		if s.ruleset.ShouldAutoRollOnWaveClear() {
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
				hud.ShowToast(i18n.TF("game.ability.new_slots", newPending))
			}
		}
		if s.wardenReady && s.wardenUnit != nil {
			s.wardenUnit.OnWaveClear()
		}
		// 道具掉落周期重置
		if dc := config.GlobalBalance().ItemDrop; dc.CycleWaves > 0 && s.wavesCleared%dc.CycleWaves == 0 {
			s.dropCycleCount = 0
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
		if s.modeID == "endless" && p.Wave >= achievement.ThresholdOf("endless_50") {
			s.unlockAchievement("endless_50", i18n.T("game.achieve.endless_50"))
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
		if p.Archetype != "" {
			s.killsByArchetype[p.Archetype]++
		}
		s.audioMgr.PlayThrottledAt(gameAudio.SFXGoldEarn, 100, gameAudio.VolUI*0.5)
		s.session.OnEnemyKilled(p.IsBoss, s.buildModeCtx())
		s.tutorial.OnEvent("enemyKilled")
		if s.wardenReady && s.wardenUnit != nil {
			s.wardenUnit.OnKill()
		}
		// 成就: 击杀数 + Boss + 金币
		s.achieveTracker.SessionKills++
		if s.achieveTracker.SessionKills >= achievement.ThresholdOf("centurion") {
			s.unlockAchievement("centurion", i18n.T("game.achieve.centurion"))
		}
		if p.IsBoss {
			s.unlockAchievement("first_boss", i18n.T("game.achieve.first_boss"))
		}
		if s.gold > s.achieveTracker.SessionMaxGold {
			s.achieveTracker.SessionMaxGold = s.gold
		}
		if s.achieveTracker.SessionMaxGold >= achievement.ThresholdOf("rich") {
			s.unlockAchievement("rich", i18n.T("game.achieve.rich"))
		}
	})
}

// unlockAchievement 尝试解锁成就并显示 Toast 提示。
func (s *StageScene) unlockAchievement(id, name string) {
	if s.achieveTracker.Unlock(id) {
		hud.ShowToast(i18n.TF("game.achieve.unlocked", name))
	}
}

// emitKill 统一发出击杀事件（弹射物/战灵/技能共用）。
// rewardScale 为敌人原型奖励倍率（如 tank=1.35, runner=0.72），0 或 1 表示无缩放。
func (s *StageScene) emitKill(isBoss bool, killerID string, rewardScale float64, archetype string) {
	gold := s.econ.KillGold()
	if rewardScale > 0 && rewardScale != 1 {
		gold = int(float64(gold) * rewardScale)
	}
	s.bus.Emit(event.EvtEnemyKilled, event.EnemyKilledPayload{
		IsBoss:    isBoss,
		KillerID:  killerID,
		GoldValue: gold,
		Archetype: archetype,
	})
}

// ── 道具掉落系统 ──

// tryItemDrop 在敌人被击杀时判定是否掉落道具。
// 掉落机制：每 CycleWaves 波为一个周期，每周期最多掉 MaxPerCycle 个，
// 概率由 DropChance 控制，周期末波若本周期无掉落则保底强制掉。
// 测试模式下每次击杀必掉。
func (s *StageScene) tryItemDrop(worldX, worldY float64) {
	switch s.ruleset.ItemDropMode() {
	case gamemode.ItemDropEveryKill:
		s.spawnItemDrop(worldX, worldY)
		return
	case gamemode.ItemDropNone:
		return
	case gamemode.ItemDropByWave:
		return // Phase 1: 经典模式每 N 波掉落（由 wave clear 回调处理）
	}
	// ItemDropProbability: 概率掉落 + 周期保底
	cfg := config.GlobalBalance().ItemDrop
	if cfg.CycleWaves <= 0 {
		return
	}
	if s.dropCycleCount >= cfg.MaxPerCycle {
		return
	}
	shouldDrop := rand.Float64() < cfg.DropChance
	// 保底：周期末波且本周期无掉落 → 强制
	if !shouldDrop && s.dropCycleCount == 0 && (s.wavesCleared+1)%cfg.CycleWaves == 0 {
		shouldDrop = true
	}
	if !shouldDrop {
		return
	}
	s.spawnItemDrop(worldX, worldY)
}

// spawnItemDrop 在指定世界坐标生成一个掉落物。
func (s *StageScene) spawnItemDrop(worldX, worldY float64) {
	cfg := config.GlobalBalance().ItemDrop
	kind := item.Kind(rand.Intn(int(item.KindCount)))

	d := &s.itemDrops[s.itemDropCur]
	s.itemDropCur = (s.itemDropCur + 1) % len(s.itemDrops)

	*d = itemDrop{
		active:      true,
		kind:        kind,
		worldX:      worldX,
		worldY:      worldY,
		groundTimer: cfg.GroundSec,
		flyDur:      cfg.FlySec,
	}
	s.dropCycleCount++
}

// tickItemDrops 更新所有活跃掉落物（地面→飞行→入库）。
func (s *StageScene) tickItemDrops(gameDT float64) {
	for i := range s.itemDrops {
		d := &s.itemDrops[i]
		if !d.active {
			continue
		}
		if !d.flying {
			d.groundTimer -= gameDT
			if d.groundTimer <= 0 {
				d.flying = true
				d.screenStartX = d.worldX - s.camX
				d.screenStartY = d.worldY - s.camY
				tx, ty := hud.ActionBarItemBtnCenter()
				d.targetX = float64(tx)
				d.targetY = float64(ty)
			}
		} else {
			d.flyTimer += gameDT
			if d.flyTimer >= d.flyDur {
				s.inventory.Add(d.kind)
				d.active = false
				s.itemBtnState.Trigger()
			}
		}
	}
}

// drawItemDropsGround 在世界空间绘制地面发光效果。
func (s *StageScene) drawItemDropsGround(worldTarget *ebiten.Image) {
	cfg := config.GlobalBalance().ItemDrop
	for i := range s.itemDrops {
		d := &s.itemDrops[i]
		if !d.active || d.flying {
			continue
		}
		clr := item.Defs[d.kind].Color
		vfx.DrawItemDropGlow(worldTarget, float32(d.worldX), float32(d.worldY),
			d.groundTimer, cfg.GroundSec, clr)
	}
}

// drawItemDropsFly 在屏幕空间绘制飞行动画。
func (s *StageScene) drawItemDropsFly(screen *ebiten.Image) {
	for i := range s.itemDrops {
		d := &s.itemDrops[i]
		if !d.active || !d.flying {
			continue
		}
		clr := item.Defs[d.kind].Color
		t := easing.Clamp01(d.flyTimer / d.flyDur)
		eased := easing.EaseInQuad(t)
		sx := easing.Lerp(d.screenStartX, d.targetX, eased)
		sy := easing.Lerp(d.screenStartY, d.targetY, eased)
		size := float32(6 * (1 - t*0.5))
		alpha := uint8(255 * (1 - t*0.3))
		draw.FilledCircle(screen, float32(sx), float32(sy), size, color.RGBA{clr.R, clr.G, clr.B, alpha})
		draw.Glow(screen, float32(sx), float32(sy), size, size+6, color.RGBA{clr.R, clr.G, clr.B, alpha / 2})
	}
}

// checkVictoryAchievements checks and unlocks all victory-related achievements.
// calcVictoryStars 计算胜利星级（同 result.go calcStars 逻辑）。
func (s *StageScene) calcVictoryStars() int {
	stars := 1
	if s.spawner.MaxWaves > 0 && s.spawner.Wave >= s.spawner.MaxWaves {
		stars = 3
	} else if s.spawner.MaxWaves > 0 && float64(s.spawner.Wave) >= float64(s.spawner.MaxWaves)*config.GlobalBalance().Gameplay.StarRatingThreshold {
		stars = 2
	}
	return stars
}

func (s *StageScene) checkVictoryAchievements() {
	// first_win — any victory
	s.unlockAchievement("first_win", i18n.T("game.achieve.first_win"))

	stars := s.calcVictoryStars()

	// perfect_star — any map 3 stars
	if stars == 3 {
		s.unlockAchievement("perfect_star", i18n.T("game.achieve.perfect_star"))
	}

	// no_leak_hard — Hard difficulty, zero leaks
	if s.diffID == "hard" && s.session.Stats.Leaked == 0 {
		s.unlockAchievement("no_leak_hard", i18n.T("game.achieve.no_leak_hard"))
	}

	// speedrun — victory within 10 minutes
	if s.session.ElapsedTime <= float64(achievement.ThresholdOf("speedrun")) {
		s.unlockAchievement("speedrun", i18n.T("game.achieve.speedrun"))
	}

	// extreme_master — any Extreme victory
	if s.diffID == "extreme" {
		s.unlockAchievement("extreme_master", i18n.T("game.achieve.extreme_master"))
		// extreme_perfect — Extreme + 3 stars
		if stars == 3 {
			s.unlockAchievement("extreme_perfect", i18n.T("game.achieve.extreme_perfect"))
		}
	}
}

// buildModeCtx 更新缓存的游戏模式上下文并返回。
// 闭包在 initModeCtx 中设一次，每帧只更新值字段（零分配）。
func (s *StageScene) buildModeCtx() *gamemode.Context {
	c := &s.modeCtx
	c.Wave = s.spawner.Wave
	c.MaxWaves = s.spawner.MaxWaves
	c.Lives = s.lives
	c.Gold = s.gold
	c.Kills = s.kills
	c.Leaked = s.session.Stats.Leaked
	c.TowersBuilt = s.session.Stats.TowersBuilt
	c.EnemiesAlive = s.enemies.Count
	c.ElapsedTime = s.session.ElapsedTime
	c.Spawning = !s.spawner.IsClear(s.enemies)
	return c
}

// initModeCtx 初始化 modeCtx 的闭包字段（只调用一次）。
func (s *StageScene) initModeCtx() {
	s.modeCtx.SetLives = func(v int) { s.lives = v }
	s.modeCtx.SetGold = func(v int) { s.gold = v }
	s.modeCtx.AddGold = func(v int) { s.gold += v }
	s.modeCtx.SetMaxWaves = func(v int) { s.spawner.MaxWaves = v }
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
	slices.Sort(names)

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

// Update 每帧逻辑更新的顶层入口。
//
// 状态分发逻辑：
//   - statePlaying → 按 interactMode 分发：modePaused(暂停输入) / modeWardenSelect(战灵选择) / 其他(handleInput + updatePlaying)
//   - stateVictory/stateDefeat → autoPlayer 通知结束 或 等待点击进入 ResultScene
//
// 不受游戏状态影响的全局更新：按钮动画、F12 截图、Toast、教程、自适应画质。
func (s *StageScene) Update() error {
	// 延迟订阅 Bus：避免在构造函数中订阅后被场景切换淡出的 bus.Clear() 清掉
	if !s.busSubscribed {
		s.busSubscribed = true
		s.subscribeBus()
		// BGM: 进入战斗场景播放战斗音乐
		s.audioMgr.PlayBGM(gameAudio.BGMBattle)
	}
	s.frame++

	// 按钮微交互动画更新（每帧 tick，不受暂停影响）
	{
		mx, my := draw.CursorPos()
		mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
		hoveredBtn := hud.ActionBarHitTest(float32(mx), float32(my))
		s.buildBtnState.Update(dt, hoveredBtn == "build", hoveredBtn == "build" && mouseDown)
		s.itemBtnState.Update(dt, hoveredBtn == "items", hoveredBtn == "items" && mouseDown)
	}

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
	s.tutorial.Tick(dt)
	if s.tutorial.Done && !s.tutorialSaved {
		s.progressMgr.SetTutorialDone()
		s.tutorialSaved = true
	}

	// 自适应画质：根据帧耗时动态调整画质等级
	totalMs := s.perfTracker.AvgUpdateMs + s.perfTracker.AvgDrawMs
	s.qualityAdaptive.Tick(totalMs)
	s.particlePool.MaxActive = game.Settings().MaxParticles

	return nil
}

// handleInput 等输入方法已移至 stage_input.go。

// tryPlaceTower 尝试在像素位置放置当前选中类型的塔。
// 检查链：可建造格子 → 无已有塔 → 金币充足 → 池未满。
// 成功后：扣金币、播放建塔动画、滚动初始能力选择、发出 TowerBuilt 事件。
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
	if placed == nil {
		return false // 池满，不扣金
	}
	placed.BuildAnim = 0.3
	// 按模式规则决定建塔时解锁多少个能力位
	tower.RollAndCachePendingChoices(placed, s.ruleset.InitialUnlockWaves(s.wavesCleared))
	s.gold -= cost
	s.gameStats.GoldSpent += cost
	s.gameStats.TowersBuilt++
	render.InvalidateMapCache() // slot occupancy changed
	s.bus.Emit(event.EvtTowerBuilt, event.TowerBuiltPayload{TowerKey: def.Key, Cost: cost})
	return true
}

// findTowerDef 根据塔的 Key 查找对应的 TowerDef。
// 当前只有一种塔类型（basic），Key 在建塔前为 "basic"。
func (s *StageScene) findTowerDef(t *tower.Tower) tower.TowerDef {
	if t == nil {
		return tower.TowerDef{}
	}
	for _, d := range s.towerDefs {
		if d.Key == t.Key {
			return d
		}
	}
	// fallback: 返回第一个 def（通常只有一种）
	if len(s.towerDefs) > 0 {
		return s.towerDefs[0]
	}
	return tower.TowerDef{}
}

// trySellTower 尝试出售像素位置上的塔。
// 设计：不立即删除，而是标记 Selling=true + 启动 0.25s 出售动画，
// 动画结束后由 updatePlaying Step 10 调用 towers.Remove() 真正移除。
// 出售立即返还金币(按回收率)并播放金币粒子。
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
	s.session.Stats.TowersSold++
	// Start sell animation instead of immediate removal
	t.SellAnim = 0.25
	t.Selling = true
	particle.EmitGoldCollect(s.particlePool, t.X, t.Y)
	render.SpawnGoldText(t.X, t.Y-10, refund)
	s.selectedTower = nil
	s.bus.Emit(event.EvtTowerSold, event.TowerSoldPayload{TowerKey: t.Key, Refund: refund})
	s.showNotify(i18n.TF("game.tower.sold", refund))
}

// showNotify 显示屏幕中央通知（通过 toast 系统，自动淡出）。
func (s *StageScene) showNotify(msg string) {
	hud.ShowToast(msg)
}

// debugActions 返回调试面板按钮列表（仅测试模式使用）。
// 按类别分组：强度/战灵/经济/波次/造怪/塔操作/敌方/显示/场景快照/配置审计。
func (s *StageScene) debugActions() []hud.DebugAction {
	rangeLabel := i18n.T("game.debug.show_range")
	if s.debugShowRange {
		rangeLabel = i18n.T("game.debug.hide_range")
	}
	inSpawn := s.imode == modeSpawnMenu || s.imode == modeSpawnPlace

	actions := []hud.DebugAction{
		// ── 强度 ──
		{Label: i18n.T("game.debug.sec_str"), IsSection: true},
		{Label: i18n.T("game.debug.str_plus50"), Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.AddPermanent(50)
				}
			})
		}},
		{Label: i18n.T("game.debug.str_minus50"), Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.AddPermanent(-50)
				}
			})
		}},
		{Label: i18n.T("game.debug.str_reset"), Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.ResetPermanent()
				}
			})
		}},
		{Label: i18n.T("game.debug.str_plus10k"), Action: func() {
			s.towers.Each(func(t *tower.Tower) {
				if t.Strength != nil {
					t.Strength.AddPermanent(10000)
					t.RecalcStats()
				}
			})
		}},
	}

	// ── 战灵 ──
	if s.wardenUnit != nil && s.wardenUnit.Active {
		actions = append(actions,
			hud.DebugAction{Label: i18n.T("game.debug.sec_warden"), IsSection: true},
			hud.DebugAction{Label: i18n.T("game.debug.warden_str100"), Action: func() {
				s.wardenUnit.SelfStrength += 100
			}},
			hud.DebugAction{Label: i18n.T("game.debug.warden_lvup"), Action: func() {
				s.wardenUnit.SelfStrength += 50
			}},
		)
	}

	// ── 经济 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_economy"), IsSection: true},
		hud.DebugAction{Label: i18n.T("game.debug.gold_500"), Action: func() { s.gold += 500 }},
		hud.DebugAction{Label: i18n.T("game.debug.gold_5000"), Action: func() { s.gold += 5000 }},
		hud.DebugAction{Label: i18n.T("game.debug.gold_zero"), Action: func() { s.gold = 0 }},
	)

	// ── 波次 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_wave"), IsSection: true},
		hud.DebugAction{Label: i18n.T("game.debug.next_wave"), Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP = 0 })
			s.spawner.WaveActive = false
			prevWave := s.spawner.Wave
			s.spawner.StartNextWave()
			if s.spawner.Wave > prevWave {
				s.onWaveTransition(prevWave)
			}
		}},
		hud.DebugAction{Label: i18n.T("game.debug.clear_enemies"), Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP = 0 })
		}},
		hud.DebugAction{Label: i18n.T("game.debug.spawn_boss"), Action: func() { s.spawnBoss() }},
	)

	// ── 造怪 ──
	if inSpawn {
		actions = append(actions,
			hud.DebugAction{Label: i18n.T("game.debug.sec_spawn"), IsSection: true},
			hud.DebugAction{Label: i18n.T("game.debug.exit_spawn"), Action: func() {
				s.imode = modeIdle
				s.spawnMode = false
				s.spawnType = ""
			}},
		)
	} else {
		actions = append(actions,
			hud.DebugAction{Label: i18n.T("game.debug.sec_spawn"), IsSection: true},
			hud.DebugAction{Label: i18n.T("game.debug.spawn_static"), Action: func() {
				s.imode = modeSpawnMenu
				s.spawnMode = true
				s.spawnMoving = false
				s.spawnType = ""
			}},
			hud.DebugAction{Label: i18n.T("game.debug.spawn_moving"), Action: func() {
				s.imode = modeSpawnMenu
				s.spawnMode = true
				s.spawnMoving = true
				s.spawnType = ""
			}},
		)
	}

	// ── 塔操作 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_tower"), IsSection: true},
	)

	// ── 敌方 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_enemy"), IsSection: true},
		hud.DebugAction{Label: i18n.T("game.debug.enemy_half_hp"), Action: func() {
			s.enemies.Each(func(e *enemy.Enemy) { e.HP *= 0.5 })
		}},
	)

	// ── 显示 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_display"), IsSection: true},
		hud.DebugAction{Label: rangeLabel, Action: func() { s.debugShowRange = !s.debugShowRange }},
	)

	// ── 场景快照 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_snapshot"), IsSection: true},
		hud.DebugAction{Label: "Save Scenario", Action: func() {
			s.saveNaming = true
			s.saveNameBuf = ""
		}},
	)

	// ── 配置审计 ──
	actions = append(actions,
		hud.DebugAction{Label: i18n.T("game.debug.sec_audit"), IsSection: true},
		hud.DebugAction{Label: i18n.T("game.debug.ability_audit"), Action: func() {
			audit, err := config.AuditAbilities()
			if err != nil {
				hud.ShowToast(i18n.TF("game.debug.audit_failed", err.Error()))
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
		hud.ShowToast(i18n.T("game.debug.save_cancelled"))
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

// drawEnemyInfoPanel 绘制敌人信息面板（左下角固定位置，测试模式专用）。
func (s *StageScene) drawEnemyInfoPanel(screen *ebiten.Image, e *enemy.Enemy) {
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
	lines = append(lines, L(ttDim, "显示血量: %.0f  奖励倍率: ×%.2f", e.DisplayHP, e.RewardScale))

	// ── 移动 ──
	lines = append(lines, L(ttHeader, "--- 移动 ---"))
	// 计算实际移动速度（与 movement.go 一致）
	actualSpeed := e.Speed
	speedUp := e.GetSpeedUp()
	if speedUp > 0 {
		actualSpeed *= (1 + speedUp)
	}
	if e.DashActiveT > 0 {
		actualSpeed *= (1 + e.DashSpeedBoost)
	}
	if e.IsStunned() || e.IsRooted() || e.IsDummy {
		actualSpeed = 0
	}
	speedInfo := fmt.Sprintf("速度: %.1f", actualSpeed)
	if actualSpeed != e.BaseSpeed {
		speedInfo += fmt.Sprintf(" (基础:%.1f", e.BaseSpeed)
		if e.IsSlowed() {
			speedInfo += fmt.Sprintf(" 减速:×%.0f%%", e.GetSlowFactor()*100)
		}
		if speedUp > 0 {
			speedInfo += fmt.Sprintf(" 光环:+%.0f%%", speedUp*100)
		}
		if e.DashActiveT > 0 {
			speedInfo += fmt.Sprintf(" 冲刺:+%.0f%%", e.DashSpeedBoost*100)
		}
		if e.IsStunned() {
			speedInfo += " 眩晕"
		}
		if e.IsRooted() {
			speedInfo += " 定身"
		}
		speedInfo += ")"
	}
	lines = append(lines, L(ttWhite, "%s  路径:%d/%d", speedInfo, e.PathIndex, len(e.Path)))

	// ── 装配能力（从能力配置表读取描述 + 实际运行时值）──
	abilTable := config.GlobalEnemyAbilityTable()
	if len(e.AbilityIDs) > 0 && abilTable != nil {
		lines = append(lines, L(ttHeader, "--- 能力 ---"))
		for _, aid := range e.AbilityIDs {
			def := abilTable[aid]
			if def == nil {
				lines = append(lines, L(ttDim, "  [%s] 未知能力", aid))
				continue
			}
			silenced := e.AbilitySilenced && def.Silenceable
			label := def.Label
			if silenced {
				label += " [沉默]"
			}
			// 显示实际运行时属性值
			runtimeVal := abilityRuntimeValue(e, aid)
			if runtimeVal != "" {
				lines = append(lines, L(ttCyan, "  %s: %s  [实际:%s]", label, def.Description, runtimeVal))
			} else {
				lines = append(lines, L(ttCyan, "  %s: %s", label, def.Description))
			}
		}
	}

	// ── 实时状态（debuff/控制）──
	hasStatus := e.IsSlowed() || e.IsStunned() || e.IsRooted() ||
		e.IsWeakened() || e.Silenced || e.AbilitySilenced || e.IsStealthed() ||
		e.IsBleeding() || e.Buffs.Has("poison") || e.IsBurning() || e.ZoneDmgAccum > 0 ||
		e.DashActiveT > 0 || e.PhaseActive || e.StrDrainActiveT > 0 || e.HasControlImmunity()
	if hasStatus {
		lines = append(lines, L(ttHeader, "--- 实时状态 ---"))
	}
	if b, ok := e.Buffs.Get("slow"); ok {
		lines = append(lines, L(ttIce, "  减速: ×%.0f%%  %.1f秒", b.Value*100, b.Remaining))
	}
	if b, ok := e.Buffs.Get("stun"); ok {
		lines = append(lines, L(ttYellow, "  眩晕: %.1f秒", b.Remaining))
	}
	if b, ok := e.Buffs.Get("root"); ok {
		lines = append(lines, L(ttIce, "  定身: %.1f秒", b.Remaining))
	}
	if b, ok := e.Buffs.Get("bleed"); ok {
		lines = append(lines, L(ttRed, "  流血: %.1f/秒 %.1f秒", b.Value, b.Remaining))
	}
	if b, ok := e.Buffs.Get("poison"); ok {
		lines = append(lines, L(ttGreen, "  中毒: %.1f/秒 %.1f秒", b.Value, b.Remaining))
	}
	if b, ok := e.Buffs.Get("burn"); ok {
		lines = append(lines, L(ttRed, "  灼烧: %.1f/秒 %.1f秒", b.Value, b.Remaining))
	}
	if e.ZoneDmgAccum > 0 {
		lines = append(lines, L(ttPurple, "  区域伤害: %.1f待结算", e.ZoneDmgAccum))
	}
	if b, ok := e.Buffs.Get("weaken"); ok {
		lines = append(lines, L(ttPurple, "  虚弱: +%.0f%% %.1f秒", b.Value*100, b.Remaining))
	}
	if e.Silenced {
		lines = append(lines, L(ttGray, "  沉默(伤害上限失效)"))
	}
	if e.AbilitySilenced {
		lines = append(lines, L(ttGray, "  能力沉默(主动能力禁用)"))
	}
	if e.IsStealthed() {
		lines = append(lines, L(ttDim, "  隐身: %.1f秒", e.StealthRemaining()))
	}
	if e.DashActiveT > 0 {
		lines = append(lines, L(ttOrange, "  冲刺中: +%.0f%% %.1f秒", e.DashSpeedBoost*100, e.DashActiveT))
	}
	if e.PhaseActive {
		lines = append(lines, L(ttPurple, "  相位免伤中: %.1f秒", e.PhaseTimer))
	}
	if e.StrDrainActiveT > 0 {
		lines = append(lines, L(ttPurple, "  削强连接中: %.1f秒", e.StrDrainActiveT))
	}
	if b, ok := e.Buffs.Get("controlImmune"); ok {
		lines = append(lines, L(ttGray, "  控制免疫: %.1f秒", b.Remaining))
	}

	// ── 绘制（左下角固定位置）──
	const (
		fontSize = 10.0
		lineH    = 13.0
		padX     = 8.0
		padY     = 5.0
		margin   = 10.0
	)
	boxW := float32(340)
	boxH := float32(float64(len(lines))*lineH + padY*2)
	sh := float32(game.ScreenHeight)
	// 左下角，底部留 margin
	bx := float32(margin)
	by := sh - boxH - float32(margin)
	// 防止超出屏幕顶部
	if by < float32(margin) {
		by = float32(margin)
		boxH = sh - float32(margin)*2
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
			Specialty:       t.Specialty,
			Strength:        permStr,
		})
	})

	// Capture enemies
	var enemies []config.EnemySnapshot
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.DyingTimer > 0 || e.SpawnTimer > 0 {
			return // skip dying/spawning enemies
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
		hud.ShowToast(i18n.TF("game.debug.save_failed", err.Error()))
		return
	}

	path := filepath.Join("config", "scenarios", sd.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		hud.ShowToast(i18n.TF("game.debug.save_failed", err.Error()))
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		hud.ShowToast(i18n.TF("game.debug.save_failed", err.Error()))
		return
	}

	hud.ShowToast(i18n.TF("game.debug.saved", path))
	log.Printf("Scenario saved: %s", path)
}

// restoreScenario places towers from a saved scenario snapshot.
// tickStrengthDrain 每帧管理"削强"机制：敌人通过光链吸取塔的 Strength 属性。
// 设计：多个削强怪优先连接不同的塔（occupied 数组防重复），全被占用时允许共享。
// 连接建立后每帧通过 SetEnemySub 施加 Strength 减益，塔被卖/敌人死亡则断开。
func (s *StageScene) tickStrengthDrain() {
	// 收集已被占用的塔位 (fixed-size array avoids map allocation)
	var occupied [20][42]bool
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.StrDrainActiveT > 0 && e.StrDrainTargetRC != ([2]int{}) {
			r, c := e.StrDrainTargetRC[0], e.StrDrainTargetRC[1]
			if r >= 0 && r < 20 && c >= 0 && c < 42 {
				occupied[r][c] = true
			}
		}
	})

	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.StrDrainRatio <= 0 {
			return
		}
		key := "strDrain_" + strconv.Itoa(e.ID)

		if e.StrDrainActiveT > 0 && e.StrDrainTargetRC == ([2]int{}) {
			// 刚进入激活状态，找塔建立连接（优先未被占用的）
			var bestTower, fallback *tower.Tower
			bestDist, fallDist := 9999.0, 9999.0
			s.towers.Each(func(t *tower.Tower) {
				d := math.Hypot(t.X-e.X, t.Y-e.Y)
				isOccupied := t.Row >= 0 && t.Row < 20 && t.Col >= 0 && t.Col < 42 && occupied[t.Row][t.Col]
				if !isOccupied {
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
				if target.Row >= 0 && target.Row < 20 && target.Col >= 0 && target.Col < 42 {
					occupied[target.Row][target.Col] = true
				}
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
				t.Strength.SetEnemySub(key, t.Strength.Effective()*e.StrDrainRatio)
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
		if e.IsDying() || e.StrDrainRatio <= 0 || e.StrDrainActiveT <= 0 {
			return
		}
		if e.StrDrainTargetRC == ([2]int{}) {
			return
		}
		t := s.towers.At(e.StrDrainTargetRC[0], e.StrDrainTargetRC[1])
		if t == nil || !t.Active {
			return
		}
		tx, ty := float32(t.X), float32(t.Y)
		ex, ey := float32(e.X), float32(e.Y)

		vfx.DrawStrengthDrainLink(screen, tx, ty, ex, ey, animTime)
	})
}

// drawEnemyAbilityVFX 绘制怪物能力视觉特效（触发特效 + 范围/连接）。
func (s *StageScene) drawEnemyAbilityVFX(screen *ebiten.Image) {
	animTime := float64(s.frame) / 60.0
	s.enemies.Each(func(e *enemy.Enemy) {
		if !e.Active {
			return
		}
		ex, ey := float32(e.X), float32(e.Y)

		// ── 触发特效 ──

		// 格挡闪光（蓝色盾形脉冲）
		if e.BlockFlash > 0 {
			vfx.DrawBlockFlash(screen, ex, ey, float32(e.Radius), e.BlockFlash)
		}

		// 闪避残影（白色偏移残影）
		if e.DodgeFlash > 0 {
			vfx.DrawDodgeFlash(screen, ex, ey, float32(e.Radius), e.DodgeFlash)
		}

		// 装甲火花（灰色小火花）
		if e.ArmorSpark > 0 {
			vfx.DrawArmorSpark(screen, ex, ey, float32(e.Radius), e.ArmorSpark)
		}

		// 坚韧触发脉冲（橙色扩散圈）
		if e.DamageCapHit > 0 {
			vfx.DrawDamageCapPulse(screen, ex, ey, float32(e.Radius), e.DamageCapHit)
		}

		// 净化脉冲（白色扩散圈）
		if e.PurgeFlash > 0 {
			vfx.DrawPurgeWave(screen, ex, ey, float32(e.Radius), e.PurgeFlash)
		}

		// ── 持续状态 ──

		// 相位偏移：紫色脉冲光环（免伤中）
		if e.PhaseActive {
			vfx.DrawPhaseAura(screen, ex, ey, float32(e.Radius), animTime)
		}

		// 受击冲刺：速度拖尾线
		if e.DashActiveT > 0 {
			dirX, dirY := -1.0, 0.0
			if e.PathIndex < len(e.Path) {
				target := e.Path[e.PathIndex]
				dx := target.X - e.X
				dy := target.Y - e.Y
				dist := math.Hypot(dx, dy)
				if dist > 0.1 {
					dirX = dx / dist
					dirY = dy / dist
				}
			}
			vfx.DrawDashTrails(screen, ex, ey, dirX, dirY)
		}

		// ── 范围/光环 （不被沉默时显示）──

		if e.AbilitySilenced {
			return
		}

		// 治疗光环范围圈（绿色虚线圈）
		if _, hr, ok := e.GetHealAuraParams(); ok && !e.IsDying() {
			vfx.DrawHealerAura(screen, ex, ey, float32(hr), animTime)
			if e.HealCooldown > e.HealInterval-0.4 {
				progress := (e.HealInterval - e.HealCooldown) / 0.4
				vfx.DrawHealPulse(screen, ex, ey, float32(e.Radius), float32(hr), progress)
			}
		}

		// 加速光环范围圈（橙色虚线圈）
		if su, ar, ok := e.GetBufferAuraParams(); ok && su > 0 && !e.IsDying() {
			vfx.DrawSpeedAura(screen, ex, ey, ar, animTime)
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
			log.Printf("restoreScenario: unknown tower key %q, skip", snap.Key)
			continue
		}
		center := s.gameMap.CellCenter(snap.Row, snap.Col)
		t := s.towers.PlaceFromSnapshot(snap.Row, snap.Col, center.X, center.Y, def, snap)
		if t == nil {
			log.Printf("restoreScenario: pool full, cannot place %s at (%d,%d)", snap.Key, snap.Row, snap.Col)
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
			log.Printf("restoreScenario: unknown enemy archetype %q, skip", snap.Archetype)
			continue
		}
		// Use snapshot HP as base (bypass wave scaling), scale=1
		baseHP := snap.MaxHP
		if baseHP <= 0 {
			baseHP = 100 * cfg.HpScale // fallback
		}
		unitCfg := *cfg        // copy to avoid mutating original
		unitCfg.HpScale = 1    // HP already baked in
		unitCfg.SpeedScale = 1 // use archetype base speed directly
		e := s.enemies.Spawn(snap.X, snap.Y, baseHP, 50*cfg.SpeedScale, snap.PathIndex, snap.Archetype, &unitCfg)
		if e == nil {
			log.Printf("restoreScenario: enemy pool full, cannot spawn %s", snap.Archetype)
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
			Radius: cfg.Radius, Boss: cfg.Boss,
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
// updatePlaying 执行一帧的完整游戏逻辑（23 步 Tick Pipeline）。
//
// ===== Tick Pipeline 总览（顺序关键，不可重排）=====
//
// 前置：暂停检查 → gameDT 计算 → Hit-stop 冻帧 → 昼夜色温
//
//	Step  1: 战灵选择检查（首波倒计时结束时弹出选择面板）
//	Step  2: 波间倒计时音效（最后 5 秒每秒 tick）
//	Step  3: 生成敌人（Spawner.Tick → 新波事件）
//	Step  4: 敌人状态效果（DoT 伤害 + buff 倒计时）
//	Steps 5-8: [合并单次遍历] 传送/移动/出生动画/死亡动画
//	Step  9: 敌人行为（healer/buffer/stealth/berserk/regen）
//	Step 10: 塔建造/出售动画
//	Step 11: 塔持续粒子（火焰/冰冻塔有目标时）
//	Step 12: 能力 tick（重置→光环→区域→经济）← 必须在索敌前
//	Step 13: 战灵行为 tick（攻击/特殊/成长）← 必须在 ClearTransient 之后
//	Step 14: 削强系统（敌人→塔的 Strength 减益连接）
//	Step 15: 收集动态光源（路径端点>Boss>战灵>塔）
//	Step 16: 塔索敌射击（瞄准→发射弹射物/光束/直接伤害）
//	Step 17: 弹射物移动 + 光束衰减
//	Step 18: 碰撞检测（空间网格 + 弹射物命中 → 伤害/击杀/连杀/VFX）
//	Step 19: 流血粒子
//	Step 20: VFX 更新（浮字/冲击/粒子/震动/道具掉落）
//	Step 21: [由 onWaveTransition 处理] 波次完成奖励
//	Step 22: 后置安全网（清除本帧新产生的 HP≤0 敌人）
//	Step 23: 胜负判定 → 持久化 → 成就检查 → 解锁提示
//
// 末尾：AutoPlay 决策钩子
func (s *StageScene) updatePlaying() {
	s.perfTracker.BeginUpdate()
	defer s.perfTracker.EndUpdate()

	if s.imode == modePaused {
		return
	}
	// 游戏速度倍率：基础 dt × 倍速(1x/2x) × 慢动作缩放(0~1)
	gameDT := dt * float64(s.gameSpeed) * s.timeScale.Tick(dt)

	// Screen effects update (hit-stop freezes game logic for this frame).
	if s.postPipeline.Effects.Update(gameDT) {
		// Hit-stop: skip game logic but still update VFX for visual feedback.
		render.UpdateFloatTexts(gameDT)
		render.UpdateImpactVFX(gameDT)
		render.UpdateSplashVFX(gameDT)
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

	// === Tick Pipeline（顺序关键，不可重排）===

	// Step 1: 非手动模式：第一波倒计时结束时自动弹出战灵选择（仅 idle 时触发，避免打断其他操作）
	if !s.wardenReady && s.spawner.Wave == 0 && s.spawner.TimeToNextWave() <= 0 && !s.testMode && s.imode == modeIdle {
		s.showWardenSelect()
		return
	}

	// Step 2: 波间倒计时音效（每整秒 tick，仅最后 5 秒）
	if s.spawner.IsIntermission() {
		t := s.spawner.TimeToNextWave()
		sec := int(math.Ceil(t))
		if sec != s.lastCountdownSec && sec > 0 && sec <= 5 {
			s.audioMgr.PlayAt("countdownTick", gameAudio.VolUI)
			s.lastCountdownSec = sec
		}
	} else {
		s.lastCountdownSec = 0
	}

	// Step 3: 生成敌人
	s.spawner.Tick(s.enemies, gameDT)
	// Emit spawn burst particles for newly spawned enemies
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsSpawning() && e.SpawnTimer >= e.SpawnDuration-gameDT*1.5 {
			particle.EmitSpawnBurst(s.particlePool, e.X, e.Y)
		}
	})
	if s.spawner.Wave > prevWave {
		s.onWaveTransition(prevWave)
	}

	// 波次公告动画更新
	s.waveAnnounce.Update(gameDT)
	s.wavePanelState.Update(gameDT, s.wavePanelOpen)

	// Step 4: 敌人状态效果（减速、流血等）
	pipeline.TickEnemyStatusEffects(s.enemies, gameDT, func(e *enemy.Enemy, dmg float64) {
		render.SpawnDamageText(e.X, e.Y-10, dmg, false, e.Boss)
		// DoT-type-specific tick sounds
		if e.IsBurning() {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXBurnTick, 1000, gameAudio.VolHit*0.3)
		}
		if e.IsBleeding() {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXBleedTick, 1000, gameAudio.VolHit*0.3)
		}
		if e.IsPoisoned() {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXPoisonTick, 1000, gameAudio.VolHit*0.3)
		}
	})

	// Steps 5-8 merged: 传送/移动/出生动画/死亡动画 (single pass over active enemies)
	// Step 6 (movement) can call KillImmediate (pool mutation), so we use Each() for safety.
	s.enemies.Each(func(e *enemy.Enemy) {
		// Step 8: dying enemies — tick shrink+fade animation
		if e.IsDying() {
			e.DyingTimer -= gameDT
			if e.DyingTimer <= 0 {
				s.enemies.FinishDying(e)
			}
			return
		}
		// Step 7: spawn animation countdown
		if e.IsSpawning() {
			e.SpawnTimer -= gameDT
			if e.SpawnTimer < 0 {
				e.SpawnTimer = 0
			}
			return
		}
		// Step 5: teleport
		if enemy.TickTeleport(e, gameDT) {
			s.audioMgr.PlayThrottledAt("teleportBlink", 200, gameAudio.VolHit)
		}
		// Step 6: movement (may KillImmediate on leak)
		if enemy.MoveAlongPath(e, s.gameMap.Waypoints, gameDT) {
			s.lives--
			if s.lives < 0 {
				s.lives = 0
			}
			s.enemies.KillImmediate(e)
			fx := s.postPipeline.Effects
			fx.HitTintR, fx.HitTintG, fx.HitTintB = 1.0, 0.1, 0.1
			fx.TriggerHitFlash(0.15)
			s.bus.Emit(event.EvtEnemyLeaked, event.EnemyLeakedPayload{})
		}
	})

	if s.regenTextCD > 0 {
		s.regenTextCD -= gameDT
	}
	// Step 9: 敌人行为 tick（治疗/隐身/旗手光环/回血）
	behaviorEvents := enemy.TickBehaviors(s.enemies, gameDT)
	for _, heal := range behaviorEvents.Heals {
		// 被治疗的怪物飘绿色回血数字（用全局浮字池，不会被伤害浮字覆盖）
		if heal.Target != nil && heal.Target.Active {
			render.SpawnText(heal.TargetX, heal.TargetY-float64(heal.Target.Radius),
				fmt.Sprintf("+%.0f", heal.Restored),
				color.RGBA{R: 60, G: 220, B: 100, A: 255}, 10, 0.8)
		}
	}
	if len(behaviorEvents.Heals) > 0 {
		s.audioMgr.PlayThrottledAt(gameAudio.SFXMedicHeal, 500, gameAudio.VolHit)
	}
	for range behaviorEvents.Reveals {
		s.audioMgr.PlaySafeAt(gameAudio.SFXStealthReveal, gameAudio.VolKill)
	}
	if len(behaviorEvents.Regens) > 0 {
		// Regen 浮字：每 1 秒节流显示一次（避免每帧刷屏）
		if s.regenTextCD <= 0 {
			for _, rg := range behaviorEvents.Regens {
				render.SpawnText(rg.X, rg.Y-20,
					fmt.Sprintf("+%.0f", rg.Restored/gameDT), // 显示每秒回复速率
					color.RGBA{R: 80, G: 200, B: 120, A: 200}, 9, 0.6)
			}
			s.regenTextCD = 1.0
		}
		s.audioMgr.PlayThrottledAt(gameAudio.SFXRegenTick, 2000, gameAudio.VolHit*0.5)
	}
	if behaviorEvents.Berserks > 0 {
		s.audioMgr.PlayThrottledAt(gameAudio.SFXBerserkActivate, 500, gameAudio.VolWave)
	}
	if behaviorEvents.HasBuffer {
		s.audioMgr.PlayThrottledAt(gameAudio.SFXBannerAura, 3000, gameAudio.VolHit*0.3)
	}
	// Step 10: 塔建造/出售动画 tick
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

	// Step 11: 持续粒子特效：火焰塔/冰冻塔在有目标时发射元素粒子
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

	// Step 12: 能力 tick（重置属性 + 光环 buff + 区域效果 + 经济产出）
	// 必须在索敌射击之前执行，确保 Range 等属性是本帧最新值
	chainActive := s.wardenType == "chain" && s.wardenReady
	abilityGold := pipeline.TickTowerAbilities(s.towers, s.enemies, gameDT, chainActive)
	s.gold += abilityGold
	s.gameStats.GoldEarned += abilityGold

	// Step 13: 战灵行为：必须在 ClearTransient 之后执行，否则 SetTemp 会被清掉
	if s.wardenReady && s.wardenUnit != nil {
		s.wardenUnit.Tick(&warden.TickContext{
			Enemies:     s.enemies,
			Towers:      s.towers,
			Projectiles: s.projectiles,
			DT:          gameDT,
			OnKill: func(e *enemy.Enemy) {
				s.audioMgr.PlaySafeAt(gameAudio.SFXEnemyDeath, gameAudio.VolKill)
				s.emitKill(e.Boss, "warden", e.RewardScale, e.Archetype)
				s.tryItemDrop(e.X, e.Y)
			},
			OnFire: func() {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXWardenFire, 100, gameAudio.VolWarden)
			},
			OnSpecial: func() {
				sfx := wardenSpecialSFX(s.wardenType)
				s.audioMgr.PlayThrottledAt(sfx, 200, gameAudio.VolWarden)
			},
			OnDamage: func(x, y, dmg float64, crit bool) {
				render.SpawnDamageText(x, y, dmg, crit, false)
			},
		})
		// 战灵 SetTemp 后需重算受影响塔的属性（仅脏标记塔）
		s.towers.Each(func(t *tower.Tower) {
			if t.StatsDirty {
				t.RecalcStats()
				t.StatsDirty = false
			}
		})
	}

	// Step 14: 削强：必须在 TickTowerAbilities（ClearTransient）之后，确保 EnemySub 不被清掉
	s.tickStrengthDrain()

	// Step 15: 收集光源（优先级：路径端点 > Boss > 战灵 > 塔）
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
		case "dodge":
			s.audioMgr.PlayThrottledAt("dodge", 200, gameAudio.VolHit)
		case "splash":
			s.audioMgr.PlayThrottledAt("explodeSplash", 150, gameAudio.VolExplo)
		}
	}

	// Step 16: 塔索敌射击（按攻击方式分发）
	pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, s.beams, gameDT, func(t *tower.Tower, style string) {
		s.audioMgr.PlayThrottledAt(gameAudio.FireSFXForStyle(style), 100, gameAudio.VolFire)
		// Muzzle flash particles toward target (or spin angle for AoE)
		if t.Target != nil {
			angle := math.Atan2(t.Target.Y-t.Y, t.Target.X-t.X)
			particle.EmitMuzzleFlash(s.particlePool, t.X, t.Y, angle)
		} else if style == tower.StyleSpinAoE {
			particle.EmitMuzzleFlash(s.particlePool, t.X, t.Y, t.SpinAngle)
		}
	}, func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool) {
		// 直接攻击方式（laser/beam/spin_aoe等）的伤害飘字+冲击特效
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, crit, e.Boss)
			e.TriggerHitFlash()
			render.SpawnTypedImpact(&e.X, &e.Y, attackStyle)
		}
		if crit {
			s.audioMgr.PlayThrottledAt(gameAudio.SFXCritHit, 150, gameAudio.VolHit)
		}
	}, onCC, func(x, y, radius float64) {
		render.SpawnSplashRing(x, y, radius)
	})

	// Step 17: 弹射物移动 + 光束衰减
	s.projectiles.Tick(gameDT)
	s.beams.Tick(gameDT)

	// Step 18: 重建碰撞网格 + 弹射物命中检测
	s.entityPosBuf = s.entityPosBuf[:0]
	for i := 0; i < s.enemies.Len(); i++ {
		e := s.enemies.ByIndex(i)
		s.entityPosBuf = append(s.entityPosBuf, physics.EntityPos{
			Index: i, X: e.X, Y: e.Y, Active: e.Active && !e.IsDying() && !e.IsSpawning(),
		})
	}
	s.collisionGrid.Rebuild(s.entityPosBuf)
	pipeline.TickProjectileHits(s.projectiles, s.enemies, s.towers, s.collisionGrid, func(e *enemy.Enemy, damage float64, killed bool, attackStyle string, crit bool) {
		if damage > 0 {
			render.SpawnDamageText(e.X, e.Y-15, damage, crit, e.Boss)
			e.TriggerHitFlash()
			// 元素类型化命中特效
			render.SpawnTypedImpact(&e.X, &e.Y, attackStyle)
			// 元素粒子
			switch attackStyle {
			case "scatter":
				particle.EmitIceParticles(s.particlePool, e.X, e.Y, 2)
				s.postPipeline.Effects.TriggerRipple(e.X, e.Y, 1.5)
			case "spin_aoe":
				particle.EmitFireParticles(s.particlePool, e.X, e.Y, 2)
			case "bounce":
				particle.EmitElectricSparks(s.particlePool, e.X, e.Y, 8)
			}
		}
		if killed {
			// Death particles: scale with multi-kill streak
			if s.multiKillCount >= 3 {
				particle.EmitDeathBurstLarge(s.particlePool, e.X, e.Y)
			} else {
				particle.EmitDeathBurst(s.particlePool, e.X, e.Y)
			}
			particle.EmitGoldCollect(s.particlePool, e.X, e.Y)
			// 分裂体死亡音效（子体已由 Pool.Kill 自动生成）
			if e.Behavior == "splitter" && e.SplitCount > 0 {
				s.audioMgr.PlaySafeAt(gameAudio.SFXSplitPop, gameAudio.VolKill)
			}
			// Overkill detection: damage > 2x MaxHP on non-boss
			if e.MaxHP > 0 && damage/e.MaxHP > 2.0 && !e.Boss {
				render.SpawnText(e.X, e.Y-20, i18n.T("game.streak.overkill"), color.RGBA{R: 255, G: 215, B: 0, A: 255}, 16, 1.5)
				particle.EmitDeathBurstLarge(s.particlePool, e.X, e.Y)
				render.TriggerShake(2.0, 0.15)
			}
			// Multi-kill tracker
			s.multiKillCount++
			s.multiKillTimer = config.GlobalBalance().Gameplay.MultiKillWindow
			if s.multiKillCount > s.gameStats.MaxKillStreak {
				s.gameStats.MaxKillStreak = s.multiKillCount
			}
			if s.multiKillCount > s.achieveTracker.SessionMaxStreak {
				s.achieveTracker.SessionMaxStreak = s.multiKillCount
			}
			if s.multiKillCount >= achievement.ThresholdOf("killstreak_20") {
				s.unlockAchievement("killstreak_20", i18n.T("game.achieve.killstreak_20"))
			}
			// Multi-kill tier feedback
			cx := float64(game.ScreenWidth) / 2
			cy := float64(game.ScreenHeight)/2 - 30
			switch {
			case s.multiKillCount == 3:
				render.SpawnText(cx, cy, i18n.T("game.streak.x3"), color.RGBA{R: 255, G: 255, B: 255, A: 220}, 14, 1.2)
			case s.multiKillCount == 5:
				render.SpawnText(cx, cy, i18n.T("game.streak.x5"), color.RGBA{R: 255, G: 220, B: 60, A: 255}, 16, 1.5)
				render.TriggerShake(1.5, 0.1)
			case s.multiKillCount == 10:
				render.SpawnText(cx, cy, i18n.T("game.streak.x10"), color.RGBA{R: 255, G: 140, B: 40, A: 255}, 18, 2.0)
				render.TriggerShake(2.0, 0.15)
			case s.multiKillCount == 20:
				render.SpawnText(cx, cy, i18n.T("game.streak.x20"), color.RGBA{R: 255, G: 60, B: 40, A: 255}, 22, 2.0)
				render.TriggerShake(3.0, 0.2)
			case s.multiKillCount == 50:
				render.SpawnText(cx, cy, i18n.T("game.streak.x50"), color.RGBA{R: 255, G: 215, B: 0, A: 255}, 24, 2.5)
				s.timeScale.Trigger(0.3, 0.1, 0.3, 0.3)
			}
			// Kill audio + boss-specific feedback
			if e.Boss {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXEnemyDeathBoss, 50, gameAudio.VolKill)
				s.postPipeline.Effects.TriggerHitStop(5)
				s.timeScale.Trigger(0.2, 0.1, 0.4, 0.3)
				render.TriggerShake(5.0, 0.4)
				s.postPipeline.Effects.TriggerRadialBlur(e.X, e.Y, 0.04, 0.6)
				s.postPipeline.Effects.TriggerRipple(e.X, e.Y, 20)
				particle.EmitBossDeathBurst(s.particlePool, e.X, e.Y)
			} else {
				s.audioMgr.PlayThrottledAt(gameAudio.SFXEnemyDeath, 50, gameAudio.VolKill)
			}
			s.emitKill(e.Boss, "projectile", e.RewardScale, e.Archetype) // 统一击杀事件：kills/gold/session/tutorial/warden
			s.tryItemDrop(e.X, e.Y)
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
	}, onCC, func(x, y, radius float64) {
		render.SpawnSplashRing(x, y, radius)
	})
	// 击杀统计/金币/session/tutorial/warden 由 emitKill → Bus 订阅者统一处理

	// Step 19: Bleed drip particles for bleeding enemies
	s.enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		if e.IsBleeding() && rand.Float64() < 0.15 { // ~9 particles/sec at 60fps
			particle.EmitBleedDrip(s.particlePool, e.X, e.Y, e.Radius)
		}
	})

	// Step 20: VFX 更新（浮动文本 + 冲击 + 粒子 + 屏幕震动）
	render.UpdateFloatTexts(gameDT)
	render.UpdateImpactVFX(gameDT)
	render.UpdateSplashVFX(gameDT)
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

	// Step 20b: 道具掉落物更新（地面→飞行→入库）
	s.tickItemDrops(gameDT)

	// Step 21: 波次完成奖励 + 事件触发（由 onWaveTransition 统一处理，
	// 此处仅处理 spawner.Tick 触发的波次变化；手动开波的变化在 tryStartWave 中处理）

	// Step 22: 后置安全网：清除本帧内被 abilities/skills/combat 击杀但尚未 Kill 的敌人
	// 前置安全网(step 2)只能处理上一帧残留，本帧新产生的 HP<=0 敌人需要在胜负判定前处理
	pipeline.TickEnemyStatusEffects(s.enemies, 0, nil) // dt=0 不触发 DoT，仅做 HP<=0 检查

	// Mascot kill VFX timer.
	if s.mascotKillVFXActive {
		s.mascotKillVFXTimer += gameDT
		if s.mascotKillVFXTimer >= s.mascotKillVFXDuration {
			s.mascotKillVFXActive = false
		}
	}

	// Step 23: 胜负判定（委托给游戏模式）
	ctx := s.buildModeCtx()
	if s.session.TickEndConditions(ctx) && s.state == statePlaying {
		if s.session.Status == gamemode.StatusVictory {
			s.state = stateVictory
			s.audioMgr.StopBGM()
			s.audioMgr.PlaySafeAt(gameAudio.SFXVictory, gameAudio.VolWave)
			newUnlocks := s.progressMgr.RecordGameResultFull(persistence.GameResultParams{
				ModeID:       s.modeID,
				MapID:        s.gameMap.Config.ID,
				DifficultyID: s.diffID,
				WardenKey:    s.wardenType,
				Kills:        s.kills,
				Score:        s.session.Mode.GetScore(s.buildModeCtx()),
				Stars:        s.calcVictoryStars(),
				Won:          true,
				ElapsedSecs:  s.session.ElapsedTime,
				EnemyKills:   s.killsByArchetype,
				AbilityPicks: s.abilitiesPicked,
			})
			if s.tutorial.IsComplete() {
				s.progressMgr.SetTutorialDone()
			}
			s.checkVictoryAchievements()
			// 显示新解锁提示
			for _, name := range newUnlocks {
				hud.ShowToast(i18n.TF("game.stage.unlocked", name))
			}
		} else if s.session.Status == gamemode.StatusDefeat {
			s.state = stateDefeat
			s.audioMgr.StopBGM()
			s.audioMgr.PlaySafeAt(gameAudio.SFXDefeat, gameAudio.VolWave)
			s.progressMgr.RecordGameResultFull(persistence.GameResultParams{
				ModeID:       s.modeID,
				MapID:        s.gameMap.Config.ID,
				DifficultyID: s.diffID,
				WardenKey:    s.wardenType,
				Kills:        s.kills,
				Won:          false,
				ElapsedSecs:  s.session.ElapsedTime,
				EnemyKills:   s.killsByArchetype,
				AbilityPicks: s.abilitiesPicked,
			})
			s.postPipeline.Effects.SetDesaturation(0.8, 1.5, 0.8, 0.2, 0.2)
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

// Draw 渲染游戏画面的顶层入口。
// 渲染管线：drawFullScene(含屏幕震动) → 场景命名覆盖 → F12 截图。
// AutoPlay 无头模式直接 return，GPU 开销≈0。
//
// shakeBuffer 屏幕震动用的离屏缓冲（懒初始化）。
var shakeBuffer *ebiten.Image

// readScreenPixels 在 Draw 内同步读取屏幕像素，返回 NRGBA image。
// 设计约束：必须在 Draw() 内同步调用，因为 Ebitengine 的 screen 在 Draw 返回后立即被清空。
// 包含全零检测（GPU 尚未渲染时跳过），避免保存空白截图。
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

	// AutoPlay 无头模式：跳过全部渲染，GPU 开销≈0
	if s.autoPlayer != nil {
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
			fname := filepath.Join("docs", "bug", "pic",
				fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405")))
			saveImageAsync(img, fname)
			hud.ShowToast(i18n.T("game.stage.screenshot_saved"))
			log.Printf("screenshot saved: %s", fname)
		}
	}
}

// drawFullScene 执行完整的场景渲染（含屏幕震动包装层）。
// 有震动时：画到 shakeBuffer → 偏移 blit 到 screen。
// 无震动时：直接画到 screen（零额外开销）。
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

// drawScene 渲染完整的一帧画面（核心渲染函数）。
//
// 渲染分三大阶段：
//
// 阶段 1 — 世界空间（受相机偏移影响，画到 worldTarget）：
//
//	地图背景 → 视差背景 → 塔 → buff 特效 → 升级指示器 → 射程圈(调试) →
//	敌人 → 萌妹击杀VFX → 削强连接线 → 怪物能力VFX →
//	[Glow Pass: 弹射物 + 光束] → 冲击/溅射VFX → 粒子 → 道具地面发光 →
//	浮动文字 → 战灵 → 道具拖拽高亮 → 建塔预览
//
// 阶段 2 — 后处理（worldTarget → sceneTarget → screen）：
//
//	相机 blit → bloom/色调/光照/径向模糊/波纹 → 输出到 screen
//
// 阶段 3 — HUD 屏幕空间（固定位置，不受相机影响，直接画到 screen）：
//
//	TopBar → ActionBar → 道具飞行 → 建塔面板 → 道具面板 → 塔信息/战灵面板 →
//	波次面板 → 战灵切换按钮 → 教程 → 敌人信息(测试) → 调试面板 →
//	调试覆盖层 → 小地图 → 波次公告 → 造怪菜单 → 暂停菜单 →
//	道具拖拽指示器 → Toast → 战灵选择 → 能力选择 → 胜负覆盖层
func (s *StageScene) drawScene(screen *ebiten.Image) {
	useCamera := s.needsCamera()

	// 后处理管线：世界元素渲染到 sceneTarget，bloom 后输出到 screen。
	physW, physH := screen.Bounds().Dx(), screen.Bounds().Dy()
	sceneTarget := s.postPipeline.SceneBuffer(physW, physH)

	// 确定世界元素的绘制目标：大地图时使用离屏缓冲，小地图画到 sceneTarget。
	worldTarget := sceneTarget
	if useCamera {
		// worldBuffer 按地图完整尺寸 × draw.Scale 创建（覆盖整个世界）
		logicalW := s.gameMap.PixelWidth() + s.gameMap.OffsetX*2
		logicalH := s.gameMap.PixelHeight() + s.gameMap.OffsetY*2
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

	// 设置视口裁剪参数 + VFX 细节等级
	render.SetViewport(s.camX, s.camY, useCamera)
	// 用实际 TPS 估算帧时间：TPS=60 → 16.7ms, TPS=30 → 33.3ms
	// (VFX level auto-adjustment removed — always render at full quality)

	// 地图（渐变背景覆盖全屏，无需 Fill）
	animTime := float64(s.frame) / 60.0
	inBuildMode := s.imode == modeBuildMenu || s.imode == modeBuildPlace
	render.DrawMap(worldTarget, s.gameMap, render.GlobalFont(), animTime, func(row, col int) bool {
		return s.towers.At(row, col) != nil
	}, inBuildMode)
	render.DrawParallaxBG(worldTarget, animTime, s.gameMap.PixelWidth()+s.gameMap.OffsetX*2, s.gameMap.PixelHeight()+s.gameMap.OffsetY*2)

	// 塔（优先 SVG 渲染，回退到彩色方块）
	s.towerRenderer.DrawTowers(worldTarget, s.towers, s.selectedTower, animTime)

	// 被 buff 的塔显示强化特效（五角星芒）
	s.towers.Each(func(t *tower.Tower) {
		if t.Buffs.Count() > 0 {
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

	// 萌妹击杀魔法阵 VFX
	if s.mascotKillVFXActive {
		vfx.DrawMascotKillMark(worldTarget, s.mascotKillVFXX, s.mascotKillVFXY,
			s.mascotKillVFXTimer, s.mascotKillVFXDuration)
	}

	// 削强连接线
	s.drawStrengthDrainLinks(worldTarget)
	s.drawEnemyAbilityVFX(worldTarget)

	// 弹射物 — Glow Pass 包装：BeginGlowPass 将后续绘制重定向到 GlowTarget，
	// EndGlowPass 将 GlowTarget 以 Additive 混合模式合成回 worldTarget，实现辉光效果。
	draw.BeginGlowPass(worldTarget)
	render.DrawProjectiles(worldTarget, s.projectiles)
	render.DrawBeams(worldTarget, s.beams, 1.0/60.0)
	draw.EndGlowPass(worldTarget)

	// 冲击特效（蓄力弹命中）
	render.DrawImpactVFX(worldTarget)
	render.DrawSplashVFX(worldTarget)

	// 粒子系统
	s.particlePool.Draw(worldTarget)

	// 道具掉落物（地面发光）
	s.drawItemDropsGround(worldTarget)

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

	// ── 阶段 2: 将世界缓冲 blit 到 sceneTarget（带相机偏移） ──
	// 大地图时 worldBuffer 是完整地图大小，需要按 camX/camY 裁剪可见区域。
	if useCamera {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-s.camX*draw.Scale, -s.camY*draw.Scale)
		sceneTarget.DrawImage(worldBuffer, opts)
	}

	// ── 后处理（bloom）→ 输出到 screen ──
	s.postPipeline.Apply(screen)

	// ── 阶段 3: HUD 元素（不受相机偏移影响，直接画到 screen）──
	// 以下所有 HUD 元素使用逻辑屏幕坐标(1200×540)，不受相机 camX/camY 影响。

	// HUD：顶部状态栏（金币/生命/波次/击杀/敌人数/倍速/倒计时）
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

	// HUD：底部动作栏（造塔按钮 + 道具按钮，含微交互动画）
	hud.DrawActionBar(screen, hud.ActionBarData{
		BuildActive:   s.imode == modeBuildMenu || s.imode == modeBuildPlace,
		ItemActive:    s.imode == modeItemPanel || s.imode == modeItemDrag,
		ItemTotal:     s.inventory.TotalCount(),
		BuildBtnState: &s.buildBtnState,
		ItemBtnState:  &s.itemBtnState,
	})

	// 道具掉落物飞行动画（屏幕空间）
	s.drawItemDropsFly(screen)

	// HUD：底部建塔菜单（仅菜单打开时构建数据，避免无用分配）
	if s.imode == modeBuildMenu {
		hud.DrawBuildMenu(screen, s.buildBuildMenuData())
	}

	// HUD：道具面板（仅面板打开时构建数据）
	if s.imode == modeItemPanel || s.imode == modeItemDrag {
		hud.DrawItemPanel(screen, s.buildItemPanelData())
	}

	// HUD：底部中央面板（塔信息面板 与 战灵信息面板 互斥显示）
	if s.selectedTower != nil {
		// 塔选中时显示塔信息面板（底部中央）
		sellValue := s.econ.SellRefund(s.selectedTower.Cost)
		vm := BuildInfoPanelVM(s.selectedTower, sellValue, s.ruleset, s.gold, s.findTowerDef(s.selectedTower).UpgradeCosts)
		hud.DrawInfoPanel(screen, vm)
		// Hover 在面板上时显示升级详情浮窗
		mx, my := draw.CursorPos()
		hud.DrawInfoPanelHoverTooltip(screen, s.selectedTower != nil, float32(mx), float32(my))
	} else if s.wardenPanelOpen && s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		// 无塔选中且战灵面板展开时显示战灵面板（底部中央）
		hud.DrawWardenPanel(screen, s.buildWardenPanelData())
	}

	// 左下角：测试模式显示敌人信息，正常模式显示波次面板
	if !s.testMode {
		hud.DrawWavePanel(screen, s.buildWavePanelData(), &s.wavePanelState)
	}

	// 右下角收起按钮（战灵，选择后才显示）
	if s.wardenReady && s.wardenUnit != nil && s.wardenUnit.Active {
		hud.DrawToggleButtonWithSprite(screen, false, s.wardenPanelOpen, "⚡", s.wardenRenderer.GetSprite(s.wardenType))
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

	// 测试模式：左下角敌人信息面板（替代波次面板）
	if s.testMode && s.hoveredEnemy != nil {
		s.drawEnemyInfoPanel(screen, s.hoveredEnemy)
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
		hud.DrawDragItem(screen, float32(mx), float32(my), def.Color, def.Name, int(s.dragItemKind), def.Icon)
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

	// 胜负覆盖层（最高层级，覆盖所有 HUD 元素）
	if s.state == stateVictory || s.state == stateDefeat {
		// 半透明遮罩 + 大号标题 + 击杀数提示（点击后进入 ResultScene）
		draw.RoundRect(screen, 0, 0, float32(game.ScreenWidth), float32(game.ScreenHeight), 0, theme.HUDGameOverlay)
		if fm := render.GlobalFont(); fm != nil {
			if s.state == stateVictory {
				fm.DrawCenteredText(screen, i18n.T("game.stage.victory"), float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, 52, theme.HUDVictoryColor)
			} else {
				fm.DrawCenteredText(screen, i18n.T("game.stage.defeat"), float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2-20, 52, theme.HUDDefeatColor)
			}
			fm.DrawCenteredText(screen, i18n.TF("game.stage.kills_continue", s.kills), float64(game.ScreenWidth)/2, float64(game.ScreenHeight)/2+30, theme.FontH2, theme.TextMuted)
		}
	}
}

// buildBuildMenuData 构建建塔面板的 ViewModel 数据。
// 包含两部分卡片：可建造的塔类型(购买用) + 攻击方式变种卡片(展示用，不可购买)。
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
	slices.SortFunc(attackAbils, func(a, b *config.AbilityDef) int {
		return cmp.Compare(a.Type, b.Type)
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
	return hud.ItemPanelData{Cards: s.buildItemPanelCards(), Visible: true, HoverIdx: s.itemHoverIdx}
}

func (s *StageScene) buildItemPanelCards() []hud.ItemCardVM {
	cards := make([]hud.ItemCardVM, item.KindCount)
	for _, k := range item.AllKinds {
		cards[k] = hud.ItemCardVM{
			Name:  item.Defs[k].Name,
			Desc:  item.Defs[k].Desc,
			Icon:  item.Defs[k].Icon,
			Count: s.inventory.Count(k),
			Color: item.Defs[k].Color,
			Kind:  int(k),
		}
	}
	return cards
}

// attackStyleDesc 返回攻击方式的纯功能描述（不含数值）。
func attackStyleDesc(abilType string) string {
	key := "game.attack_desc." + abilType
	label := i18n.T(key)
	if label == key {
		return ""
	}
	return label
}

// towerRoleTags 返回塔的角色标签和颜色。
func towerRoleTags(def tower.TowerDef) (string, color.RGBA) {
	for _, ab := range def.Abilities {
		switch ab {
		case "stun":
			return i18n.T("game.role.output_stun"), color.RGBA{R: 180, G: 120, B: 220, A: 255}
		case "bounce":
			return i18n.T("game.role.output_chain"), color.RGBA{R: 220, G: 180, B: 80, A: 255}
		case "splash":
			return i18n.T("game.role.output_splash"), color.RGBA{R: 220, G: 120, B: 80, A: 255}
		case "bleedDot", "burn":
			return i18n.T("game.role.output_dot"), color.RGBA{R: 220, G: 80, B: 80, A: 255}
		case "executionBonus":
			return i18n.T("game.role.output_execute"), color.RGBA{R: 180, G: 60, B: 60, A: 255}
		case "damageUpAura", "attackSpeedAura":
			return i18n.T("game.role.support_aura"), color.RGBA{R: 80, G: 200, B: 120, A: 255}
		case "poisonZone", "silenceZone":
			return i18n.T("game.role.control_zone"), color.RGBA{R: 100, G: 160, B: 200, A: 255}
		case "goldPassive":
			return i18n.T("game.role.economy"), color.RGBA{R: 220, G: 200, B: 80, A: 255}
		}
	}
	return i18n.T("game.role.output"), color.RGBA{R: 200, G: 200, B: 200, A: 200}
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
	default:
		return ""
	}
}

// drawUpgradeIndicators 在有待选能力的塔上方绘制脉冲金色菱形指示器。
func (s *StageScene) drawUpgradeIndicators(target *ebiten.Image, animTime float64) {
	s.towers.Each(func(t *tower.Tower) {
		if t.HasPendingUpgrade() {
			vfx.DrawUpgradeDiamond(target, float32(t.X), float32(t.Y), animTime)
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

// buildWavePanelData 根据当前波次和出怪状态构建波次面板显示数据。
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

// buildWardenPanelData 根据当前战灵状态和配置构建面板显示数据。
func (s *StageScene) buildWardenPanelData() hud.WardenPanelData {
	w := s.wardenUnit
	d := hud.WardenPanelData{
		Type:         w.Type,
		Icon:         s.wardenRenderer.GetSprite(w.Type),
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
			parts += i18n.TF("game.warden.growth_kill_val", cfg.GrowthOnKill)
		}
		if cfg.GrowthOnWaveClear > 0 {
			if parts != "" {
				parts += " "
			}
			parts += i18n.TF("game.warden.growth_wave_val", cfg.GrowthOnWaveClear)
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
// 核心桥梁：将 JSON 配置中的"能力驱动"敌人定义（abilities 数组）
// 转换为运行时可直接使用的 SpawnConfig 结构（行为字段 + 能力 ID 列表 + 潜力表）。
// 每个能力通过 applyEnemyAbilityToSpawnConfig 映射为具体的行为参数。
func convertArchetypesToSpawnConfigs(archetypes map[string]*config.EnemyArchetype) map[string]*enemy.SpawnConfig {
	result := make(map[string]*enemy.SpawnConfig, len(archetypes))
	for key, a := range archetypes {
		sprite := a.Sprite
		if sprite == "" {
			sprite = a.ID // 回退：用 id 作为精灵目录名
		}
		sc := &enemy.SpawnConfig{
			Label:       a.Label,
			Sprite:      sprite,
			HpScale:     a.HPScale,
			SpeedScale:  a.SpeedScale,
			Radius:      a.Radius,
			Boss:        a.Boss,
			RewardScale: a.RewardScale,
			// 分裂默认值（被 deathSplit 能力覆盖）。从 balance.json 读取。
			SplitScale:      config.GlobalBalance().Split.HpRatio,
			SplitHPRatio:    config.GlobalBalance().Split.HpRatio,
			SplitSpeedScale: config.GlobalBalance().Split.SpeedScale,
		}
		// 从能力配置装配行为
		for _, ref := range a.Abilities {
			def := config.ResolveEnemyAbility(ref)
			if def == nil {
				log.Printf("convertArchetypes: unknown enemy ability %q for %s", ref.Type, key)
				continue
			}
			applyEnemyAbilityToSpawnConfig(sc, def)
			sc.AbilityIDs = append(sc.AbilityIDs, def.Type)
			// 记录 potential 用于 spawn 时按波次叠加
			if p := config.ResolveEffectivePotential(ref); p != 0 {
				sc.AbilityPotentials = append(sc.AbilityPotentials, enemy.AbilityPotentialEntry{
					Type: ref.Type, Potential: p,
				})
			}
		}
		result[key] = sc
	}
	return result
}

// abilityRuntimeValue 返回敌人某个能力的运行时实际值（用于测试模式 info panel）。
func abilityRuntimeValue(e *enemy.Enemy, abilityID string) string {
	switch abilityID {
	case enemy.AbilArmorPlating:
		return fmt.Sprintf("减免=%.1f", e.ArmorFlat)
	case enemy.AbilDamageCap:
		return fmt.Sprintf("上限=%.1f", e.DamageCap)
	case enemy.AbilDamageCapPct:
		return fmt.Sprintf("上限=%.2f%%HP", e.DamageCapPercent*100)
	case enemy.AbilEvasion:
		return fmt.Sprintf("闪避=%.0f%%", e.EvasionChance*100)
	case enemy.AbilProjectileBlock:
		return fmt.Sprintf("格挡=%.0f%%", e.ProjectileBlockChance*100)
	case enemy.AbilStrengthDrain:
		return fmt.Sprintf("削弱=%.0f%%", e.StrDrainRatio*100)
	case enemy.AbilDashOnHit:
		return fmt.Sprintf("冲刺+%.0f%%", e.DashSpeedBoost*100)
	default:
		return ""
	}
}

// applyEnemyAbilityToSpawnConfig 将一个怪物能力定义映射到 SpawnConfig 的具体字段。
// 按能力类别分组：defense(格挡/护甲/闪避/坚韧) → resist(CC免疫/净化) →
// movement(隐身/冲刺/相位/传送) → offense(削强) → support(治疗/速度光环) →
// behavior(减伤/狂暴/回血) → death(分裂/召唤)。
func applyEnemyAbilityToSpawnConfig(sc *enemy.SpawnConfig, def *config.EnemyAbilityDef) {
	switch def.Type {
	// ── defense ──
	case enemy.AbilProjectileBlock:
		sc.ProjectileBlockChance = def.Base
	case enemy.AbilArmorPlating:
		sc.ArmorFlat = def.Base
	case enemy.AbilEvasion:
		sc.EvasionChance = def.Base
	case enemy.AbilDamageCap:
		sc.DamageCap = def.Base
	case enemy.AbilDamageCapPct:
		sc.DamageCapPercent = def.Base

	// ── resist ──
	case enemy.AbilCCImmune:
		sc.CCImmune = true
	case enemy.AbilSlowImmune:
		sc.SlowImmune = true
	case enemy.AbilPurge:
		sc.PurgeInterval = def.Base   // base=间隔秒数
		sc.PurgeImmuneDur = def.Param // param=免疫时间

	// ── movement ──
	case enemy.AbilStealth:
		sc.StealthDuration = def.Base
		sc.Behavior = "stealth"
	case enemy.AbilDashOnHit:
		sc.DashSpeedBoost = def.Base // base=速度提升比例
		sc.DashDuration = def.Param  // param=持续时间
		sc.DashCooldown = def.Param2 // param2=冷却(5s)
	case enemy.AbilPhaseShift:
		sc.PhaseDuration = def.Base  // base=免伤时间
		sc.PhaseCooldown = def.Param // param=冷却时间
	case enemy.AbilTeleport:
		sc.TeleportInterval = def.Param // param=传送间隔
		sc.TeleportSkip = int(def.Base) // base=跳过段数

	// ── offense ──
	case enemy.AbilStrengthDrain:
		sc.StrDrainRatio = def.Base      // base=减益比例
		sc.StrDrainInterval = def.Param  // param=间隔
		sc.StrDrainDuration = def.Param2 // param2=持续时间(6s)

	// ── support ──
	case enemy.AbilHealAura:
		sc.HealScale = def.Base
		sc.HealRadius = def.Param
		sc.HealInterval = def.Param2 // param2=治疗间隔(3s)
		sc.Behavior = "healer"
	case enemy.AbilSpeedAura:
		sc.AuraSpeedUp = def.Base
		sc.AuraRange = def.Param
		sc.Behavior = "buffer"

	// ── behavior buff ──
	case enemy.AbilDamageReduce:
		sc.DamageReduceRatio = def.Base
	case enemy.AbilBerserk:
		sc.BerserkThreshold = def.Base
		sc.BerserkSpeedScale = def.Param
	case enemy.AbilRegen:
		sc.RegenRatio = def.Base

	// ── death ──
	case enemy.AbilDeathSplit:
		sc.SplitCount = int(def.Base)
		sc.SplitHPRatio = def.Param
		sc.SplitSpeedScale = def.Param2 // param2=子体速度倍率(1.4)
		sc.Behavior = "splitter"
	case enemy.AbilDeathSpawn:
		sc.DeathSpawnCount = int(def.Base)
		sc.DeathSpawnArch = def.SpawnArch
		if sc.DeathSpawnArch == "" {
			sc.DeathSpawnArch = "normal"
		}
	}
}

// ── 战灵选择逻辑 ────────────────────────────────────

// tryStartWave 尝试开波。若战灵未选择则先弹出战灵选择面板。
// onWaveTransition 处理波次变化（prevWave → 当前 Wave）。
// 统一由 spawner.Tick 自动开波和 tryStartWave 手动开波两条路径调用。
//
// 执行顺序：
//  1. 前波完成奖励（wave bonus + perfect bonus → 加金币 + 浮字）
//  2. 发出 WaveCleared 事件（触发能力解锁/战灵通知/掉落重置/成就等）
//  3. 更新生命快照（用于下一波的完美波次检测）
//  4. 发出 WaveStarted 事件（触发音效/BGM/震屏/波次公告等）
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
				i18n.T("game.stage.perfect"), color.RGBA{255, 215, 0, 255}, 20, 2.0)
		}
		s.showNotify(result.Message)

		s.bus.Emit(event.EvtWaveCleared, event.WaveClearedPayload{
			Wave: prevWave, Perfect: perfect,
		})
	}

	// 新波开始：更新快照 + 发事件
	s.waveLivesSnapshot = s.lives
	s.bus.Emit(event.EvtWaveStarted, event.WaveStartedPayload{
		Wave: s.spawner.Wave, IsBoss: s.spawner.IsBossWave(),
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
// 覆盖层关闭后的回调：激活战灵 → 立即开第一波 → 发波次事件。
// WardenEnabled=false 时跳过选择，直接无战灵开波。
func (s *StageScene) showWardenSelect() {
	if !WardenEnabled {
		// 功能关闭：跳过战灵选择，直接无战灵开波
		s.activateWarden("")
		prevWave := s.spawner.Wave
		s.spawner.StartNextWave()
		if s.spawner.Wave > prevWave {
			s.onWaveTransition(prevWave)
		}
		return
	}
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
// key="" 表示选择"不选"（wardenUnit 保持 nil，wardenReady 仍设为 true）。
// 激活后恢复自动开波（非手动/测试模式下）。
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

	// 加载战灵配置（用于 HUD 面板显示）
	// 注：基础属性和成长参数已在 NewWarden() 中从全局缓存加载，此处仅保留 wardenCfg 引用。
	if wc := config.GlobalWardenConfig(key); wc != nil {
		cfg := *wc
		s.wardenCfg = &cfg
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
	if len(defs) == 0 {
		return nil
	}
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
	case "wideBeam":
		return color.RGBA{R: 255, G: 80, B: 80, A: 255} // red
	case "scatter":
		return color.RGBA{R: 100, G: 180, B: 255, A: 255} // ice blue
	case "spin_aoe":
		return color.RGBA{R: 255, G: 120, B: 30, A: 255} // fire orange
	default:
		return color.RGBA{R: 255, G: 240, B: 220, A: 255} // warm white
	}
}

// ─── AutoPlay 集成 ───

// SetAutoPlayer 注入自动对局驱动器（用于 cmd/autoplay 自动化测试）。
// 设为 nil 恢复手动模式。非 nil 时 Draw() 直接跳过，GPU 开销≈0。
func (s *StageScene) SetAutoPlayer(ap AutoPlayer) {
	s.autoPlayer = ap
}

// buildAutoPlaySnapshot 构建当前游戏状态的完整快照（纯值复制，无引用）。
// 包含：全局状态 + 地图信息(首帧缓存) + 敌人列表 + 塔列表 + 可建位 + 可用塔类型。
// AutoPlayer 基于此快照做决策，避免直接访问内部状态。
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
	modeNames := []string{"idle", "buildMenu", "buildPlace", "towerSel", "spawnMenu", "spawnPlace", "paused", "wardenSelect", "upgrade", "itemPanel", "itemDrag"}
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

	// 地图静态数据（仅首帧填充）
	if s.cachedMapInfo == nil {
		gm := s.gameMap
		mi := &AutoPlayMapInfo{
			CellSize:  gm.CellSize,
			Rows:      gm.Config.Rows,
			Cols:      gm.Config.Cols,
			MultiPath: gm.MultiPath,
		}
		// 复制 grid
		mi.Grid = make([][]int, len(gm.Config.Grid))
		for i, row := range gm.Config.Grid {
			mi.Grid[i] = make([]int, len(row))
			copy(mi.Grid[i], row)
		}
		// 默认路径
		mi.Waypoints = make([]AutoPlayPoint, len(gm.Waypoints))
		for i, p := range gm.Waypoints {
			mi.Waypoints[i] = AutoPlayPoint{X: p.X, Y: p.Y}
		}
		// 多路径
		for _, pe := range gm.Paths {
			ap := AutoPlayPath{ID: pe.ID, Weight: pe.Weight}
			ap.Waypoints = make([]AutoPlayPoint, len(pe.Waypoints))
			for i, p := range pe.Waypoints {
				ap.Waypoints[i] = AutoPlayPoint{X: p.X, Y: p.Y}
			}
			mi.Paths = append(mi.Paths, ap)
		}
		s.cachedMapInfo = mi
	}
	snap.MapInfo = s.cachedMapInfo

	// 敌人快照
	s.enemies.Each(func(e *enemy.Enemy) {
		snap.Enemies = append(snap.Enemies, AutoPlayEnemy{
			ID: e.ID, X: e.X, Y: e.Y,
			HP: e.HP, MaxHP: e.MaxHP, Speed: e.Speed,
			Archetype: e.Archetype, Boss: e.Boss,
			Active: e.Active, Dying: e.IsDying(),
			IsSlowed: e.IsSlowed(), IsStunned: e.IsStunned(),
			IsBurning: e.IsBurning(), IsBleeding: e.IsBleeding(),
			IsRooted:  e.IsRooted(),
			IsHit:     e.HitFlash > 0,
			BaseSpeed: e.BaseSpeed, DamageAmplify: e.GetWeakenAmplify(),
			AbilitySilenced: e.AbilitySilenced, PhaseActive: e.PhaseActive,
			ArmorFlat: e.ArmorFlat, EvasionChance: e.EvasionChance,
			DamageCap: e.DamageCap, DamageCapPct: e.DamageCapPercent,
			HealRadius: e.GetHealRadius(), BuffRadius: e.GetBufferRadius(),
			SplitCount: e.SplitCount, AbilityIDs: e.AbilityIDs,
			PathIndex: e.PathIndex, PathTotal: len(e.Path),
		})
	})

	// 已建塔快照
	s.towers.Each(func(t *tower.Tower) {
		str := 100 // default base strength
		if t.Strength != nil {
			str = int(t.Strength.Effective())
		}
		snap.Towers = append(snap.Towers, AutoPlayTower{
			Key: t.Key, Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y, Damage: t.Damage,
			Range: t.Range, Cost: t.Cost, Strength: str,
			Abilities:   t.Abilities,
			AttackStyle: string(t.AttackStyleID),
			HasTarget:   t.Target != nil,
			AttackSpeed: t.AttackSpeed, BaseDamage: t.BaseDamage,
			Kills: t.Kills,
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

// executeAutoPlayAction 执行一个 AutoPlayer 发出的操作指令。
// 支持 6 种操作：Build(建塔)/Upgrade(升级)/Sell(卖塔)/StartWave(开波)/SelectWarden(选战灵)/AddAbility(加能力)。
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
