// decision.go — AI 决策引擎。
//
// Phase 1 简单版：基于经济状态和进度阶段做造塔/升级决策。
// 选位逻辑：优先靠近地图中心的格子。
// Phase 2 将加入威胁评估、能力选择、个性系统。
package aiplayer

import (
	"math"
	"math/rand"
)

// DecisionType 决策类型。
type DecisionType int

const (
	DecisionIdle    DecisionType = iota // 什么都不做
	DecisionBuild                       // 造塔
	DecisionUpgrade                     // 升级
)

// Decision 决策结果。
type Decision struct {
	Type     DecisionType
	Row, Col int     // 目标格子
	TowerKey string  // 塔类型（Build 时使用）
	CenterX  float64 // 目标像素中心
	CenterY  float64
}

// AISnapshot AI 可见的游戏状态快照（纯值，零 core 依赖）。
type AISnapshot struct {
	Gold       int
	Wave       int
	MaxWaves   int
	WaveActive bool
	Lives      int
	Towers     []AITower
	BuildCells []AICell
	TowerDefs  []AITowerDef
	MapCenterX float64
	MapCenterY float64
}

// AITower 塔快照。
type AITower struct {
	Row, Col    int
	Damage      float64
	Strength    int
	AttackSpeed float64
	Range       float64
	Kills       int
}

// AICell 可建造格子。
type AICell struct {
	Row, Col int
	X, Y     float64
}

// AITowerDef 可建造塔定义。
type AITowerDef struct {
	Key    string
	Cost   int
	Damage float64
	Range  float64
	Index  int
}

// DecisionEngine 决策引擎。
type DecisionEngine struct {
	strengthBuyCost int
}

// NewDecisionEngine 创建决策引擎。
func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{
		strengthBuyCost: 10,
	}
}

// SetStrengthBuyCost 设置升级花费。
func (e *DecisionEngine) SetStrengthBuyCost(cost int) {
	e.strengthBuyCost = cost
}

// Evaluate 评估当前状态，返回最佳决策。
//
// 策略概要:
//   1. 早期(进度<40%): 优先造塔
//   2. 中后期(>=40%): 升级为主，30% 概率补塔
//   3. 金币不足: idle
func (e *DecisionEngine) Evaluate(snap AISnapshot) Decision {
	progress := 0.0
	if snap.MaxWaves > 0 {
		progress = float64(snap.Wave) / float64(snap.MaxWaves)
	}

	canBuild := false
	for _, d := range snap.TowerDefs {
		if d.Cost <= snap.Gold {
			canBuild = true
			break
		}
	}
	canUpgrade := len(snap.Towers) > 0 && snap.Gold >= e.strengthBuyCost

	switch {
	case !canBuild && !canUpgrade:
		return Decision{Type: DecisionIdle}
	case progress < 0.4 && canBuild && len(snap.BuildCells) > 0:
		return e.pickBuild(snap)
	case progress >= 0.4 && canUpgrade:
		// 中后期升级为主，偶尔补塔
		if canBuild && len(snap.BuildCells) > 0 && rand.Float64() < 0.3 {
			return e.pickBuild(snap)
		}
		return e.pickUpgrade(snap)
	case canBuild && len(snap.BuildCells) > 0:
		return e.pickBuild(snap)
	case canUpgrade:
		return e.pickUpgrade(snap)
	default:
		return Decision{Type: DecisionIdle}
	}
}

// pickBuild 选择造塔位置和类型。
func (e *DecisionEngine) pickBuild(snap AISnapshot) Decision {
	// 选性价比最高的可负担塔
	var bestDef AITowerDef
	bestScore := -1.0
	for _, d := range snap.TowerDefs {
		if d.Cost > snap.Gold {
			continue
		}
		score := (d.Damage * d.Range) / float64(d.Cost)
		if score > bestScore {
			bestScore = score
			bestDef = d
		}
	}

	// 选位置：靠近地图中心
	cx, cy := snap.MapCenterX, snap.MapCenterY
	if cx == 0 && cy == 0 {
		cx, cy = 600, 270
	}

	bestCell := snap.BuildCells[0]
	bestDist := math.MaxFloat64
	for _, c := range snap.BuildCells {
		dist := math.Hypot(c.X-cx, c.Y-cy)
		if dist < bestDist {
			bestDist = dist
			bestCell = c
		}
	}

	return Decision{
		Type:     DecisionBuild,
		Row:      bestCell.Row,
		Col:      bestCell.Col,
		TowerKey: bestDef.Key,
		CenterX:  bestCell.X,
		CenterY:  bestCell.Y,
	}
}

// pickUpgrade 选伤害最高的塔升级。
func (e *DecisionEngine) pickUpgrade(snap AISnapshot) Decision {
	best := snap.Towers[0]
	for _, t := range snap.Towers[1:] {
		if t.Damage > best.Damage {
			best = t
		}
	}
	return Decision{
		Type: DecisionUpgrade,
		Row:  best.Row,
		Col:  best.Col,
	}
}
