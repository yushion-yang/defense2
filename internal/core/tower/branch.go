// branch.go — 分支特化系统。
// 每座塔可一次性选择一个分支，应用对应的属性加成。
package tower

// BranchConfig 分支特化配置。
type BranchConfig struct {
	RangeBonus        float64 // 射程加成倍率
	DamageBonus       float64 // 伤害加成倍率
	SplashRadiusBonus float64 // 溅射半径加成
	SplashDamageBonus float64 // 溅射伤害加成（加到 splashDamageMult 上）
	RemoveSplash      bool    // 移除溅射
	FireRateBonus     float64 // 攻速加成（减少 fireRate）
	FireRateFloor     float64 // 攻速下限（默认0.18）
	FireRatePenalty   float64 // 攻速惩罚倍率（如深冻分支）
	AttackStyle       string  // 切换攻击方式（如 "piercingLaser"）
	RangeMultiplier   float64 // 攻击方式切换时的射程乘数
	DamageMultiplier  float64 // 攻击方式切换时的伤害乘数
}

// defaultFireRateFloor 攻速下限默认值。
const defaultFireRateFloor = 0.18

// ApplyBranch 对塔应用分支特化，每座塔只能特化一次。
// 已特化的塔返回 false，成功返回 true。
func ApplyBranch(t *Tower, branchKey string, cfg BranchConfig) bool {
	if t.Branch != "" {
		return false
	}

	// 射程加成
	if cfg.RangeBonus != 0 {
		t.Range += t.BaseRange * cfg.RangeBonus
	}

	// 伤害加成
	if cfg.DamageBonus != 0 {
		t.Damage += t.BaseDamage * cfg.DamageBonus
	}

	// 攻速加成（减少冷却间隔）
	if cfg.FireRateBonus != 0 && t.AttackSpeed > 0 {
		interval := 1.0 / t.AttackSpeed
		interval -= cfg.FireRateBonus
		floor := cfg.FireRateFloor
		if floor <= 0 {
			floor = defaultFireRateFloor
		}
		if interval < floor {
			interval = floor
		}
		t.AttackSpeed = 1.0 / interval
	}

	// 攻速惩罚（乘法降低）
	if cfg.FireRatePenalty != 0 {
		t.AttackSpeed *= (1 - cfg.FireRatePenalty)
	}

	// 切换攻击方式
	if cfg.AttackStyle != "" {
		t.AttackStyleID = AttackStyle(cfg.AttackStyle)
		if cfg.RangeMultiplier != 0 {
			t.Range *= cfg.RangeMultiplier
		}
		if cfg.DamageMultiplier != 0 {
			t.Damage *= cfg.DamageMultiplier
		}
	}

	t.Branch = branchKey
	return true
}
