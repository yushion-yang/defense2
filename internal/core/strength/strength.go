// strength.go — 塔强度系统。
// 强度是塔的核心增幅属性，影响有效伤害和攻速。
// 来源：基础强度 + 临时增益（光环/buff）+ 链网络加成（相邻塔）。
package strength

import (
	"defense2/internal/core/tower"
)

// TowerStrength 单座塔的强度状态。
type TowerStrength struct {
	Base       float64 // 基础强度（默认 100）
	TempBonus  float64 // 临时增益总量（来自光环/buff，每帧重算）
	ChainBonus float64 // 链网络加成（来自相邻塔）
}

// Effective 返回有效强度 = Base + TempBonus + ChainBonus。
func (s *TowerStrength) Effective() float64 {
	return s.Base + s.TempBonus + s.ChainBonus
}

// DamageMultiplier 返回强度对伤害的乘数（100 强度 = 1.0x）。
func (s *TowerStrength) DamageMultiplier() float64 {
	return s.Effective() / 100.0
}

// SpeedMultiplier 返回强度对攻速的乘数（缩放较温和）。
func (s *TowerStrength) SpeedMultiplier() float64 {
	eff := s.Effective()
	if eff <= 100 {
		return 1.0
	}
	// 超过 100 的部分以 50% 效率影响攻速
	return 1.0 + (eff-100)*0.005
}

// DefaultStrength 创建默认强度（基础 100）。
func DefaultStrength() TowerStrength {
	return TowerStrength{Base: 100}
}

// CalcChainBonus 计算链网络加成：相邻塔每座贡献 5 点强度。
func CalcChainBonus(t *tower.Tower, allTowers *tower.Pool) float64 {
	bonus := 0.0
	allTowers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		// 相邻定义：行列差 ≤ 1
		dr := t.Row - other.Row
		dc := t.Col - other.Col
		if dr < 0 {
			dr = -dr
		}
		if dc < 0 {
			dc = -dc
		}
		if dr <= 1 && dc <= 1 {
			bonus += 5
		}
	})
	return bonus
}
