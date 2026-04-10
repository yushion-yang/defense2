// thunder_strike.go — 雷击系统。
// 定时对范围内非 Boss 敌人施加百分比伤害 + 溅射。
package combat

import (
	"math"
	"sort"

	"defense2/internal/core/enemy"
)

// ThunderStrikeConfig 雷击配置。
type ThunderStrikeConfig struct {
	Cooldown    float64 // 冷却时间（秒）
	DamageRatio float64 // 伤害比例（占目标 maxHP）
	FlatDamage  float64 // 固定伤害加成
	Radius      float64 // 溅射范围
	SplashRatio float64 // 溅射伤害比例（默认 0.4）
	Targets     int     // 每次打击目标数
	TargetMode  string  // 目标选择模式（random/highestHp/nearest/lowest）
	Range       float64 // 选择范围
}

// ThunderStrikeState 雷击运行时状态。
type ThunderStrikeState struct {
	Timer float64 // 冷却计时器
}

// ThunderHit 单次雷击命中结果。
type ThunderHit struct {
	EnemyID   int     // 命中敌人 ID
	X, Y      float64 // 命中坐标
	Damage    float64 // 造成伤害
	IsPrimary bool    // 是否主目标（非溅射）
}

// NewThunderStrikeState 创建雷击运行时状态。
func NewThunderStrikeState(cfg ThunderStrikeConfig) *ThunderStrikeState {
	return &ThunderStrikeState{
		Timer: cfg.Cooldown,
	}
}

// TickThunderStrike 每帧驱动雷击系统。
// 冷却到期时选择目标并施加伤害，返回是否发射和命中列表。
func TickThunderStrike(state *ThunderStrikeState, cfg ThunderStrikeConfig, enemies []*enemy.Enemy, ownerX, ownerY, dt float64) (fired bool, hits []ThunderHit) {
	if state == nil {
		return false, nil
	}

	state.Timer -= dt
	if state.Timer > 0 {
		return false, nil
	}

	// 重置冷却
	state.Timer = cfg.Cooldown

	// 选择主目标
	targets := selectThunderTargets(enemies, ownerX, ownerY, cfg)
	if len(targets) == 0 {
		return false, nil
	}

	splashRatio := cfg.SplashRatio
	if splashRatio <= 0 {
		splashRatio = 0.4
	}

	for _, t := range targets {
		// 百分比 maxHP + 固定伤害
		rawDmg := t.MaxHP*cfg.DamageRatio + cfg.FlatDamage
		if rawDmg < 1 {
			rawDmg = 1
		}
		// 走伤害管线（免疫/上限/阈值/死亡检查）
		r := ProcessDamage(DamageInput{
			Target:     t,
			RawDamage:  rawDmg,
			DamageType: DmgMagic,
		})

		hits = append(hits, ThunderHit{
			EnemyID:   t.ID,
			X:         t.X,
			Y:         t.Y,
			Damage:    r.FinalDamage,
			IsPrimary: true,
		})

		// 溅射：对主目标附近非 Boss 敌人施加衰减伤害
		if cfg.Radius > 0 {
			splashRaw := rawDmg * splashRatio
			for _, e := range enemies {
				if !e.Active || e.HP <= 0 || e.Boss || e.ID == t.ID || e.IsDying() || e.IsSpawning() {
					continue
				}
				dx := e.X - t.X
				dy := e.Y - t.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= cfg.Radius {
					sr := ProcessDamage(DamageInput{
						Target:     e,
						RawDamage:  splashRaw,
						DamageType: DmgMagic,
					})
					hits = append(hits, ThunderHit{
						EnemyID:   e.ID,
						X:         e.X,
						Y:         e.Y,
						Damage:    sr.FinalDamage,
						IsPrimary: false,
					})
				}
			}
		}
	}

	return true, hits
}

// selectThunderTargets 选择雷击目标（过滤存活 + 非 Boss + 范围内，按模式排序）。
func selectThunderTargets(enemies []*enemy.Enemy, ownerX, ownerY float64, cfg ThunderStrikeConfig) []*enemy.Enemy {
	var candidates []*enemy.Enemy
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 || e.Boss {
			continue
		}
		dx := e.X - ownerX
		dy := e.Y - ownerY
		dist := math.Sqrt(dx*dx + dy*dy)
		if cfg.Range > 0 && dist > cfg.Range {
			continue
		}
		candidates = append(candidates, e)
	}

	if len(candidates) == 0 {
		return nil
	}

	// 按模式排序
	switch cfg.TargetMode {
	case "highestHp":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].HP > candidates[j].HP
		})
	case "nearest":
		sort.Slice(candidates, func(i, j int) bool {
			di := math.Sqrt(sqDist(candidates[i].X, candidates[i].Y, ownerX, ownerY))
			dj := math.Sqrt(sqDist(candidates[j].X, candidates[j].Y, ownerX, ownerY))
			return di < dj
		})
	case "lowest":
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].HP < candidates[j].HP
		})
	default:
		// "random" — 保持原始顺序（伪随机）
	}

	maxTargets := cfg.Targets
	if maxTargets <= 0 {
		maxTargets = 1
	}
	if maxTargets > len(candidates) {
		maxTargets = len(candidates)
	}
	return candidates[:maxTargets]
}

// sqDist 计算两点距离的平方。
func sqDist(x1, y1, x2, y2 float64) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return dx*dx + dy*dy
}
