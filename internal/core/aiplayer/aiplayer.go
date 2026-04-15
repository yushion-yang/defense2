// aiplayer.go — AI 玩家主结构体。
//
// 职责：编排决策引擎、行动队列、精灵、气泡，驱动 AI 的完整行为循环。
// 关联：由 stage.go 每帧调用 Tick()，由 hud/ai_overlay.go 渲染。
// 设计：零 core 依赖（除自身子模块），通过 StageOps 接口与游戏交互。
package aiplayer

import (
	"defense2/internal/core/aiplayer/llm"
	"math/rand"
)

// StageOps AI 可用的游戏操作接口。
// 由 StageScene 实现，避免 AI 直接依赖 scene 包。
type StageOps interface {
	BuildTowerForAI(key string, row, col, ownerID int) bool
	UpgradeTowerForAI(row, col int) bool
	SellTowerForAI(row, col int) bool
	TowerCost(key string) int
	StrengthBuyCost() int
	StartWave() bool              // 开始下一波
	SelectWarden(key string) bool // 选择战灵
}

// ZoneProvider 区域查询接口。Zone 和 CoopZone 都实现此接口。
type ZoneProvider interface {
	OwnerOf(row, col int) int
}

// Config AI 玩家配置。
type Config struct {
	ZoneProvider ZoneProvider // 区域查询（Zone 或 CoopZone）
	OwnerID      int          // 此 AI 的 owner ID（1..N-1）
	StartGold    int
	Ops          StageOps
	CellSize     int     // 网格像素尺寸
	SpawnX       float64 // 精灵初始 X（0=自动计算）
	SpawnY       float64 // 精灵初始 Y（0=自动计算）
	LLMEnabled   bool    // 是否启用 LLM 弹幕（或 ANTHROPIC_API_KEY 存在时自动启用）
}

// AIPlayer AI 玩家。
type AIPlayer struct {
	gold         int
	ownerID      int // 此 AI 的 owner ID（1..N-1）
	personality  Personality
	zoneProvider ZoneProvider
	ops          StageOps
	sprite       *Sprite
	bubble       *BubbleManager
	actions      *ActionQueue
	engine       *DecisionEngine
	dialogue     *DialogueBank
	ping         *PingState
	llmConn      *llm.Connector // LLM 弹幕连接器（nil 表示禁用）
	lastAction   string         // 上一次执行的行动标签（供 LLM prompt 使用）

	// 决策节奏
	decisionTimer    float64
	decisionInterval float64
}

// New 创建 AI 玩家。
func New(cfg Config) *AIPlayer {
	cellSize := cfg.CellSize
	if cellSize <= 0 {
		cellSize = 60
	}

	spawnX, spawnY := cfg.SpawnX, cfg.SpawnY
	if spawnX == 0 && spawnY == 0 {
		// fallback: 地图中心偏右
		spawnX = float64(18 * cellSize)
		spawnY = float64(6 * cellSize)
	}

	ownerID := cfg.OwnerID
	if ownerID == 0 {
		ownerID = 1 // 默认 AI owner = 1
	}

	personality := RandomPersonality()

	ap := &AIPlayer{
		gold:         cfg.StartGold,
		ownerID:      ownerID,
		personality:  personality,
		zoneProvider: cfg.ZoneProvider,
		ops:          cfg.Ops,
		sprite:       NewSprite(spawnX, spawnY),
		bubble:       NewBubbleManager(),
		actions:      NewActionQueue(),
		engine:       NewDecisionEngine(),
		dialogue:     DefaultDialogueBank(),
		ping:         NewPingState(),
		lastAction:   "idle",
	}
	ap.engine.SetPersonality(personality)
	ap.rollNextInterval()

	// LLM 弹幕连接器：显式启用或环境变量 ANTHROPIC_API_KEY 存在时创建
	llmCfg := llm.DefaultConfig()
	if cfg.LLMEnabled {
		llmCfg.Enabled = true
	}
	if llmCfg.Enabled {
		ap.llmConn = llm.NewConnector(llmCfg)
	}

	if cfg.Ops != nil {
		ap.engine.SetStrengthBuyCost(cfg.Ops.StrengthBuyCost())
	}

	return ap
}

// NewWithPersonality 创建指定个性的 AI 玩家（测试用）。
func NewWithPersonality(cfg Config, p Personality) *AIPlayer {
	ap := New(cfg)
	ap.personality = p
	ap.engine.SetPersonality(p)
	ap.rollNextInterval()
	return ap
}

// OwnerID 返回此 AI 的 owner ID。
func (ap *AIPlayer) OwnerID() int { return ap.ownerID }

// Personality 返回 AI 个性（只读）。
func (ap *AIPlayer) Personality() Personality { return ap.personality }

// ── 只读访问器 ──

// Gold 返回当前金币。
func (ap *AIPlayer) Gold() int { return ap.gold }

// AddGold 增加金币（击杀收入等）。
func (ap *AIPlayer) AddGold(amount int) { ap.gold += amount }

// SpriteX 精灵 X 坐标。
func (ap *AIPlayer) SpriteX() float64 { return ap.sprite.X() }

// SpriteY 精灵渲染 Y 坐标（含 bob）。
func (ap *AIPlayer) SpriteY() float64 { return ap.sprite.DrawY() }

// SpriteState 精灵状态。
func (ap *AIPlayer) SpriteState() SpriteState { return ap.sprite.State() }

// BubbleVisible 气泡是否可见。
func (ap *AIPlayer) BubbleVisible() bool { return ap.bubble.Visible() }

// BubbleText 气泡文案。
func (ap *AIPlayer) BubbleText() string { return ap.bubble.Text() }

// BubbleAlpha 气泡透明度。
func (ap *AIPlayer) BubbleAlpha() float64 { return ap.bubble.Alpha() }

// SendPing 玩家向 AI 发送 ping 建议（冷却中返回 false）。
func (ap *AIPlayer) SendPing(row, col int, x, y float64) bool {
	return ap.ping.Send(row, col, x, y)
}

// PingOnCooldown 返回 ping 是否在冷却中。
func (ap *AIPlayer) PingOnCooldown() bool {
	return ap.ping.OnCooldown()
}

// ── 主循环 ──

// Tick 每帧更新。
//  1. 行动队列处理
//  2. 精灵更新
//  3. 气泡更新
//  4. Ping 冷却递减
//  5. 引擎波间延迟递减
//  6. Ping 响应（优先于正常决策）
//  7. 决策评估（按间隔）
func (ap *AIPlayer) Tick(dt float64, snap AISnapshot) {
	ap.actions.Tick(dt)
	ap.sprite.Tick(dt)
	ap.bubble.Tick(dt)
	ap.ping.Tick(dt)
	ap.engine.TickWaveDelay(dt)

	// 有待执行行动时不做新决策
	if ap.actions.Busy() {
		return
	}

	// ── Ping 响应（优先于正常决策循环）──
	// 收到 ping 时立即响应，替代本轮正常决策。
	if ap.ping.HasPending() {
		ap.handlePing(snap)
		return
	}

	// 决策节奏控制
	ap.decisionTimer -= dt
	if ap.decisionTimer > 0 {
		return
	}
	ap.rollNextInterval()

	// 注入 AI 当前金币到快照
	snap.Gold = ap.gold

	// 过滤出 AI 区域的可建造格子和塔
	snap.BuildCells = ap.filterAICells(snap.BuildCells)
	snap.Towers = ap.filterAITowers(snap.Towers)

	decision := ap.engine.Evaluate(snap)
	ap.executeDecision(decision)

	// LLM 弹幕：异步获取自然语言评论，不阻塞游戏循环
	ap.tickLLM(dt, snap)
}

// ── 内部方法 ──

func (ap *AIPlayer) filterAICells(cells []AICell) []AICell {
	var result []AICell
	for _, c := range cells {
		if ap.zoneProvider != nil && ap.zoneProvider.OwnerOf(c.Row, c.Col) == ap.ownerID {
			result = append(result, c)
		}
	}
	return result
}

func (ap *AIPlayer) filterAITowers(towers []AITower) []AITower {
	var result []AITower
	for _, t := range towers {
		if t.Owner == ap.ownerID {
			result = append(result, t)
		}
	}
	return result
}

func (ap *AIPlayer) executeDecision(d Decision) {
	// 记录行动标签供 LLM prompt 使用
	switch d.Type {
	case DecisionBuild:
		ap.lastAction = "built tower"
	case DecisionUpgrade:
		ap.lastAction = "upgraded"
	case DecisionStartWave:
		ap.lastAction = "started wave"
	case DecisionSelectWarden:
		ap.lastAction = "selected warden"
	default:
		ap.lastAction = "idle"
	}

	switch d.Type {
	case DecisionIdle:
		// 拟人化：8% 概率显示思考气泡
		if rand.Float64() < 0.08 {
			ap.bubble.Show(ap.dialogue.Random("idle"), BubbleThink, 1.5)
		}

	case DecisionBuild:
		ap.sprite.MoveTo(d.CenterX, d.CenterY)
		ap.bubble.Show(ap.dialogue.Random("build_thinking"), BubbleThink, 1.5)

		cost := 50
		if ap.ops != nil {
			cost = ap.ops.TowerCost(d.TowerKey)
		}
		towerKey := d.TowerKey
		row, col := d.Row, d.Col

		ap.actions.Enqueue(DelayedAction{
			Delay: 1.0 + rand.Float64()*0.5,
			Label: "build",
			Execute: func() {
				if ap.gold < cost {
					ap.bubble.Show(ap.dialogue.Random("low_gold"), BubbleEmotion, 1.0)
					return
				}
				if ap.ops != nil && ap.ops.BuildTowerForAI(towerKey, row, col, ap.ownerID) {
					ap.gold -= cost
					ap.bubble.Show(ap.dialogue.Random("build_done"), BubbleAction, 1.0)
				}
			},
		})

	case DecisionUpgrade:
		row, col := d.Row, d.Col
		ap.bubble.Show(ap.dialogue.Random("upgrade_thinking"), BubbleThink, 1.0)

		ap.actions.Enqueue(DelayedAction{
			Delay: 0.5 + rand.Float64()*0.5,
			Label: "upgrade",
			Execute: func() {
				cost := 10
				if ap.ops != nil {
					cost = ap.ops.StrengthBuyCost()
				}
				if ap.gold < cost {
					ap.bubble.Show(ap.dialogue.Random("low_gold"), BubbleEmotion, 1.0)
					return
				}
				if ap.ops != nil && ap.ops.UpgradeTowerForAI(row, col) {
					ap.gold -= cost
				}
			},
		})

	case DecisionStartWave:
		ap.bubble.Show(ap.dialogue.Random("wave_start"), BubbleAction, 1.2)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.3 + rand.Float64()*0.3,
			Label: "start_wave",
			Execute: func() {
				if ap.ops != nil {
					ap.ops.StartWave()
				}
			},
		})

	case DecisionSelectWarden:
		wardenKey := d.WardenKey
		ap.bubble.Show(ap.dialogue.Random("warden_select"), BubbleThink, 1.5)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.5 + rand.Float64()*0.5,
			Label: "select_warden",
			Execute: func() {
				if ap.ops != nil {
					ap.ops.SelectWarden(wardenKey)
				}
			},
		})
	}
}

// handlePing 处理玩家 ping 请求。
// 评分公式: compliance * 0.7 + positionScore * 0.3 > 0.4 → 同意。
// 同意时移动精灵到 ping 位置，排队造塔行动。
// 拒绝时显示拒绝气泡。
func (ap *AIPlayer) handlePing(snap AISnapshot) {
	req := ap.ping.Consume()
	if req == nil {
		return
	}

	// 先显示"收到"气泡
	ap.bubble.Show(ap.dialogue.Random("ping_received"), BubbleReply, 0.8)

	// 计算位置评分（复用引擎的建造评分逻辑）
	posScore := 0.0
	cell := AICell{Row: req.Row, Col: req.Col, X: req.X, Y: req.Y}
	towerRange := 100.0
	if len(snap.TowerDefs) > 0 {
		towerRange = snap.TowerDefs[0].Range
		if towerRange <= 0 {
			towerRange = 100
		}
	}
	posScore = ap.engine.scoreBuildCell(cell, snap, towerRange)

	// 综合评分: compliance 权重 70% + 位置评分权重 30%
	totalScore := ap.personality.Compliance*0.7 + posScore*0.3

	if totalScore > 0.4 {
		// 同意: 移动精灵 + 排队造塔
		ap.sprite.MoveTo(req.X, req.Y)

		towerKey := "basic"
		cost := 50
		if len(snap.TowerDefs) > 0 {
			towerKey = snap.TowerDefs[0].Key
			if ap.ops != nil {
				cost = ap.ops.TowerCost(towerKey)
			}
		}
		row, col := req.Row, req.Col

		ap.actions.Enqueue(DelayedAction{
			Delay: 0.8 + rand.Float64()*0.5,
			Label: "ping_build",
			Execute: func() {
				if ap.gold < cost {
					ap.bubble.Show(ap.dialogue.Random("low_gold"), BubbleEmotion, 1.0)
					return
				}
				if ap.ops != nil && ap.ops.BuildTowerForAI(towerKey, row, col, ap.ownerID) {
					ap.gold -= cost
					ap.bubble.Show(ap.dialogue.Random("ping_agree"), BubbleReply, 1.5)
				}
			},
		})
	} else {
		// 拒绝: 显示拒绝气泡
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.5 + rand.Float64()*0.3,
			Label: "ping_refuse",
			Execute: func() {
				ap.bubble.Show(ap.dialogue.Random("ping_refuse"), BubbleReply, 2.0)
			},
		})
	}
}

// rollNextInterval 根据个性设置下次决策间隔。
// 高 Reaction(1.0) → 1.5~2.5s，低 Reaction(0.0) → 3.0~4.5s。
func (ap *AIPlayer) rollNextInterval() {
	// 线性插值: reaction=0 → base=3.0,jitter=1.5; reaction=1 → base=1.5,jitter=1.0
	reaction := ap.personality.Reaction
	base := 3.0 - reaction*1.5   // 1.5 ~ 3.0
	jitter := 1.5 - reaction*0.5 // 1.0 ~ 1.5
	ap.decisionInterval = base + rand.Float64()*jitter
	ap.decisionTimer = ap.decisionInterval
}

// ── LLM 弹幕 ──

// tickLLM 驱动 LLM 连接器，收到响应时显示为气泡。
// 气泡不覆盖当前正在显示的模板文案（避免抢占重要操作反馈）。
func (ap *AIPlayer) tickLLM(dt float64, snap AISnapshot) {
	if ap.llmConn == nil {
		return
	}
	situation := ap.buildSituation(snap)
	text := ap.llmConn.Tick(dt, llm.BuildPrompt(situation))
	if text != "" && !ap.bubble.Visible() {
		ap.bubble.Show(text, BubbleAction, 3.0)
	}
}

// buildSituation 从快照构建 LLM 局势摘要。
func (ap *AIPlayer) buildSituation(snap AISnapshot) llm.Situation {
	// 威胁评估：基于敌人 HP 占比和 Boss 存在
	threatLevel := "low"
	if len(snap.Enemies) > 0 {
		activeCount := 0
		hasBoss := false
		for _, e := range snap.Enemies {
			if e.Active {
				activeCount++
				if e.Boss {
					hasBoss = true
				}
			}
		}
		if hasBoss || activeCount > 15 {
			threatLevel = "high"
		} else if activeCount > 5 {
			threatLevel = "medium"
		}
	}

	// 人类塔数 = 快照中总塔数 - AI 自己的塔数
	// （快照在 filterAITowers 之前的原始数据已被过滤，
	//   这里用 snap.Towers 是 AI 过滤后的，需要额外信息。
	//   简化处理：人类塔数暂用 0，不影响 prompt 质量。）
	return llm.Situation{
		Wave:            snap.Wave,
		MaxWaves:        snap.MaxWaves,
		AIGold:          ap.gold,
		HumanGold:       0, // AI 不可见人类金币
		Lives:           snap.Lives,
		MaxLives:        20, // 默认最大生命
		AITowerCount:    len(snap.Towers),
		HumanTowerCount: 0, // AI 不可见人类塔数
		LastAction:      ap.lastAction,
		ThreatLevel:     threatLevel,
	}
}
