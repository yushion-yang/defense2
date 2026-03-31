//go:build !unittest

// controller.go — AutoPlay 控制器。
// 实现 scene.AutoPlayer 接口，编排策略/录制/异常检测/截图。
// 使用 unittest build tag 排除此文件以避免 Ebitengine GLFW 初始化。
package autoplay

import (
	"log"
	"path/filepath"
	"time"

	"defense2/internal/core/telemetry"
	"defense2/internal/scene"
)

// ControllerConfig 控制器配置。
type ControllerConfig struct {
	Strategy   Strategy
	OutputDir  string // 兼容旧用法: JSON+PNG 混合输出 (当 JSONDir/PNGDir 为空时使用)
	JSONDir    string // JSON 报告输出目录 (纯 JSON)
	PNGDir     string // 截图输出目录 (纯 PNG)
	SessionID  string
	MapID      string
	Difficulty string
	Warden     string
}

// Controller 自动对局控制器，实现 scene.AutoPlayer。
type Controller struct {
	strategy      Strategy
	recorder      *Recorder
	anomaly       *AnomalyDetector
	screenshotter *Screenshotter

	jsonDir   string // JSON 报告写入目录
	pngDir    string // 截图写入目录
	sessionID string

	prevWave    int
	prevLives   int
	gameStarted bool
	done        bool
	startTime   time.Time
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
		jsonDir:       jsonDir,
		pngDir:        pngDir,
		sessionID:     cfg.SessionID,
		startTime:     time.Now(),
	}
}

// OnUpdate 每帧调用，返回要执行的操作。实现 scene.AutoPlayer。
func (c *Controller) OnUpdate(snap scene.AutoPlaySnapshot) []scene.AutoPlayAction {
	state := snapshotToGameState(snap)

	// 首帧初始化
	if !c.gameStarted {
		c.gameStarted = true
		c.prevLives = state.Lives
		telemetry.T.Reset() // 每局开始清除遥测数据
		c.strategy.Init(state)
		c.screenshotter.RequestStart()
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
		// 上一波结束
		if c.prevWave > 0 {
			c.recorder.OnWaveEnd(c.prevWave, state)
		}
		c.recorder.OnWaveStart(state.Wave, state)

		// 截图触发
		if state.Wave%5 == 0 {
			c.screenshotter.RequestWave(state.Wave)
		}
		if state.Wave%5 == 0 && state.Wave > 0 {
			c.screenshotter.RequestBossWave(state.Wave)
		}
	}

	// 泄漏检测（生命减少）
	if state.Lives < c.prevLives {
		c.screenshotter.RequestLeak(state.Wave, state.Tick)
	}
	c.prevLives = state.Lives
	c.prevWave = state.Wave

	// 录制 tick
	c.recorder.OnTick(state, 1.0/60.0)
	c.recorder.TickDPS(1.0 / 60.0)

	// 策略决策
	actions := c.strategy.Decide(state)

	// 记录操作审计（下帧验证结果）
	for _, a := range actions {
		switch a.Type {
		case ActionBuild:
			c.anomaly.RecordBuildAction(state.Tick, a.Cell.Row, a.Cell.Col, a.TowerKey)
		case ActionUpgrade:
			// 找当前强度
			for _, t := range state.Towers {
				if t.Row == a.Row && t.Col == a.Col {
					c.anomaly.RecordUpgradeAction(state.Tick, a.Row, a.Col, t.Strength)
					break
				}
			}
		}
	}

	// 转换为 scene 包的 Action 类型
	return actionsToSceneActions(actions)
}

// OnGameEnd 游戏结束时调用。实现 scene.AutoPlayer。
func (c *Controller) OnGameEnd(snap scene.AutoPlaySnapshot, won bool) {
	if c.done {
		return
	}
	c.done = true

	state := snapshotToGameState(snap)
	c.screenshotter.RequestResult()

	// 最后一波
	if c.prevWave > 0 {
		c.recorder.OnWaveEnd(c.prevWave, state)
	}

	// 生成报告
	record := c.recorder.Finalize(state, c.anomaly.Anomalies(), c.screenshotter.CapturedFiles())
	if err := WriteJSON(record, c.jsonDir); err != nil {
		log.Printf("report write error: %v", err)
	} else {
		log.Printf("[DONE] session=%s result=%s waves=%d/%d kills=%d anomalies=%d",
			c.sessionID, record.Result, record.WavesSurvived, record.TotalWaves,
			record.TotalKills, len(record.Anomalies))
	}
}

// ScreenshotRequested 返回下一个截图路径。实现 scene.AutoPlayer。
func (c *Controller) ScreenshotRequested() string {
	fname := c.screenshotter.NextPending()
	if fname == "" {
		return ""
	}
	return filepath.Join(c.pngDir, fname)
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
		})
	}

	for _, t := range snap.Towers {
		state.Towers = append(state.Towers, TowerInfo{
			Key: t.Key, Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y, Damage: t.Damage,
			Range: t.Range, Cost: t.Cost, Strength: t.Strength,
			Abilities: t.Abilities, SkillName: t.SkillName,
			AttackStyle: t.AttackStyle,
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
			TowerKey:   a.TowerKey,
			Row:        a.Row,
			Col:        a.Col,
			WardenKey:  a.WardenKey,
			EventIndex: a.EventIndex,
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
		case ActionChooseEvent:
			sa.Type = scene.APActionChooseEvent
		case ActionAssignSkill:
			sa.Type = scene.APActionAssignSkill
			sa.SkillName = a.SkillName
			sa.SkillToWarden = a.SkillToWarden
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
