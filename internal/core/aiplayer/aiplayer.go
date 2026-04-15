// aiplayer.go — AI 玩家主结构体。
//
// 职责：编排决策引擎、行动队列、精灵、气泡，驱动 AI 的完整行为循环。
// 关联：由 stage.go 每帧调用 Tick()，由 hud/ai_overlay.go 渲染。
// 设计：零 core 依赖（除自身子模块），通过 StageOps 接口与游戏交互。
package aiplayer

import (
	"math/rand"
)

// StageOps AI 可用的游戏操作接口。
// 由 StageScene 实现，避免 AI 直接依赖 scene 包。
type StageOps interface {
	BuildTowerForAI(key string, row, col int) bool
	UpgradeTowerForAI(row, col int) bool
	SellTowerForAI(row, col int) bool
	TowerCost(key string) int
	StrengthBuyCost() int
}

// Config AI 玩家配置。
type Config struct {
	Zone      *Zone
	StartGold int
	Ops       StageOps
	CellSize  int // 网格像素尺寸，用于精灵初始位置计算
}

// AIPlayer AI 玩家。
type AIPlayer struct {
	gold     int
	zone     *Zone
	ops      StageOps
	sprite   *Sprite
	bubble   *BubbleManager
	actions  *ActionQueue
	engine   *DecisionEngine
	dialogue *DialogueBank

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

	// 精灵初始位置：AI 区域中心
	spawnX := float64(cfg.Zone.SplitCol+cfg.Zone.cols/4) * float64(cellSize)
	spawnY := float64(cfg.Zone.rows/2) * float64(cellSize)

	ap := &AIPlayer{
		gold:     cfg.StartGold,
		zone:     cfg.Zone,
		ops:      cfg.Ops,
		sprite:   NewSprite(spawnX, spawnY),
		bubble:   NewBubbleManager(),
		actions:  NewActionQueue(),
		engine:   NewDecisionEngine(),
		dialogue: DefaultDialogueBank(),
	}
	ap.rollNextInterval()

	if cfg.Ops != nil {
		ap.engine.SetStrengthBuyCost(cfg.Ops.StrengthBuyCost())
	}

	return ap
}

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

// ── 主循环 ──

// Tick 每帧更新。
//  1. 行动队列处理
//  2. 精灵更新
//  3. 气泡更新
//  4. 决策评估（按间隔）
func (ap *AIPlayer) Tick(dt float64, snap AISnapshot) {
	ap.actions.Tick(dt)
	ap.sprite.Tick(dt)
	ap.bubble.Tick(dt)

	// 有待执行行动时不做新决策
	if ap.actions.Busy() {
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
}

// ── 内部方法 ──

func (ap *AIPlayer) filterAICells(cells []AICell) []AICell {
	var result []AICell
	for _, c := range cells {
		if ap.zone.OwnerOf(c.Row, c.Col) == ZoneAI {
			result = append(result, c)
		}
	}
	return result
}

func (ap *AIPlayer) filterAITowers(towers []AITower) []AITower {
	var result []AITower
	for _, t := range towers {
		if ap.zone.OwnerOf(t.Row, t.Col) == ZoneAI {
			result = append(result, t)
		}
	}
	return result
}

func (ap *AIPlayer) executeDecision(d Decision) {
	switch d.Type {
	case DecisionIdle:
		if rand.Float64() < 0.1 {
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
				if ap.ops != nil && ap.ops.BuildTowerForAI(towerKey, row, col) {
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
	}
}

func (ap *AIPlayer) rollNextInterval() {
	ap.decisionInterval = 2.0 + rand.Float64()*2.0
	ap.decisionTimer = ap.decisionInterval
}
