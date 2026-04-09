//go:build !unittest

// controller.go — AutoPlay 控制器。
// 实现 scene.AutoPlayer 接口，编排策略/录制/异常检测。
// 使用 unittest build tag 排除此文件以避免 Ebitengine GLFW 初始化。
package autoplay

import (
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
	SessionID  string
	MapID      string
	Difficulty string
	Warden     string
	Seed       int64 // 随机种子（记录到报告，用于复现）
}

// Controller 自动对局控制器，实现 scene.AutoPlayer。
type Controller struct {
	strategy      Strategy
	recorder      *Recorder
	anomaly       *AnomalyDetector
	assertChecker *AssertionChecker

	jsonDir   string
	sessionID string
	seed      int64

	prevWave    int
	prevLives   int
	prevKills   int // 上一帧累计击杀数（用于增量检测）
	gameStarted bool
	done        bool
	resultDrawn bool // 结果画面已绘制
	startTime   time.Time
}

// NewController 创建自动对局控制器。
func NewController(cfg ControllerConfig) *Controller {
	jsonDir := filepath.Join(cfg.OutputDir, cfg.SessionID)
	if cfg.JSONDir != "" {
		jsonDir = filepath.Join(cfg.JSONDir, cfg.SessionID)
	}
	return &Controller{
		strategy:  cfg.Strategy,
		recorder:  NewRecorder(cfg.SessionID, cfg.Strategy.Name(), cfg.MapID, cfg.Difficulty, cfg.Warden),
		anomaly:   NewAnomalyDetector(),
		jsonDir:   jsonDir,
		sessionID: cfg.SessionID,
		seed:      cfg.Seed,
		startTime: time.Now(),
	}
}

// SetAssertions 初始化断言检查器（用于能力测试场景）。
func (c *Controller) SetAssertions(assertions []Assertion) {
	if len(assertions) > 0 {
		c.assertChecker = NewAssertionChecker(assertions)
	}
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
		telemetry.T.Reset()
		c.strategy.Init(state)
	}

	// 异常检测
	elapsed := time.Since(c.startTime)
	updateMs := float64(elapsed.Milliseconds()) / float64(max(state.Tick, 1))
	anomalies := c.anomaly.Check(state, updateMs)
	for _, a := range anomalies {
		log.Printf("[ANOMALY] tick=%d type=%s severity=%s detail=%s",
			a.Tick, a.Type, a.Severity.String(), a.Detail)
	}

	// 断言检查（能力测试场景）
	if c.assertChecker != nil {
		c.assertChecker.Check(state)
	}

	// 波次变化检测
	if state.Wave > c.prevWave && state.Wave > 0 {
		if c.prevWave > 0 {
			c.recorder.OnWaveEnd(c.prevWave, state)
		}
		c.recorder.OnWaveStart(state.Wave, state)
	}

	// 泄漏检测
	c.prevLives = state.Lives
	c.prevWave = state.Wave

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
func (c *Controller) OnGameEnd(snap scene.AutoPlaySnapshot, won bool) {
	if c.done {
		return
	}

	state := snapshotToGameState(snap)

	if c.prevWave > 0 {
		c.recorder.OnWaveEnd(c.prevWave, state)
	}

	// 断言结果收集
	if c.assertChecker != nil {
		c.recorder.Assertions = c.assertChecker.Finalize(state.Tick)
	}

	record := c.recorder.Finalize(state, c.anomaly.Anomalies())
	record.Seed = c.seed
	if err := WriteJSON(record, c.jsonDir); err != nil {
		log.Printf("report write error: %v", err)
	} else {
		log.Printf("[DONE] session=%s result=%s waves=%d/%d kills=%d anomalies=%d",
			c.sessionID, record.Result, record.WavesSurvived, record.TotalWaves,
			record.TotalKills, len(record.Anomalies))
	}

	c.done = true
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
			IsRooted:  e.IsRooted,
			IsHit:     e.IsHit,
			BaseSpeed: e.BaseSpeed, DamageAmplify: e.DamageAmplify,
			AbilitySilenced: e.AbilitySilenced, PhaseActive: e.PhaseActive,
			ArmorFlat: e.ArmorFlat, EvasionChance: e.EvasionChance,
			DamageCap: e.DamageCap, DamageCapPct: e.DamageCapPct,
			HealRadius: e.HealRadius, BuffRadius: e.BuffRadius,
			SplitCount: e.SplitCount, AbilityIDs: e.AbilityIDs,
		})
	}

	for _, t := range snap.Towers {
		state.Towers = append(state.Towers, TowerInfo{
			Key: t.Key, Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y, Damage: t.Damage,
			Range: t.Range, Cost: t.Cost, Strength: t.Strength,
			Abilities:   t.Abilities,
			AttackStyle: t.AttackStyle,
			HasTarget:   t.HasTarget,
			AttackSpeed: t.AttackSpeed, BaseDamage: t.BaseDamage,
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
			sa.Row = a.Row
			sa.Col = a.Col
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
