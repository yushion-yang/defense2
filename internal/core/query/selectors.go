// selectors.go — 状态选择器。
// 纯查询函数，从游戏状态中提取信息用于 UI 和 AI 提示。
// 不修改任何游戏状态。
package query

// TowerInfo 塔的查询快照（避免直接依赖 tower 包）。
type TowerInfo struct {
	SlotIndex    int     // 格子索引
	Damage       float64 // 当前伤害
	AttackSpeed  float64 // 当前攻速（次/秒）
	Range        float64 // 当前射程
	SplashRadius float64 // 溅射半径（0=无溅射）
}

// WaveComposition 波次组成统计。
type WaveComposition struct {
	Normal      int // 普通敌人数
	Runner      int // 快速敌人数
	Swarm       int // 蜂群敌人数
	Tank        int // 坦克敌人数
	Medic       int // 治疗敌人数
	Regenerator int // 回血敌人数
	Berserker   int // 狂暴敌人数
	Elite       int // 精英敌人数
	Boss        int // Boss数
}

// ReadinessLevel 防御准备度等级。
type ReadinessLevel string

const (
	ReadinessStrong ReadinessLevel = "strong" // 优势
	ReadinessReady  ReadinessLevel = "ready"  // 可一战
	ReadinessWeak   ReadinessLevel = "weak"   // 火力偏弱
)

// ReadinessResult 防御准备度评估结果。
type ReadinessResult struct {
	Level  ReadinessLevel // 准备度等级
	Label  string         // 显示标签
	Advice string         // 策略建议
	Score  float64        // 防御得分（调试用）
}

// GetTowerAtSlot 根据格子索引查找塔，未找到返回 nil。
func GetTowerAtSlot(towers []TowerInfo, slotIndex int) *TowerInfo {
	for i := range towers {
		if towers[i].SlotIndex == slotIndex {
			return &towers[i]
		}
	}
	return nil
}

// 波次压力权重表（敌人类型 → 威胁权重）
var pressureWeights = map[string]float64{
	"normal":      1.0,
	"runner":      0.8,
	"swarm":       0.9,
	"tank":        1.6,
	"medic":       1.35,
	"regenerator": 1.4,
	"berserker":   1.2,
	"elite":       3.1,
	"boss":        8.0,
}

// GetWavePressureScore 计算波次压力分数。
// wave: 波次编号; comp: 波次组成。
func GetWavePressureScore(wave int, comp WaveComposition) float64 {
	score := float64(wave) * 0.8
	score += float64(comp.Normal) * pressureWeights["normal"]
	score += float64(comp.Runner) * pressureWeights["runner"]
	score += float64(comp.Swarm) * pressureWeights["swarm"]
	score += float64(comp.Tank) * pressureWeights["tank"]
	score += float64(comp.Medic) * pressureWeights["medic"]
	score += float64(comp.Regenerator) * pressureWeights["regenerator"]
	score += float64(comp.Berserker) * pressureWeights["berserker"]
	score += float64(comp.Elite) * pressureWeights["elite"]
	score += float64(comp.Boss) * pressureWeights["boss"]
	return score
}

// GetDefenseReadiness 评估当前防御准备度。
// towers: 所有已建塔; wavePressure: 下一波压力分数。
func GetDefenseReadiness(towers []TowerInfo, wavePressure float64) ReadinessResult {
	// 计算防御分数
	totalDPS := 0.0
	splashCount := 0
	for _, t := range towers {
		if t.AttackSpeed > 0 {
			totalDPS += t.Damage * t.AttackSpeed
		}
		if t.SplashRadius > 0 {
			splashCount++
		}
	}
	defenseScore := totalDPS/22.0 + float64(len(towers))*1.6 + float64(splashCount)*2.4
	delta := defenseScore - wavePressure

	var result ReadinessResult
	result.Score = defenseScore

	switch {
	case delta >= 4.5:
		result.Level = ReadinessStrong
		result.Label = "优势"
		result.Advice = "火力充沛，可以存钱投资"
	case delta >= 0:
		result.Level = ReadinessReady
		result.Label = "可一战"
		result.Advice = "刚好够用，注意补强薄弱点"
	default:
		result.Level = ReadinessWeak
		result.Label = "火力偏弱"
		result.Advice = "建议补充防御塔或升级现有塔"
	}

	return result
}
