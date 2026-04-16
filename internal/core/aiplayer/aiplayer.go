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
	StartWave() bool                                                    // 开始下一波
	SelectWarden(key string) bool                                       // 选择战灵
	ChooseAbility(row, col int, slotIndex int, abilityName string) bool // 选择能力
	UseItemForAI(itemKind int, towerRow, towerCol int) bool             // 使用道具
	UnlockAbilitySlotForAI(row, col int) (cost int, ok bool)           // 付费解锁能力槽
	SellRefundAmount(row, col int) int                                  // 查询卖塔退款金额
}

// ZoneProvider 区域查询接口。Zone 和 CoopZone 都实现此接口。
type ZoneProvider interface {
	OwnerOf(row, col int) int
}

// Config AI 玩家配置。
type Config struct {
	ZoneProvider       ZoneProvider // 区域查询（Zone 或 CoopZone）
	OwnerID            int          // 此 AI 的 owner ID（1..N-1）
	StartGold          int
	Ops                StageOps
	CellSize           int     // 网格像素尺寸
	SpawnX             float64 // 精灵初始 X（0=自动计算）
	SpawnY             float64 // 精灵初始 Y（0=自动计算）
	LLMEnabled         bool    // 是否启用 LLM 弹幕（或 ANTHROPIC_API_KEY 存在时自动启用）
	LLMOverrideEnabled bool    // LLM 战略决策是否可覆盖本地启发式（默认 true when LLM enabled）
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
	llmConn            *llm.Connector // LLM 弹幕连接器（nil 表示禁用）
	llmOverrideEnabled bool            // LLM 战略决策是否可覆盖本地启发式
	lastAction         string          // 上一次执行的行动标签（供 LLM prompt 使用）

	// ── Phase 2-3: 拟人行为 + 观战评论 ──
	behavior  *BehaviorState  // 拟人行为状态（漏怪/连杀/焦虑/巡视/发呆）
	spectator *SpectatorState // 观战评论状态（评论人类玩家行为）

	// ── AI 道具背包 ──
	// AI 区域掉落的道具进入此背包，使用时从此背包扣除。
	inventory map[int]int // itemKind → count

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
		behavior:     NewBehaviorState(20), // 默认 20 生命
		spectator:    NewSpectatorState(20),
		inventory:    make(map[int]int),
	}
	ap.engine.SetPersonality(personality)
	ap.engine.SetOwnerID(ownerID)
	ap.rollNextInterval()

	// LLM 弹幕连接器：显式启用或环境变量 ANTHROPIC_API_KEY 存在时创建
	llmCfg := llm.DefaultConfig()
	if cfg.LLMEnabled {
		llmCfg.Enabled = true
	}
	if llmCfg.Enabled {
		ap.llmConn = llm.NewConnector(llmCfg)
		// LLM 战略覆盖：默认启用（除非显式配置为 false 且配置了 LLMOverrideEnabled 字段）
		ap.llmOverrideEnabled = true
		if cfg.LLMEnabled && !cfg.LLMOverrideEnabled {
			// 显式 LLMEnabled=true 但 LLMOverrideEnabled 默认零值 false
			// → 需要区分"未设置"和"设置为 false"。Go 零值无法区分，
			// 因此当 LLMEnabled=true 时默认启用 override。
			ap.llmOverrideEnabled = true
		}
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

// AddItem 向 AI 背包添加一个道具（掉落拾取时调用）。
func (ap *AIPlayer) AddItem(kind int) {
	ap.inventory[kind]++
}

// ItemCount 返回指定道具数量。
func (ap *AIPlayer) ItemCount(kind int) int {
	return ap.inventory[kind]
}

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

// LatestAdvice 返回最近一次局势感知建议（只读）。
func (ap *AIPlayer) LatestAdvice() StrategicAdvice { return ap.engine.LatestAdvice }

// LatestCoop 返回最近一次协作分析结果（只读）。
func (ap *AIPlayer) LatestCoop() CoopAnalysis { return ap.engine.LatestCoop }

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
//  6. 拟人行为 + 观战评论（每帧）
//  7. Ping 响应（优先于正常决策）
//  8. 决策评估（按间隔）
//  9. LLM 弹幕
func (ap *AIPlayer) Tick(dt float64, snap AISnapshot) {
	ap.actions.Tick(dt)
	ap.sprite.Tick(dt)
	ap.bubble.Tick(dt)
	ap.ping.Tick(dt)
	ap.engine.TickWaveDelay(dt)

	// ── 拟人行为（每帧，不受 actionsBusy 完全阻断）──
	ap.behavior.Tick(dt, snap, ap.gold,
		ap.sprite, ap.bubble, ap.dialogue, ap.actions.Busy())

	// ── 观战评论（每帧，内部有 3s 检查间隔 + 8s 评论冷却）──
	ap.spectator.Tick(dt, snap, ap.bubble, ap.dialogue)

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

	// 注入 AI 道具背包到快照
	snap.Items = ap.buildItemSnapshot()

	// 保存全量塔数据（协作分析需要看到人类+AI 的全部塔）
	snap.AllTowers = snap.Towers

	// 过滤出 AI 区域的可建造格子和塔
	snap.BuildCells = ap.filterAICells(snap.BuildCells)
	snap.Towers = ap.filterAITowers(snap.Towers)

	decision := ap.engine.Evaluate(snap)

	// LLM 战略覆盖：如果 LLM 有新的战略决策，可覆盖本地启发式
	decision = ap.applyLLMStrategicOverride(dt, snap, decision)

	ap.executeDecision(decision)

	// 通知行为系统做了决策（重置 idle 时间）
	if decision.Type != DecisionIdle {
		ap.behavior.NotifyDecision()
	}

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

// itemCategoryNames 将 item.Kind 映射到类别名称。
// 保持与 item 包 kindFromString 一致的命名。
var itemCategoryNames = [...]string{
	0: "baseDamage",
	1: "potentialDamage",
	2: "baseSpeed",
	3: "potentialSpeed",
	4: "baseRange",
	5: "potentialRange",
}

// buildItemSnapshot 将 AI 背包转为快照格式。
func (ap *AIPlayer) buildItemSnapshot() []AIItem {
	var items []AIItem
	for kind, count := range ap.inventory {
		if count <= 0 {
			continue
		}
		cat := ""
		if kind >= 0 && kind < len(itemCategoryNames) {
			cat = itemCategoryNames[kind]
		}
		items = append(items, AIItem{
			Kind:     kind,
			Category: cat,
			Count:    count,
		})
	}
	return items
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
	case DecisionChooseAbility:
		ap.lastAction = "chose ability"
	case DecisionSell:
		ap.lastAction = "sold tower"
	case DecisionUseItem:
		ap.lastAction = "used item"
	case DecisionUnlockSlot:
		ap.lastAction = "unlocked slot"
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
		if d.Reason != "" {
			ap.bubble.Show(d.Reason, BubbleThink, 2.0)
		} else {
			ap.bubble.Show(ap.dialogue.Random("build_thinking"), BubbleThink, 1.5)
		}

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
		if d.Reason != "" {
			ap.bubble.Show(d.Reason, BubbleThink, 1.5)
		} else {
			ap.bubble.Show(ap.dialogue.Random("upgrade_thinking"), BubbleThink, 1.0)
		}

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
		// 根据 Reason 选择对应的文案类别
		dialogueKey := "wave_start"
		if d.Reason == "wave_early" {
			dialogueKey = "wave_early"
		} else if d.Reason == "boss_prepare" {
			dialogueKey = "boss_prepare"
		}
		ap.bubble.Show(ap.dialogue.Random(dialogueKey), BubbleAction, 1.2)
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

	case DecisionChooseAbility:
		row, col := d.Row, d.Col
		slotIdx, abilityName := d.SlotIndex, d.AbilityName
		ap.bubble.Show(ap.dialogue.Random("ability_choose"), BubbleThink, 1.5)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.8 + rand.Float64()*0.5,
			Label: "ability",
			Execute: func() {
				if ap.ops != nil {
					ap.ops.ChooseAbility(row, col, slotIdx, abilityName)
				}
			},
		})

	case DecisionSell:
		row, col := d.Row, d.Col
		ap.bubble.Show(ap.dialogue.Random("sell_thinking"), BubbleThink, 1.0)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.8 + rand.Float64()*0.5,
			Label: "sell",
			Execute: func() {
				if ap.ops == nil {
					return
				}
				// SellTowerForAI 内部已按 Owner 发放退款到对应 AI 玩家
				if ap.ops.SellTowerForAI(row, col) {
					ap.bubble.Show(ap.dialogue.Random("sell_done"), BubbleAction, 1.0)
				}
			},
		})

	case DecisionUseItem:
		row, col := d.Row, d.Col
		itemKind := d.ItemKind
		ap.bubble.Show(ap.dialogue.Random("item_use"), BubbleAction, 1.5)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.5 + rand.Float64()*0.3,
			Label: "use_item",
			Execute: func() {
				if ap.ops == nil {
					return
				}
				if ap.inventory[itemKind] <= 0 {
					return
				}
				if ap.ops.UseItemForAI(itemKind, row, col) {
					ap.inventory[itemKind]--
				}
			},
		})

	case DecisionUnlockSlot:
		row, col := d.Row, d.Col
		ap.bubble.Show(ap.dialogue.Random("unlock_slot"), BubbleThink, 1.5)
		ap.actions.Enqueue(DelayedAction{
			Delay: 0.6 + rand.Float64()*0.4,
			Label: "unlock_slot",
			Execute: func() {
				if ap.ops == nil {
					return
				}
				cost, ok := ap.ops.UnlockAbilitySlotForAI(row, col)
				if ok {
					ap.gold -= cost
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
	memory := ap.llmConn.GetMemory()
	text := ap.llmConn.Tick(dt, llm.BuildPrompt(situation, memory))
	if text != "" && !ap.bubble.Visible() {
		ap.bubble.Show(text, BubbleAction, 3.0)
		// 记录到短期记忆
		ap.llmConn.RecordMemory(ap.lastAction, text)
	}
}

// applyLLMStrategicOverride 检查 LLM 战略决策，可能覆盖本地启发式决策。
//
// 支持多步操作链：LLM 返回最多 3 个操作，第一个立即执行，
// 后续操作以交错延迟入队到 ActionQueue。
//
// 覆盖规则（基于第一个操作）：
//   - LLM 未启用或未返回决策 → 保持原决策
//   - wait → 覆盖为 Idle
//   - build + priority → 覆盖 build 决策的偏好方向
//   - upgrade → 覆盖为 Upgrade
//   - sell → 强制卖塔（选最弱的）
//   - ability → 设置能力偏好
//   - item → 使用道具
//   - LLM 的 say 用作气泡文案（优先于 reason）
func (ap *AIPlayer) applyLLMStrategicOverride(dt float64, snap AISnapshot, local Decision) Decision {
	if ap.llmConn == nil || !ap.llmOverrideEnabled {
		return local
	}

	// 构建战略 prompt
	coopDesc := CoopDescription(ap.engine.LatestCoop)
	situation := ap.buildSituation(snap)
	prompt := llm.BuildStrategicPrompt(situation, ap.engine.LatestAdvice.Priority, coopDesc)

	// 驱动战略决策通道（异步，非阻塞）
	decision := ap.llmConn.TickStrategic(dt, prompt)
	if decision == nil {
		return local
	}

	// 弹幕文案（如果 LLM 提供了）
	if decision.Say != "" {
		ap.bubble.Show(decision.Say, BubbleAction, 2.5)
	}

	// 处理多步操作链：第一个操作覆盖本地决策，
	// 后续操作入队到 ActionQueue（交错延迟）
	if len(decision.Actions) > 1 {
		ap.enqueueFollowUpActions(decision.Actions[1:], snap)
	}

	// 用第一个操作覆盖本地决策
	first := decision.FirstAction()
	reason := first.Reason
	if reason == "" {
		reason = decision.Say
	}

	return ap.applyLLMAction(first, reason, local, snap)
}

// applyLLMAction 将单个 LLM 操作转换为游戏决策。
func (ap *AIPlayer) applyLLMAction(action llm.LLMAction, reason string, local Decision, snap AISnapshot) Decision {
	switch action.Type {
	case "wait":
		return Decision{Type: DecisionIdle, Reason: reason}

	case "build":
		// 覆盖建造偏好方向
		if action.Priority != "" {
			priorityMap := map[string]string{
				"cc": "build_cc", "dps": "build_dps", "aoe": "build_aoe",
			}
			if p, ok := priorityMap[action.Priority]; ok {
				ap.engine.LatestAdvice.Priority = p
			}
		}
		if local.Type == DecisionBuild {
			local.Reason = reason
			return local
		}
		if len(snap.BuildCells) > 0 && len(snap.TowerDefs) > 0 {
			forced := ap.engine.ForceBuild(snap)
			forced.Reason = reason
			return forced
		}
		return local

	case "upgrade":
		if local.Type == DecisionUpgrade {
			local.Reason = reason
			return local
		}
		if len(snap.Towers) > 0 {
			forced := ap.engine.ForceUpgrade(snap)
			forced.Reason = reason
			return forced
		}
		return local

	case "sell":
		// 强制卖塔：选评分最低的（复用引擎逻辑）
		if len(snap.Towers) >= 3 {
			if d, ok := ap.engine.pickSell(snap); ok {
				d.Reason = reason
				return d
			}
		}
		return local

	case "ability":
		// 设置能力偏好方向（影响下次 pickAbility 评分）
		// ability 操作本身不直接执行，而是影响评分偏好
		return local

	case "item":
		// 强制使用道具
		if d, ok := ap.engine.pickItemUse(snap); ok {
			d.Reason = reason
			return d
		}
		return local
	}

	return local
}

// enqueueFollowUpActions 将后续 LLM 操作以交错延迟入队到 ActionQueue。
// 每个后续操作延迟 2-3 秒（给前一个操作执行和视觉反馈的时间）。
func (ap *AIPlayer) enqueueFollowUpActions(actions []llm.LLMAction, snap AISnapshot) {
	for i, action := range actions {
		// 捕获循环变量
		act := action
		delay := float64(i+1)*2.0 + rand.Float64()

		ap.actions.Enqueue(DelayedAction{
			Delay: delay,
			Label: "llm_followup_" + act.Type,
			Execute: func() {
				ap.executeLLMFollowUp(act, snap)
			},
		})
	}
}

// executeLLMFollowUp 执行 LLM 后续操作。
// 每个操作独立检查前置条件（金币、库存等），条件不满足时静默跳过。
func (ap *AIPlayer) executeLLMFollowUp(action llm.LLMAction, snap AISnapshot) {
	switch action.Type {
	case "build":
		// 检查金币是否足够
		cost := 50
		if ap.ops != nil && len(snap.TowerDefs) > 0 {
			cost = ap.ops.TowerCost(snap.TowerDefs[0].Key)
		}
		if ap.gold < cost || len(snap.BuildCells) == 0 {
			return
		}
		// 使用引擎选最佳位置
		d := ap.engine.ForceBuild(snap)
		if ap.ops != nil && ap.ops.BuildTowerForAI(d.TowerKey, d.Row, d.Col, ap.ownerID) {
			ap.gold -= cost
			if action.Reason != "" {
				ap.bubble.Show(action.Reason, BubbleAction, 1.5)
			}
		}

	case "upgrade":
		buyCost := 10
		if ap.ops != nil {
			buyCost = ap.ops.StrengthBuyCost()
		}
		if ap.gold < buyCost || len(snap.Towers) == 0 {
			return
		}
		d := ap.engine.ForceUpgrade(snap)
		if ap.ops != nil && ap.ops.UpgradeTowerForAI(d.Row, d.Col) {
			ap.gold -= buyCost
		}

	case "sell":
		if len(snap.Towers) < 3 {
			return
		}
		if d, ok := ap.engine.pickSell(snap); ok {
			if ap.ops != nil {
				ap.ops.SellTowerForAI(d.Row, d.Col)
			}
		}

	case "item":
		if d, ok := ap.engine.pickItemUse(snap); ok {
			if ap.inventory[d.ItemKind] > 0 && ap.ops != nil {
				if ap.ops.UseItemForAI(d.ItemKind, d.Row, d.Col) {
					ap.inventory[d.ItemKind]--
				}
			}
		}

	case "wait":
		// 什么都不做
	}
}

// TriggerLLMEvent 触发 LLM 即时调用（用于重要事件）。
// 事件如：Boss 来袭、连续漏怪、玩家 ping 等。
func (ap *AIPlayer) TriggerLLMEvent(event string, snap AISnapshot) {
	if ap.llmConn == nil {
		return
	}
	situation := ap.buildSituation(snap)
	situation.Mood = event // 用事件作为情绪上下文
	memory := ap.llmConn.GetMemory()
	ap.llmConn.TriggerImmediate(llm.BuildPrompt(situation, memory))
}

// buildSituation 从快照构建 LLM 局势摘要（含详细塔/敌人/道具数据）。
func (ap *AIPlayer) buildSituation(snap AISnapshot) llm.Situation {
	// 威胁评估：基于敌人 HP 占比和 Boss 存在
	threatLevel := "low"
	hasBoss := false
	activeCount := 0
	if len(snap.Enemies) > 0 {
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

	// 情绪推断：基于最近事件
	mood := "calm"
	if snap.Lives < ap.behavior.prevLives {
		mood = "worried"
	} else if ap.behavior.killStreak >= killStreakSmall {
		mood = "excited"
	} else if ap.gold < economyWorryGold {
		mood = "anxious"
	} else if hasBoss {
		mood = "tense"
	}

	// 个性描述
	personalityDesc := ap.buildPersonalityDesc()

	// 协作分析描述
	coopDesc := CoopDescription(ap.engine.LatestCoop)

	// ── 构建详细塔描述 ──
	aiTowers := ap.buildTowerDescs(snap)

	// ── 构建敌人分组 ──
	enemyGroups := buildEnemyGroups(snap.Enemies)

	// ── 构建可用道具列表 ──
	var availItems []string
	for _, it := range snap.Items {
		if it.Count > 0 {
			availItems = append(availItems, it.Category)
		}
	}

	// ── 统计待选能力的塔数量 ──
	pendingAbilities := 0
	for _, t := range snap.Towers {
		if len(t.PendingSlots) > 0 {
			pendingAbilities++
		}
	}

	// ── 判断可用操作 ──
	canBuild := false
	for _, d := range snap.TowerDefs {
		if d.Cost <= ap.gold {
			canBuild = true
			break
		}
	}
	canUpgrade := len(snap.Towers) > 0 && ap.gold >= ap.engine.strengthBuyCost

	return llm.Situation{
		Wave:            snap.Wave,
		MaxWaves:        snap.MaxWaves,
		AIGold:          ap.gold,
		HumanGold:       snap.HumanGold,
		Lives:           snap.Lives,
		MaxLives:        20,
		AITowerCount:    len(snap.Towers),
		HumanTowerCount: snap.HumanTowerCount,
		LastAction:      ap.lastAction,
		ThreatLevel:     threatLevel,
		PersonalityDesc: personalityDesc,
		Mood:            mood,
		CoopDesc:        coopDesc,
		AdvicePriority:  ap.engine.LatestAdvice.Priority,

		// ── 深度知识字段 ──
		AITowers:         aiTowers,
		EnemyGroups:      enemyGroups,
		AvailableItems:   availItems,
		PendingAbilities: pendingAbilities,
		NextWaveBoss:     snap.NextWaveIsBoss,
		CanBuild:         canBuild,
		CanUpgrade:       canUpgrade,
	}
}

// buildTowerDescs 将 AI 塔快照转为 LLM 可读的描述列表。
// 使用路径点数据估算每座塔在路径上的位置（entrance/middle/exit）。
func (ap *AIPlayer) buildTowerDescs(snap AISnapshot) []llm.TowerDesc {
	descs := make([]llm.TowerDesc, 0, len(snap.Towers))
	pathLen := len(snap.PathPoints)

	for _, t := range snap.Towers {
		// 估算位置区段
		pos := estimateTowerPosition(t, pathLen)

		// 提取攻击方式（从能力列表中找 attack 类）
		style := ""
		for _, ab := range t.Abilities {
			if attackStyles[ab] {
				style = ab
				break
			}
		}

		descs = append(descs, llm.TowerDesc{
			Position:  pos,
			Abilities: t.Abilities,
			Damage:    t.Damage,
			Range:     t.Range,
			Kills:     t.Kills,
			Style:     style,
			Strength:  t.Strength,
		})
	}
	return descs
}

// attackStyles 攻击方式能力集合，用于识别塔的主要攻击方式。
var attackStyles = map[string]bool{
	"projectile": true, "scatter": true, "wideBeam": true,
	"spinAoe": true, "radial": true, "barrage": true,
}

// estimateTowerPosition 根据塔的列位置估算其在路径上的区段。
// 简单三分法：前 1/3 路径 = entrance，中间 = middle，后 1/3 = exit。
func estimateTowerPosition(t AITower, pathLen int) string {
	if pathLen == 0 {
		return "middle"
	}
	// 用列位置粗略估计路径进度（假设地图宽度约 20 列）
	maxCol := 20.0
	progress := float64(t.Col) / maxCol
	switch {
	case progress < 0.33:
		return "entrance"
	case progress > 0.66:
		return "exit"
	default:
		return "middle"
	}
}

// buildEnemyGroups 将敌人快照按类型聚合为分组（tank/runner/boss/swarm/normal）。
func buildEnemyGroups(enemies []AIEnemy) []llm.EnemyGroup {
	if len(enemies) == 0 {
		return nil
	}

	// 先统计活跃敌人和均值
	var active []AIEnemy
	var totalHP, totalSpeed float64
	for _, e := range enemies {
		if e.Active {
			active = append(active, e)
			totalHP += e.HP
			totalSpeed += e.Speed
		}
	}
	if len(active) == 0 {
		return nil
	}
	avgHP := totalHP / float64(len(active))
	avgSpeed := totalSpeed / float64(len(active))

	// 按类型分组
	groups := map[string]*llm.EnemyGroup{}
	for _, e := range active {
		typ := "normal"
		switch {
		case e.Boss:
			typ = "boss"
		case e.HP > avgHP*1.5:
			typ = "tank"
		case e.Speed > avgSpeed*1.2:
			typ = "runner"
		}
		if g, ok := groups[typ]; ok {
			g.Count++
			// 滚动平均
			g.AvgHP = (g.AvgHP*float64(g.Count-1) + e.HP) / float64(g.Count)
		} else {
			groups[typ] = &llm.EnemyGroup{Type: typ, Count: 1, AvgHP: e.HP}
		}
	}

	// 总数 > 10 且 normal 占多数 → 标记为 swarm
	if g, ok := groups["normal"]; ok && g.Count > 10 && float64(g.Count) > float64(len(active))*0.6 {
		g.Type = "swarm"
	}

	result := make([]llm.EnemyGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}
	return result
}

// buildPersonalityDesc 根据个性参数生成自然语言描述。
func (ap *AIPlayer) buildPersonalityDesc() string {
	p := ap.personality
	desc := "你的性格: "
	if p.Aggression > 0.65 {
		desc += "激进，喜欢DPS塔，追求火力输出。"
	} else if p.Aggression < 0.35 {
		desc += "稳健，喜欢控制塔，注重防守。"
	} else {
		desc += "均衡，攻防兼顾。"
	}
	if p.Economy > 0.65 {
		desc += "精打细算，喜欢攒钱。"
	} else if p.Economy < 0.35 {
		desc += "大手大脚，有钱就花。"
	}
	if p.Risk > 0.65 {
		desc += "胆子大，敢冒险。"
	} else if p.Risk < 0.35 {
		desc += "谨慎保守。"
	}
	return desc
}
