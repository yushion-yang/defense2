// passive_adapter.go — 被动技能到 CoreSkill 注册表的桥接适配器。
// 把 4 种被动技能注册为 CoreSkill，统一驱动。
package skill

import (
	"defense2/internal/core/enemy"
)

func init() {
	// 注册 4 种被动技能适配器
	Register("missileBarrage", func() CoreSkill {
		return newPassiveAdapter("missile-barrage")
	})
	Register("judgmentBeam", func() CoreSkill {
		return newPassiveAdapter("judgment-beam")
	})
	Register("chainLightningBolts", func() CoreSkill {
		return newPassiveAdapter("chain-lightning")
	})
	Register("judgmentRain", func() CoreSkill {
		return newPassiveAdapter("judgment-rain")
	})
}

// passiveAdapter 将 PassiveSkillState 包装为 CoreSkill。
type passiveAdapter struct {
	typeName string             // 被动技能类型名
	passive  *PassiveSkillState // 被动技能运行时状态
}

// newPassiveAdapter 创建被动技能适配器。
func newPassiveAdapter(typeName string) *passiveAdapter {
	def := GetPassiveDefaults(typeName)
	return &passiveAdapter{
		typeName: typeName,
		passive:  NewPassiveSkill(def),
	}
}

func (a *passiveAdapter) Name() string { return a.typeName }

func (a *passiveAdapter) Init(_ interface{}) {
	// 重置状态
	a.passive.Charge = 0
	a.passive.Ready = false
}

func (a *passiveAdapter) Tick(_ interface{}, _ []*enemy.Enemy, dt float64, _ *SkillContext) bool {
	a.passive.Tick(dt)

	// 充满时自动释放（被动技能自动触发）
	if a.passive.Ready {
		a.passive.TryRelease()
		return true // 释放帧压制普攻
	}

	return false
}

func (a *passiveAdapter) ShouldSuppressFire(_ interface{}) bool { return false }
func (a *passiveAdapter) ShouldSuppressMove(_ interface{}) bool { return false }

func (a *passiveAdapter) GetProgress(_ interface{}) (float64, bool) {
	return a.passive.Progress(), a.passive.Ready
}
