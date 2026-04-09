//go:build !unittest

// controller.go — AutoPlay 控制器。
// 实现 scene.AutoPlayer 接口，编排策略/录制/异常检测/截图。
// 使用 unittest build tag 排除此文件以避免 Ebitengine GLFW 初始化。
package autoplay

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"defense2/internal/config"
	"defense2/internal/core/telemetry"
	"defense2/internal/scene"
)

// ControllerConfig 控制器配置。
type ControllerConfig struct {
	Strategy   Strategy
	OutputDir  string
	JSONDir    string
	PNGDir     string
	SessionID  string
	MapID      string
	Difficulty string
	Warden     string
	Seed int64 // 随机种子（记录到报告，用于复现）
}

// Controller 自动对局控制器，实现 scene.AutoPlayer。
type Controller struct {
	strategy      Strategy
	recorder      *Recorder
	anomaly       *AnomalyDetector
	screenshotter *Screenshotter
	visualTracker *VisualTracker

	jsonDir   string
	pngDir    string
	sessionID string
	seed      int64

	prevWave    int
	prevLives   int
	prevKills   int  // 上一帧累计击杀数（用于增量检测）
	prevMode    int  // 上一帧交互模式
	modeDelay   bool // 模式切换延迟标志：先截图，下帧再操作
	gameStarted bool
	done        bool
	resultDrawn bool // 结果画面已截图
	startTime   time.Time

	// HUD 截图追踪（每种模式只截一次）
	modeCaptured map[int]bool
	firstTowerCaptured bool
}

// NewController 创建自动对局控制器。
func NewController(cfg ControllerConfig) *Controller {
	// 分离模式: JSONDir/PNGDir 分别存放; 兼容模式: 全部放 OutputDir
	jsonDir := filepath.Join(cfg.OutputDir, cfg.SessionID)
	pngDir := jsonDir
	if cfg.JSONDir != "" {
		jsonDir = filepath.Join(cfg.JSONDir, cfg.SessionID)
	}
	if cfg.PNGDir != "" {
		pngDir = filepath.Join(cfg.PNGDir, cfg.SessionID)
	}
	return &Controller{
		strategy:      cfg.Strategy,
		recorder:      NewRecorder(cfg.SessionID, cfg.Strategy.Name(), cfg.MapID, cfg.Difficulty, cfg.Warden),
		anomaly:       NewAnomalyDetector(),
		screenshotter: NewScreenshotter(pngDir),
		visualTracker: nil, // 延迟初始化
		jsonDir:       jsonDir,
		pngDir:        pngDir,
		sessionID:     cfg.SessionID,
		seed:          cfg.Seed,
		startTime:     time.Now(),
		modeCaptured:  make(map[int]bool),
	}
}

// 交互模式名称（与 scene.interactMode 对应）。
var modeNames = map[int]string{
	0: "idle", 1: "buildMenu", 2: "buildPlace", 3: "towerSel",
	4: "spawnMenu", 5: "spawnPlace", 7: "paused", 8: "wardenSelect",
}

// hudModes 需要截图的 HUD 模式（进入时截一张以验证面板渲染正确）。
var hudModes = map[int]bool{
	1: true, // buildMenu
	3: true, // towerSel
	7: true, // paused
	8: true, // wardenSelect
}

// maxSessionTicks 单局最大 tick 数，超过强制结束防止死循环。
const maxSessionTicks = 30000 // ~8 分钟 @60TPS

// OnUpdate 每帧调用，返回要执行的操作。实现 scene.AutoPlayer。
func (c *Controller) OnUpdate(snap scene.AutoPlaySnapshot) []scene.AutoPlayAction {
	state := snapshotToGameState(snap)

	// 安全超时: 防止游戏逻辑 bug 导致永不结束
	if state.Tick > maxSessionTicks && !c.done {
		log.Printf("[TIMEOUT] session=%s tick=%d exceeded max %d, forcing end", c.sessionID, state.Tick, maxSessionTicks)
		c.OnGameEnd(snap, false)
		return nil
	}

	// 首帧初始化
	if !c.gameStarted {
		c.gameStarted = true
		c.prevLives = state.Lives
		c.prevMode = state.InteractMode
		telemetry.T.Reset()
		c.visualTracker = NewVisualTracker(c.screenshotter)
		c.strategy.Init(state)
		c.screenshotter.RequestStart()
	}

	// ── HUD 模式截图：检测模式切换，先截图再操作 ──
	if state.InteractMode != c.prevMode {
		if hudModes[state.InteractMode] && !c.modeCaptured[state.InteractMode] {
			// 新进入一个 HUD 模式 → 请求截图，本帧不执行操作
			name := modeNames[state.InteractMode]
			c.screenshotter.RequestCapture(fmt.Sprintf("hud_%s_%d.png", name, state.Tick))
			c.modeCaptured[state.InteractMode] = true
			c.modeDelay = true
		}
		c.prevMode = state.InteractMode
	}
	// 模式延迟：上帧刚请求截图，本帧 Draw 会渲染，下帧再操作
	if c.modeDelay {
		c.modeDelay = false
		return nil // 空操作，让 Draw 有机会截到 HUD
	}

	// 异常检测
	elapsed := time.Since(c.startTime)
	updateMs := float64(elapsed.Milliseconds()) / float64(max(state.Tick, 1))
	anomalies := c.anomaly.Check(state, updateMs)
	for _, a := range anomalies {
		log.Printf("[ANOMALY] tick=%d type=%s severity=%s detail=%s",
			a.Tick, a.Type, a.Severity.String(), a.Detail)
		c.screenshotter.RequestAnomaly(a.Type, a.Tick)
	}

	// 波次变化检测
	if state.Wave > c.prevWave && state.Wave > 0 {
		if c.prevWave > 0 {
			c.recorder.OnWaveEnd(c.prevWave, state)
		}
		c.recorder.OnWaveStart(state.Wave, state)

		if state.Wave > 0 && state.Wave%5 == 0 {
			c.screenshotter.RequestBossWave(state.Wave)
		}
	}

	// 泄漏检测
	if state.Lives < c.prevLives {
		c.screenshotter.RequestLeak(state.Wave, state.Tick)
	}
	c.prevLives = state.Lives
	c.prevWave = state.Wave

	// 视觉内容追踪：检测首次出现的塔/敌人/攻击方式/状态效果等
	if c.visualTracker != nil {
		c.visualTracker.Check(state)
	}

	// 波次公告截图（波次变化后立即请求，公告动画正在显示）
	if state.Wave > c.prevWave && state.Wave > 1 && state.Wave <= 3 {
		// 只对前几波截公告（避免重复）
		c.screenshotter.RequestCapture(fmt.Sprintf("wave_announce_%d.png", state.Wave))
	}

	// 击杀增量同步 (BUG-001: total_kills always 0)
	if state.TotalKills > c.prevKills {
		delta := state.TotalKills - c.prevKills
		for i := 0; i < delta; i++ {
			c.recorder.OnKill()
		}
		c.prevKills = state.TotalKills
	}

	// 理论 DPS 采样 (BUG-002: DPS snapshots all zero)
	// 对有目标的塔累加 Damage 作为瞬时 DPS 近似
	for _, t := range state.Towers {
		if t.HasTarget {
			c.recorder.RecordDamage(t.Damage)
		}
	}

	// 录制
	c.recorder.OnTick(state, 1.0/60.0)
	c.recorder.TickDPS(1.0 / 60.0)

	// 策略决策
	actions := c.strategy.Decide(state)

	// 记录操作审计（下帧验证结果）
	for _, a := range actions {
		switch a.Type {
		case ActionBuild:
			c.anomaly.RecordBuildAction(state.Tick, a.Cell.Row, a.Cell.Col, a.TowerKey)
			// BUG-003: 记录建塔花费到波次统计
			for _, d := range state.TowerDefs {
				if d.Key == a.TowerKey {
					c.recorder.OnTowerBuilt(d.Cost)
					break
				}
			}
		case ActionUpgrade:
			// 找当前强度
			for _, t := range state.Towers {
				if t.Row == a.Row && t.Col == a.Col {
					c.anomaly.RecordUpgradeAction(state.Tick, a.Row, a.Col, t.Strength)
					break
				}
			}
			// 记录升级花费到波次统计
			c.recorder.OnTowerBuilt(config.GlobalBalance().Tower.StrengthBuyCost)
		}
	}

	// 转换为 scene 包的 Action 类型
	return actionsToSceneActions(actions)
}

// OnGameEnd 游戏结束时调用。实现 scene.AutoPlayer。
// 第一次调用：请求结果截图 + 写报告，但 Done() 返回 false（让 Draw 截图）。
// 第二次调用：设置 done=true，触发 Termination。
func (c *Controller) OnGameEnd(snap scene.AutoPlaySnapshot, won bool) {
	if c.done {
		return
	}

	if !c.resultDrawn {
		// 第一次：请求截图 + 写报告
		c.resultDrawn = true

		state := snapshotToGameState(snap)
		c.screenshotter.RequestResult()

		if c.prevWave > 0 {
			c.recorder.OnWaveEnd(c.prevWave, state)
		}

		record := c.recorder.Finalize(state, c.anomaly.Anomalies(), c.screenshotter.CapturedFiles())
		record.Seed = c.seed
		if err := WriteJSON(record, c.jsonDir); err != nil {
			log.Printf("report write error: %v", err)
		} else {
			log.Printf("[DONE] session=%s result=%s waves=%d/%d kills=%d anomalies=%d",
				c.sessionID, record.Result, record.WavesSurvived, record.TotalWaves,
				record.TotalKills, len(record.Anomalies))
		}

		// 生成视觉审查 manifest（截图 + 检查清单配对）
		entries := BuildReviewEntries(c.screenshotter.CapturedFiles())
		if err := WriteVisualReview(entries, c.pngDir); err != nil {
			log.Printf("visual review write error: %v", err)
		} else if len(entries) > 0 {
			log.Printf("[REVIEW] %d screenshots with checklist -> visual_review.md", len(entries))
		}
		return // 不设 done，让 Draw 有机会截到结果画面
	}

	// 第二次：截图已完成，可以退出
	c.done = true
}

// ScreenshotRequested 返回下一个截图路径。实现 scene.AutoPlayer。
func (c *Controller) ScreenshotRequested() string {
	fname := c.screenshotter.NextPending()
	if fname == "" {
		return ""
	}
	return filepath.Join(c.pngDir, fname)
}

// HasPendingScreenshot 检查是否有待截图请求。实现 scene.AutoPlayer。
func (c *Controller) HasPendingScreenshot() bool {
	return c.screenshotter.HasPending()
}

// Done 返回是否已完成。实现 scene.AutoPlayer。
func (c *Controller) Done() bool {
	return c.done
}

// snapshotToGameState 将 scene.AutoPlaySnapshot 转换为 autoplay.GameState。
func snapshotToGameState(snap scene.AutoPlaySnapshot) *GameState {
	state := &GameState{
		Tick:            snap.Tick,
		Gold:            snap.Gold,
		Lives:           snap.Lives,
		Wave:            snap.Wave,
		MaxWaves:        snap.MaxWaves,
		WaveActive:      snap.WaveActive,
		GameOver:        snap.GameOver,
		Victory:         snap.Victory,
		WardenReady:     snap.WardenReady,
		InteractMode:    snap.InteractMode,
		WavesCleared:    snap.WavesCleared,
		TotalKills:      snap.TotalKills,
		TotalLeaked:     snap.TotalLeaked,
		EnemyPoolCount:  snap.EnemyPoolCount,
		ProjectileCount: snap.ProjectileCount,
		TowerCount:      snap.TowerCount,
		WardenX:         snap.WardenX,
		WardenY:         snap.WardenY,
		MapPixelW:       snap.MapPixelW,
		MapPixelH:       snap.MapPixelH,
		GameSpeed:       snap.GameSpeed,
		Telemetry:       snap.Telemetry,
	}

	for _, e := range snap.Enemies {
		state.Enemies = append(state.Enemies, EnemyInfo{
			ID: e.ID, X: e.X, Y: e.Y,
			HP: e.HP, MaxHP: e.MaxHP, Speed: e.Speed,
			Archetype: e.Archetype, Boss: e.Boss,
			Active: e.Active, Dying: e.Dying,
			IsSlowed: e.IsSlowed, IsStunned: e.IsStunned,
			IsBurning: e.IsBurning, IsBleeding: e.IsBleeding,
			IsRooted: e.IsRooted,
			IsHit: e.IsHit,
		})
	}

	for _, t := range snap.Towers {
		state.Towers = append(state.Towers, TowerInfo{
			Key: t.Key, Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y, Damage: t.Damage,
			Range: t.Range, Cost: t.Cost, Strength: t.Strength,
			Abilities: t.Abilities,
			AttackStyle: t.AttackStyle,
			HasTarget: t.HasTarget,
		})
	}

	for _, c := range snap.BuildCells {
		state.BuildCells = append(state.BuildCells, Cell{
			Row: c.Row, Col: c.Col, X: c.X, Y: c.Y,
		})
	}

	for _, d := range snap.TowerDefs {
		state.TowerDefs = append(state.TowerDefs, TowerDefInfo{
			Key: d.Key, Cost: d.Cost, Range: d.Range,
			Damage: d.Damage, Index: d.Index,
		})
	}

	return state
}

// actionsToSceneActions 将 autoplay.Action 转换为 scene.AutoPlayAction。
func actionsToSceneActions(actions []Action) []scene.AutoPlayAction {
	if len(actions) == 0 {
		return nil
	}
	result := make([]scene.AutoPlayAction, 0, len(actions))
	for _, a := range actions {
		sa := scene.AutoPlayAction{
			TowerKey:  a.TowerKey,
			Row:       a.Row,
			Col:       a.Col,
			WardenKey: a.WardenKey,
		}
		switch a.Type {
		case ActionBuild:
			sa.Type = scene.APActionBuild
			sa.Row = a.Cell.Row
			sa.Col = a.Cell.Col
		case ActionUpgrade:
			sa.Type = scene.APActionUpgrade
		case ActionSell:
			sa.Type = scene.APActionSell
		case ActionStartWave:
			sa.Type = scene.APActionStartWave
		case ActionSelectWarden:
			sa.Type = scene.APActionSelectWarden
		case ActionAddAbility:
			sa.Type = scene.APActionAddAbility
			sa.AbilityName = a.AbilityName
		default:
			continue
		}
		result = append(result, sa)
	}
	return result
}

// SessionID 返回会话 ID（供外部使用）。
func (c *Controller) SessionID() string {
	return c.sessionID
}

// Note: FormatSessionID moved to strategy.go to avoid build tag dependency.
