// awareness.go — 局势感知系统。
//
// 分析敌人组成、路径压力、防御缺口，为决策引擎提供高层策略建议。
// 职责：将低层快照数据提炼为高层威胁评估和策略建议。
// 关联：由 decision.go Evaluate() 调用，输出 StrategicAdvice 影响造塔/升级决策。
package aiplayer

import "math"

// ── 威胁分析 ──────────────────────────────────────────

// ThreatAnalysis 当前场上敌人的威胁评估结果。
type ThreatAnalysis struct {
	TotalEnemies   int
	BossCount      int
	TankCount      int     // HP > average * 1.5 的高血量敌人
	FastCount      int     // Speed > average * 1.2 的快速敌人
	AvgHP          float64
	AvgSpeed       float64
	ThreatLevel    string // "low" / "medium" / "high" / "critical"
	DominantThreat string // "tank" / "fast" / "boss" / "swarm" / "none"
}

// AnalyzeThreats 分析场上活跃敌人的威胁构成。
//
// 流程：
//  1. 统计活跃且未死亡的敌人，计算均值
//  2. 按阈值分类：tank(HP>1.5x均值)、fast(速度>1.2x均值)、boss
//  3. 综合判定威胁等级和主要威胁类型
func AnalyzeThreats(enemies []AIEnemy) ThreatAnalysis {
	ta := ThreatAnalysis{DominantThreat: "none", ThreatLevel: "low"}

	// 只统计活跃敌人
	var active []AIEnemy
	for _, e := range enemies {
		if e.Active {
			active = append(active, e)
		}
	}

	ta.TotalEnemies = len(active)
	if ta.TotalEnemies == 0 {
		return ta
	}

	// 计算均值
	var totalHP, totalSpeed float64
	for _, e := range active {
		totalHP += e.HP
		totalSpeed += e.Speed
		if e.Boss {
			ta.BossCount++
		}
	}
	ta.AvgHP = totalHP / float64(ta.TotalEnemies)
	ta.AvgSpeed = totalSpeed / float64(ta.TotalEnemies)

	// 分类计数（需要均值先算好）
	for _, e := range active {
		if e.HP > ta.AvgHP*1.5 {
			ta.TankCount++
		}
		if e.Speed > ta.AvgSpeed*1.2 {
			ta.FastCount++
		}
	}

	// 威胁等级：基于敌人总数
	switch {
	case ta.TotalEnemies >= 16:
		ta.ThreatLevel = "critical"
	case ta.TotalEnemies >= 6:
		ta.ThreatLevel = "high"
	case ta.TotalEnemies >= 1:
		ta.ThreatLevel = "medium"
	}

	// 主要威胁类型：优先级 boss > tank/fast > swarm > none
	// Boss 存在时一定是主要威胁
	if ta.BossCount > 0 {
		ta.DominantThreat = "boss"
		return ta
	}

	// 在 tank 和 fast 中取更多的那个
	// swarm: 总数 > 10 且没有明显的 tank/fast 主导
	dominantCount := 0
	if ta.TankCount > ta.FastCount {
		ta.DominantThreat = "tank"
		dominantCount = ta.TankCount
	} else if ta.FastCount > ta.TankCount {
		ta.DominantThreat = "fast"
		dominantCount = ta.FastCount
	}

	// 主导类型不到总数 30% 且总数 > 10 → swarm
	if ta.TotalEnemies > 10 && float64(dominantCount) < float64(ta.TotalEnemies)*0.3 {
		ta.DominantThreat = "swarm"
	}

	return ta
}

// ── 防御缺口检测 ──────────────────────────────────────────

// DefenseGap 防线中的薄弱环节。
type DefenseGap struct {
	NeedsCC       bool  // 缺少减速/眩晕
	NeedsDPS      bool  // 缺少输出（总 DPS 不足以应对当前 HP 增长）
	NeedsAOE      bool  // 缺少范围伤害（面对 swarm 时）
	NeedsAntiTank bool  // 缺少反坦（面对高 HP 敌人时）
	CoverageGap   []int // 路径中没有被任何塔覆盖的区段索引
	WeakestZone   string // "entrance" / "middle" / "exit" — 防守最薄弱的区域
}

// CC 类能力关键字。命中其一即视为该塔有控制能力。
var ccAbilities = map[string]bool{
	"slow": true, "stun": true, "root": true, "freeze": true,
}

// AOE 类能力关键字。
var aoeAbilities = map[string]bool{
	"splash": true, "spinAoe": true, "scatter": true, "radial": true,
	"mortar": true, "nova": true,
}

// DetectDefenseGaps 检测当前防线的薄弱环节。
//
// 流程：
//  1. 扫描塔的能力列表，统计 CC/AOE/DPS 覆盖
//  2. 估算总 DPS 与敌人 HP 增长的对比
//  3. 将路径分为三段，检查每段的塔覆盖情况
func DetectDefenseGaps(towers []AITower, threats ThreatAnalysis, pathLen int) DefenseGap {
	gap := DefenseGap{WeakestZone: "middle"}

	hasCC := false
	hasAOE := false
	totalDPS := 0.0

	for _, t := range towers {
		// 检查能力覆盖
		for _, ab := range t.Abilities {
			if ccAbilities[ab] {
				hasCC = true
			}
			if aoeAbilities[ab] {
				hasAOE = true
			}
		}
		// 估算每座塔的 DPS（damage * attackSpeed）
		as := t.AttackSpeed
		if as <= 0 {
			as = 1.0 // fallback
		}
		totalDPS += t.Damage * as
	}

	// NeedsCC: 没有 CC 能力且场上有 tank 或 boss
	gap.NeedsCC = !hasCC && (threats.TankCount > 0 || threats.BossCount > 0)

	// NeedsDPS: 总 DPS 不足以在一波内杀光敌人
	// 粗略估计：DPS 需要 > avgHP * enemyCount * 0.1（10 秒内清场的能力）
	if threats.TotalEnemies > 0 {
		requiredDPS := threats.AvgHP * float64(threats.TotalEnemies) * 0.1
		gap.NeedsDPS = totalDPS < requiredDPS
	}

	// NeedsAOE: 面对大量敌人但没有 AOE
	gap.NeedsAOE = threats.TotalEnemies > 8 && !hasAOE

	// NeedsAntiTank: tank 多但 DPS 不足以应对
	if threats.TankCount > 2 && threats.AvgHP > 0 {
		tankDPSNeeded := float64(threats.TankCount) * threats.AvgHP * 0.05
		gap.NeedsAntiTank = totalDPS < tankDPSNeeded
	}

	// 路径覆盖分析：将路径分为三段（entrance / middle / exit）
	if pathLen > 0 {
		zoneSize := pathLen / 3
		if zoneSize == 0 {
			zoneSize = 1
		}
		zoneTowerCounts := [3]int{} // entrance / middle / exit

		for _, t := range towers {
			// 用塔的列位置粗略判断在哪个区段
			// 归一化到 [0, pathLen) 空间
			towerPathPos := estimateTowerPathPosition(t, pathLen)
			zone := towerPathPos / zoneSize
			if zone > 2 {
				zone = 2
			}
			zoneTowerCounts[zone]++
		}

		// 找最弱的区段
		zoneNames := [3]string{"entrance", "middle", "exit"}
		weakest := 0
		for i := 1; i < 3; i++ {
			if zoneTowerCounts[i] < zoneTowerCounts[weakest] {
				weakest = i
			}
		}
		gap.WeakestZone = zoneNames[weakest]

		// 收集无塔覆盖的区段索引
		for i := 0; i < 3; i++ {
			if zoneTowerCounts[i] == 0 {
				gap.CoverageGap = append(gap.CoverageGap, i)
			}
		}
	}

	return gap
}

// estimateTowerPathPosition 根据塔的行列位置估算其在路径上的位置（0-based 索引）。
// 使用简单的列映射：假设路径从左到右穿越地图。
func estimateTowerPathPosition(t AITower, pathLen int) int {
	// 用列位置作为路径进度的粗略估计
	// 假设地图宽度约 20 列，映射到 pathLen
	maxCol := 20.0
	col := float64(t.Col)
	pos := int(col / maxCol * float64(pathLen))
	if pos < 0 {
		pos = 0
	}
	if pos >= pathLen {
		pos = pathLen - 1
	}
	return pos
}

// ── 策略建议 ──────────────────────────────────────────

// StrategicAdvice 高层策略建议。
type StrategicAdvice struct {
	Priority      string  // "build_cc" / "build_dps" / "build_aoe" / "upgrade" / "balanced"
	Urgency       float64 // 0-1, 越高越紧急
	Reason        string  // 可读的策略原因（用于气泡文案）
	PreferredSlot string  // "entrance" / "middle" / "exit" — 建议建造区域
}

// GenerateAdvice 综合缺口和威胁生成策略建议。
//
// 优先级规则（从高到低）：
//  1. 危急 + 缺 DPS → build_dps (urgency=1.0)
//  2. 有 Boss + 缺反坦 → upgrade (urgency=0.9)
//  3. 大量敌人 + 缺 AOE → build_aoe (urgency=0.8)
//  4. 缺 CC → build_cc (urgency=0.7)
//  5. 缺 DPS（非危急）→ build_dps (urgency=0.6)
//  6. 无缺口 → balanced (urgency=0.3)
//
// 个性调制：激进型偏 DPS，防守型偏 CC。
func GenerateAdvice(gaps DefenseGap, threats ThreatAnalysis, towers []AITower, personality Personality) StrategicAdvice {
	advice := StrategicAdvice{
		Priority:      "balanced",
		Urgency:       0.3,
		Reason:        "目前防线还行",
		PreferredSlot: gaps.WeakestZone,
	}

	// 从高到低逐条检查，命中即返回
	if threats.ThreatLevel == "critical" && gaps.NeedsDPS {
		advice.Priority = "build_dps"
		advice.Urgency = 1.0
		advice.Reason = "火力不够，加输出"
		return applyPersonalityBias(advice, personality)
	}

	if threats.BossCount > 0 && gaps.NeedsAntiTank {
		advice.Priority = "upgrade"
		advice.Urgency = 0.9
		advice.Reason = "先升级现有的"
		return applyPersonalityBias(advice, personality)
	}

	if threats.TotalEnemies > 8 && gaps.NeedsAOE {
		advice.Priority = "build_aoe"
		advice.Urgency = 0.8
		advice.Reason = "怪太多了，要范围伤害"
		return applyPersonalityBias(advice, personality)
	}

	if gaps.NeedsCC {
		advice.Priority = "build_cc"
		advice.Urgency = 0.7
		advice.Reason = "缺控制，补个减速"
		return applyPersonalityBias(advice, personality)
	}

	if gaps.NeedsDPS {
		advice.Priority = "build_dps"
		advice.Urgency = 0.6
		advice.Reason = "需要更多伤害"
		return applyPersonalityBias(advice, personality)
	}

	// 有塔但无明显缺口 → 建议升级（如果塔够多）
	if len(towers) >= 3 && !gaps.NeedsDPS && !gaps.NeedsAOE && !gaps.NeedsCC {
		advice.Priority = "upgrade"
		advice.Urgency = 0.4
		advice.Reason = "升级比造新的划算"
	}

	return applyPersonalityBias(advice, personality)
}

// applyPersonalityBias 根据个性微调策略建议。
// 激进型(高 Aggression)倾向 DPS，防守型(低 Aggression)倾向 CC。
func applyPersonalityBias(advice StrategicAdvice, p Personality) StrategicAdvice {
	// 激进型：将 build_cc 降级为 build_dps（如果不是很紧急）
	if p.Aggression > 0.7 && advice.Priority == "build_cc" && advice.Urgency < 0.8 {
		advice.Priority = "build_dps"
		advice.Reason = "火力不够，加输出"
	}

	// 防守型：将 build_dps 升级为 build_cc（如果不是危急）
	if p.Aggression < 0.3 && advice.Priority == "build_dps" && advice.Urgency < 0.9 {
		advice.Priority = "build_cc"
		advice.Reason = "缺控制，补个减速"
	}

	// 经济型：偏好升级而非建造
	if p.Economy > 0.7 && advice.Urgency < 0.7 {
		urgencyBefore := advice.Urgency
		advice.Urgency = math.Max(0.1, advice.Urgency-0.15)
		_ = urgencyBefore // 个性调制只影响 urgency 权重
	}

	return advice
}
