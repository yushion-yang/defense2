// chain.go — 能量串联型战灵。
// Chain 是间接型战灵，无物理实体。
// 被动增强所有塔的伤害，并定期对随机敌人造成直接伤害。
package types

import (
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&ChainBehavior{})
}

// ChainState 能量串联的内部状态。
type ChainState struct {
	AttackTimer    float64            // 攻击倒计时（秒）
	Damage         float64            // 单次伤害
	AttackInterval float64            // 攻击间隔（秒）
	TowerBonus     float64            // 每座塔的伤害加成
	BoostedSet     map[*tower.Tower]bool // 已加成的塔集合
}

// ChainBehavior 能量串联行为实现。
type ChainBehavior struct{}

func (b *ChainBehavior) Type() string { return "chain" }

func (b *ChainBehavior) Init(w *warden.Warden) interface{} {
	return &ChainState{
		Damage:         15,
		AttackInterval: 2.0,
		TowerBonus:     10,
		BoostedSet:     make(map[*tower.Tower]bool),
	}
}

func (b *ChainBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*ChainState)
	if !ok {
		return
	}

	// 1. 塔增强：为新建的塔添加伤害加成
	ctx.Towers.Each(func(t *tower.Tower) {
		if !s.BoostedSet[t] {
			t.Damage += s.TowerBonus
			s.BoostedSet[t] = true
		}
	})

	// 2. 攻击计时器
	s.AttackTimer -= ctx.DT
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval

		// 选择随机存活敌人
		target := pickRandomEnemy(ctx.Enemies)
		if target != nil {
			target.HP -= s.Damage
			if target.HP <= 0 {
				target.Active = false
				ctx.OnKill()
			}
		}
	}
}

// pickRandomEnemy 从存活敌人中随机选一个。
func pickRandomEnemy(enemies *enemy.Pool) *enemy.Enemy {
	// 先计数
	count := 0
	enemies.Each(func(_ *enemy.Enemy) {
		count++
	})
	if count == 0 {
		return nil
	}

	// 选随机索引
	pick := rand.Intn(count)
	var result *enemy.Enemy
	i := 0
	enemies.Each(func(e *enemy.Enemy) {
		if i == pick {
			result = e
		}
		i++
	})
	return result
}
