// features.go — 特征提取系统。
//
// 将游戏状态快照转换为结构化特征向量，供权重模型评分。
// 每个决策场景（造塔/升级/开波/经济）有独立的特征提取函数。
// 所有特征归一化到 [0,1] 区间，便于权重直接加权求和。
//
// 零 core 依赖：只接收基本类型参数，不 import aiplayer 以外的包。
package learning

import "math"

// ── 特征向量 ──────────────────────────────────────────

// FeatureVec 固定长度特征向量。
// 使用固定数组而非 map，零分配、缓存友好。
type FeatureVec struct {
	Values [MaxFeatures]float64
	Len    int // 实际使用的特征数量
}

// MaxFeatures 特征向量最大长度。
// 所有场景的特征数量上限，预留空间避免后续扩展改结构。
// 当前最大: econ 15 维，留余量到 24。
const MaxFeatures = 24

// ── 造塔位置特征 ──────────────────────────────────────────

// 造塔评分特征索引常量（14 维）。
const (
	FeatBuildPathDist         = iota // 路径距离分（离最近路径点越近越好）
	FeatBuildCoverage                // 路径覆盖分（射程内路径点密度）
	FeatBuildSpread                  // 塔间距分（与现有塔的分散程度）
	FeatBuildRedundancy              // 冗余惩罚（已有塔覆盖重叠度）
	FeatBuildSynergy                 // 协同加成（附近有 CC 塔）
	FeatBuildNoise                   // 随机噪声（拟人化）
	FeatBuildProgress                // 游戏进度（影响前/后期偏好）
	FeatBuildGoldRatio               // 金币充裕度（当前金/塔成本）
	FeatBuildTowerCount              // 现有塔数量归一化
	FeatBuildChokepoint              // 咽喉点（路径弯道附近，0/1）
	FeatBuildPathBendDist            // 到最近路径弯道的距离归一化
	FeatBuildNearestTowerDist        // 到最近现有塔的像素距离归一化
	FeatBuildWaveProgress            // 当前波次进度 wave/maxWaves
	FeatBuildBossNext                // 下一波是否 Boss（0/1）
	FeatBuildLen                     // 造塔特征总数（必须在最后）
)

// BuildCellInput 造塔评分的输入数据（纯值，零 core 依赖）。
type BuildCellInput struct {
	CellX, CellY    float64 // 候选格子像素坐标
	CellRow, CellCol int

	// 路径信息
	PathPoints             []PathPt
	MapCenterX, MapCenterY float64

	// 射程
	TowerRange float64

	// 现有塔信息
	ExistingTowers []TowerInfo

	// 经济和进度
	Gold, TowerCost int
	Progress        float64 // wave/maxWaves [0,1]
	TowerCount      int

	// 随机种子（可控复现）
	Noise float64

	// ── 新增特征输入（Phase 2）──
	BossNext bool    // 下一波是否 Boss
	CellSize int     // 网格单元尺寸（像素，用于咽喉检测）
}

// PathPt 路径点（像素坐标）。
type PathPt struct{ X, Y float64 }

// TowerInfo 塔的基础信息（特征提取用）。
type TowerInfo struct {
	Row, Col    int
	X, Y        float64 // 像素坐标
	Range       float64
	Damage      float64
	Strength    int
	AttackSpeed float64
	Kills       int
	Abilities   []string
	Cost        int
}

// ExtractBuildFeatures 提取造塔位置评分特征。
//
// 特征维度（全部 [0,1]）:
//   - PathDist: 离最近路径点越近越好
//   - Coverage: 射程内路径点密度
//   - Spread: 与现有塔的分散程度
//   - Redundancy: 1 - 重叠覆盖比例
//   - Synergy: 附近有 CC 塔
//   - Noise: 随机噪声
//   - Progress: 游戏进度
//   - GoldRatio: 金币充裕度
//   - TowerCount: 塔数量归一化
func ExtractBuildFeatures(in BuildCellInput) FeatureVec {
	var f FeatureVec
	f.Len = FeatBuildLen

	// ── 1. 路径距离 ──
	if len(in.PathPoints) > 0 {
		minDist := math.MaxFloat64
		for _, p := range in.PathPoints {
			d := math.Hypot(in.CellX-p.X, in.CellY-p.Y)
			if d < minDist {
				minDist = d
			}
		}
		f.Values[FeatBuildPathDist] = clamp01(1.0 - minDist/500.0)
	} else {
		cx, cy := in.MapCenterX, in.MapCenterY
		if cx == 0 && cy == 0 {
			cx, cy = 600, 270
		}
		dist := math.Hypot(in.CellX-cx, in.CellY-cy)
		f.Values[FeatBuildPathDist] = clamp01(1.0 - dist/400.0)
	}

	// ── 2. 路径覆盖 ──
	if len(in.PathPoints) > 0 {
		covered := 0
		for _, p := range in.PathPoints {
			if math.Hypot(in.CellX-p.X, in.CellY-p.Y) <= in.TowerRange {
				covered++
			}
		}
		f.Values[FeatBuildCoverage] = clamp01(float64(covered) / 10.0)
	}

	// ── 3. 塔间距 ──
	f.Values[FeatBuildSpread] = 1.0
	if len(in.ExistingTowers) > 0 {
		minDist := math.MaxFloat64
		for _, t := range in.ExistingTowers {
			dr := float64(in.CellRow - t.Row)
			dc := float64(in.CellCol - t.Col)
			d := math.Sqrt(dr*dr + dc*dc)
			if d < minDist {
				minDist = d
			}
		}
		f.Values[FeatBuildSpread] = clamp01(minDist / 5.0)
	}

	// ── 4. 冗余惩罚 ──
	redundancy := 0.0
	if len(in.PathPoints) > 0 && len(in.ExistingTowers) > 0 {
		coveredByCandidate := 0
		coveredByBoth := 0
		for _, p := range in.PathPoints {
			if math.Hypot(in.CellX-p.X, in.CellY-p.Y) <= in.TowerRange {
				coveredByCandidate++
				for _, t := range in.ExistingTowers {
					if math.Hypot(t.X-p.X, t.Y-p.Y) <= t.Range {
						coveredByBoth++
						break
					}
				}
			}
		}
		if coveredByCandidate > 0 {
			redundancy = float64(coveredByBoth) / float64(coveredByCandidate)
		}
	}
	f.Values[FeatBuildRedundancy] = clamp01(1.0 - redundancy)

	// ── 5. 协同加成 ──
	for _, t := range in.ExistingTowers {
		dr := float64(in.CellRow - t.Row)
		dc := float64(in.CellCol - t.Col)
		if math.Sqrt(dr*dr+dc*dc) <= 4 {
			for _, ab := range t.Abilities {
				if isCC(ab) {
					f.Values[FeatBuildSynergy] = 1.0
					break
				}
			}
		}
		if f.Values[FeatBuildSynergy] > 0 {
			break
		}
	}

	// ── 6-8. 辅助特征 ──
	f.Values[FeatBuildNoise] = in.Noise
	f.Values[FeatBuildProgress] = clamp01(in.Progress)
	if in.TowerCost > 0 {
		f.Values[FeatBuildGoldRatio] = clamp01(float64(in.Gold) / float64(in.TowerCost*5))
	}
	f.Values[FeatBuildTowerCount] = clamp01(float64(in.TowerCount) / 8.0)

	// ── 9. 咽喉点检测 ──
	cellSize := in.CellSize
	if cellSize <= 0 {
		cellSize = 60
	}
	if IsChokepoint(in.PathPoints, in.CellX, in.CellY, cellSize) {
		f.Values[FeatBuildChokepoint] = 1.0
	}

	// ── 10. 到最近路径弯道的距离 ──
	f.Values[FeatBuildPathBendDist] = clamp01(1.0 - NearestPathBendDist(in.PathPoints, in.CellX, in.CellY)/400.0)

	// ── 11. 到最近现有塔的距离 ──
	if len(in.ExistingTowers) > 0 {
		minTowerDist := math.MaxFloat64
		for _, t := range in.ExistingTowers {
			d := math.Hypot(in.CellX-t.X, in.CellY-t.Y)
			if d < minTowerDist {
				minTowerDist = d
			}
		}
		f.Values[FeatBuildNearestTowerDist] = clamp01(minTowerDist / 400.0)
	} else {
		f.Values[FeatBuildNearestTowerDist] = 1.0 // 无塔 → 距离远
	}

	// ── 12. 波次进度（重复 progress，显式供神经网络单独学习权重）──
	f.Values[FeatBuildWaveProgress] = clamp01(in.Progress)

	// ── 13. 下一波 Boss ──
	if in.BossNext {
		f.Values[FeatBuildBossNext] = 1.0
	}

	return f
}

// ── 升级评分特征 ──────────────────────────────────────────

// 升级评分特征索引常量（11 维）。
const (
	FeatUpgROI              = iota // 预期 DPS 提升 / 升级成本
	FeatUpgPathCoverage            // 咽喉要塔（覆盖路径点多）
	FeatUpgHeadroom                // 强度提升空间（低 str 边际收益高）
	FeatUpgKills                   // 击杀贡献（有效位置验证）
	FeatUpgAbilityCount            // 能力协同（能力多→升级收益高）
	FeatUpgProgress                // 游戏进度
	FeatUpgThreatLevel             // 当前威胁等级
	FeatUpgCostEfficiency          // 费效比 (damage*attackSpeed)/totalCost
	FeatUpgAbilitySlotsFull        // 能力槽使用率 filled/6
	FeatUpgIsChokepoint            // 塔是否在咽喉点（0/1）
	FeatUpgGoldAfterUpgrade        // 升级后剩余金币归一化
	FeatUpgLen                     // 升级特征总数
)

// UpgradeTowerInput 升级评分的输入数据。
type UpgradeTowerInput struct {
	Tower       TowerInfo
	PathPoints  []PathPt
	UpgCost     int
	Progress    float64
	ThreatLevel float64 // 0=low, 0.5=medium, 1.0=high/critical

	// ── 新增输入（Phase 2）──
	Gold     int // 当前可用金币（用于计算升级后剩余）
	CellSize int // 网格单元尺寸（像素，用于咽喉检测）
}

// ExtractUpgradeFeatures 提取升级评分特征。
func ExtractUpgradeFeatures(in UpgradeTowerInput) FeatureVec {
	var f FeatureVec
	f.Len = FeatUpgLen

	// ── 1. ROI 估算 ──
	str := float64(in.Tower.Strength)
	if str <= 0 {
		str = 100
	}
	improvement := in.Tower.Damage * 10.0 / str
	cost := float64(in.UpgCost)
	if cost <= 0 {
		cost = 10
	}
	f.Values[FeatUpgROI] = clamp01(improvement / cost * 0.5)

	// ── 2. 路径覆盖 ──
	if len(in.PathPoints) > 0 {
		covered := 0
		for _, p := range in.PathPoints {
			if math.Hypot(in.Tower.X-p.X, in.Tower.Y-p.Y) <= in.Tower.Range {
				covered++
			}
		}
		f.Values[FeatUpgPathCoverage] = clamp01(float64(covered) / 10.0)
	} else {
		f.Values[FeatUpgPathCoverage] = 0.5
	}

	// ── 3. 强度提升空间 ──
	f.Values[FeatUpgHeadroom] = clamp01(1.0 - float64(in.Tower.Strength-100)/200.0)

	// ── 4. 击杀贡献 ──
	f.Values[FeatUpgKills] = clamp01(float64(in.Tower.Kills) / 10.0)

	// ── 5. 能力协同 ──
	f.Values[FeatUpgAbilityCount] = clamp01(float64(len(in.Tower.Abilities)) / 4.0)

	// ── 6-7. 辅助特征 ──
	f.Values[FeatUpgProgress] = clamp01(in.Progress)
	f.Values[FeatUpgThreatLevel] = clamp01(in.ThreatLevel)

	// ── 8. 费效比 ──
	as := in.Tower.AttackSpeed
	if as <= 0 {
		as = 1.0
	}
	dps := in.Tower.Damage * as
	totalCost := float64(in.Tower.Cost)
	if totalCost <= 0 {
		totalCost = 50
	}
	f.Values[FeatUpgCostEfficiency] = clamp01(dps / totalCost * 0.1)

	// ── 9. 能力槽使用率 ──
	f.Values[FeatUpgAbilitySlotsFull] = clamp01(float64(len(in.Tower.Abilities)) / 6.0)

	// ── 10. 咽喉点检测 ──
	cellSize := in.CellSize
	if cellSize <= 0 {
		cellSize = 60
	}
	towerPixelX := float64(in.Tower.Col)*float64(cellSize) + float64(cellSize)/2.0
	towerPixelY := float64(in.Tower.Row)*float64(cellSize) + float64(cellSize)/2.0
	if IsChokepoint(in.PathPoints, towerPixelX, towerPixelY, cellSize) {
		f.Values[FeatUpgIsChokepoint] = 1.0
	}

	// ── 11. 升级后剩余金币 ──
	if in.Gold > 0 {
		remaining := float64(in.Gold - in.UpgCost)
		if remaining < 0 {
			remaining = 0
		}
		f.Values[FeatUpgGoldAfterUpgrade] = clamp01(remaining / float64(in.Gold))
	}

	return f
}

// ── 经济决策特征 ──────────────────────────────────────────

// 经济决策（造塔 vs 升级）特征索引常量（15 维）。
const (
	FeatEconProgress           = iota // 游戏进度
	FeatEconTowerCount                // 现有塔数量
	FeatEconGoldRatio                 // 可用金/总金
	FeatEconUrgency                   // 策略紧迫度
	FeatEconThreatLevel               // 威胁等级
	FeatEconAggression                // 个性: 激进度
	FeatEconEconomy                   // 个性: 经济偏好
	FeatEconAdviceBuild               // 策略建议偏造塔（0/1）
	FeatEconAdviceUpgrade             // 策略建议偏升级（0/1）
	FeatEconBossNext                  // 下一波是否 Boss
	FeatEconWavesSinceLastBuild       // 自上次造塔以来的波数归一化
	FeatEconEnemyHPTrend              // 敌人血量趋势（本波/上波）
	FeatEconLivesPercent              // 当前生命/最大生命
	FeatEconTowerDPSvsEnemyHP         // 全塔 DPS / 本波预期敌人总 HP
	FeatEconPerfectWaveStreak         // 连续完美通过波数归一化
	FeatEconLen                       // 经济特征总数
)

// EconInput 经济决策的输入数据。
type EconInput struct {
	Progress       float64 // wave/maxWaves
	TowerCount     int
	Gold           int
	TotalGold      int     // Gold + reserve (总金币池)
	Urgency        float64 // 策略紧迫度
	ThreatLevel    float64 // 威胁等级 [0,1]
	Aggression     float64 // 个性
	Economy        float64 // 个性
	AdvicePriority string  // "build_cc"/"build_dps"/"upgrade"/"balanced" 等
	BossNext       bool

	// ── 新增输入（Phase 2）──
	WavesSinceLastBuild int     // 自上次造塔以来经过的波数
	EnemyHPTrend        float64 // 本波平均敌人 HP / 上波平均敌人 HP（>1 变硬）
	Lives               int     // 当前生命
	MaxLives            int     // 最大生命
	TotalTowerDPS       float64 // 所有塔的 DPS 总和
	WaveEnemyTotalHP    float64 // 本波预期敌人总 HP
	PerfectWaveStreak   int     // 连续完美通过的波数
}

// ExtractEconFeatures 提取经济决策特征（造塔 vs 升级偏好）。
func ExtractEconFeatures(in EconInput) FeatureVec {
	var f FeatureVec
	f.Len = FeatEconLen

	f.Values[FeatEconProgress] = clamp01(in.Progress)
	f.Values[FeatEconTowerCount] = clamp01(float64(in.TowerCount) / 8.0)

	if in.TotalGold > 0 {
		f.Values[FeatEconGoldRatio] = clamp01(float64(in.Gold) / float64(in.TotalGold))
	} else {
		f.Values[FeatEconGoldRatio] = 0
	}

	f.Values[FeatEconUrgency] = clamp01(in.Urgency)
	f.Values[FeatEconThreatLevel] = clamp01(in.ThreatLevel)
	f.Values[FeatEconAggression] = clamp01(in.Aggression)
	f.Values[FeatEconEconomy] = clamp01(in.Economy)

	// 策略建议编码为 0/1 特征
	switch in.AdvicePriority {
	case "build_cc", "build_dps", "build_aoe":
		f.Values[FeatEconAdviceBuild] = 1.0
	case "upgrade":
		f.Values[FeatEconAdviceUpgrade] = 1.0
	}

	if in.BossNext {
		f.Values[FeatEconBossNext] = 1.0
	}

	// ── 新增特征（Phase 2）──

	// 自上次造塔以来的波数（归一化到 [0,1]，10 波封顶）
	f.Values[FeatEconWavesSinceLastBuild] = clamp01(float64(in.WavesSinceLastBuild) / 10.0)

	// 敌人血量趋势（>1 = 变硬，用 sigmoid 映射到 [0,1]）
	if in.EnemyHPTrend > 0 {
		// trend=1 → 0.5, trend=2 → ~0.73, trend=0.5 → ~0.27
		f.Values[FeatEconEnemyHPTrend] = clamp01(1.0 / (1.0 + math.Exp(-2.0*(in.EnemyHPTrend-1.0))))
	} else {
		f.Values[FeatEconEnemyHPTrend] = 0.5 // 无数据 → 中性
	}

	// 生命百分比
	if in.MaxLives > 0 {
		f.Values[FeatEconLivesPercent] = clamp01(float64(in.Lives) / float64(in.MaxLives))
	} else {
		f.Values[FeatEconLivesPercent] = 1.0
	}

	// 全塔 DPS vs 敌人总 HP（DPS 充足时偏升级，不足时偏造塔）
	if in.WaveEnemyTotalHP > 0 {
		f.Values[FeatEconTowerDPSvsEnemyHP] = clamp01(in.TotalTowerDPS / in.WaveEnemyTotalHP)
	} else {
		f.Values[FeatEconTowerDPSvsEnemyHP] = 0.5
	}

	// 连续完美波数（归一化到 [0,1]，5 波封顶）
	f.Values[FeatEconPerfectWaveStreak] = clamp01(float64(in.PerfectWaveStreak) / 5.0)

	return f
}

// ── 咽喉点检测 ──────────────────────────────────────────

// IsChokepoint 判断指定像素坐标是否靠近路径弯道（咽喉点）。
//
// 检测逻辑：在候选格子 2 个 cellSize 半径内找到最近的路径段，
// 检查该段前后路径方向是否有显著变化（>30°）。
// 路径弯道处建塔能覆盖更多路径长度，是高价值位置。
func IsChokepoint(pathPoints []PathPt, cellX, cellY float64, cellSize int) bool {
	if len(pathPoints) < 3 {
		return false
	}

	searchRadius := float64(cellSize) * 2.0

	for i := 1; i < len(pathPoints)-1; i++ {
		// 检查路径点是否在搜索半径内
		if math.Hypot(pathPoints[i].X-cellX, pathPoints[i].Y-cellY) > searchRadius {
			continue
		}

		// 计算前后段的方向变化角度
		dx1 := pathPoints[i].X - pathPoints[i-1].X
		dy1 := pathPoints[i].Y - pathPoints[i-1].Y
		dx2 := pathPoints[i+1].X - pathPoints[i].X
		dy2 := pathPoints[i+1].Y - pathPoints[i].Y

		// 用叉积和点积算角度变化
		cross := dx1*dy2 - dy1*dx2
		dot := dx1*dx2 + dy1*dy2
		angle := math.Abs(math.Atan2(cross, dot))

		// >30° 视为弯道
		if angle > math.Pi/6 {
			return true
		}
	}
	return false
}

// NearestPathBendDist 返回到最近路径弯道的像素距离。
// 无弯道时返回一个大值（999）。
func NearestPathBendDist(pathPoints []PathPt, cellX, cellY float64) float64 {
	if len(pathPoints) < 3 {
		return 999.0
	}

	minDist := 999.0
	for i := 1; i < len(pathPoints)-1; i++ {
		dx1 := pathPoints[i].X - pathPoints[i-1].X
		dy1 := pathPoints[i].Y - pathPoints[i-1].Y
		dx2 := pathPoints[i+1].X - pathPoints[i].X
		dy2 := pathPoints[i+1].Y - pathPoints[i].Y

		cross := dx1*dy2 - dy1*dx2
		dot := dx1*dx2 + dy1*dy2
		angle := math.Abs(math.Atan2(cross, dot))

		if angle > math.Pi/6 {
			d := math.Hypot(pathPoints[i].X-cellX, pathPoints[i].Y-cellY)
			if d < minDist {
				minDist = d
			}
		}
	}
	return minDist
}

// ── 辅助函数 ──────────────────────────────────────────

// clamp01 钳制到 [0,1]。
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// isCC 判断能力是否为控制类。
func isCC(ability string) bool {
	switch ability {
	case "slow", "stun", "root", "freeze":
		return true
	}
	return false
}
