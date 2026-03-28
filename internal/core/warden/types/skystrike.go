// skystrike.go — 天降型战灵。
// Skystrike 是间接型战灵，定期在敌人最密集处释放 AoE 打击。
package types

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&SkystrikeBehavior{})
}

// SkystrikeState 天降的内部状态。
type SkystrikeState struct {
	AttackTimer    float64 // 攻击倒计时（秒）
	Damage         float64 // 单次伤害
	AttackInterval float64 // 攻击间隔（秒）
	AoERadius      float64 // 打击范围半径（像素）
	StrikeX        float64 // 上次打击位置 X（用于渲染）
	StrikeY        float64 // 上次打击位置 Y（用于渲染）
	StrikeTimer    float64 // 视觉效果倒计时（秒）
}

// SkystrikeBehavior 天降行为实现。
type SkystrikeBehavior struct{}

func (b *SkystrikeBehavior) Type() string { return "skystrike" }

func (b *SkystrikeBehavior) Init(w *warden.Warden) interface{} {
	return &SkystrikeState{
		Damage:         40,
		AttackInterval: 5.0,
		AoERadius:      60,
	}
}

func (b *SkystrikeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*SkystrikeState)
	if !ok {
		return
	}

	// 1. 视觉效果衰减
	if s.StrikeTimer > 0 {
		s.StrikeTimer -= ctx.DT
	}

	// 2. 攻击计时器
	s.AttackTimer -= ctx.DT
	if s.AttackTimer > 0 {
		return
	}
	s.AttackTimer += s.AttackInterval

	// 3. 找到最密集的敌人聚集点
	center := findDensestEnemy(ctx.Enemies, s.AoERadius)
	if center == nil {
		return
	}

	// 4. 对范围内所有敌人造成伤害
	cx, cy := center.X, center.Y
	s.StrikeX = cx
	s.StrikeY = cy
	s.StrikeTimer = 0.5

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if math.Hypot(e.X-cx, e.Y-cy) <= s.AoERadius {
			e.HP -= s.Damage
			if e.HP <= 0 {
				e.Active = false
				ctx.OnKill()
			}
		}
	})
}

// findDensestEnemy 找到邻居最多的敌人作为 AoE 中心。
func findDensestEnemy(enemies *enemy.Pool, radius float64) *enemy.Enemy {
	// 收集所有存活敌人的指针
	var alive []*enemy.Enemy
	enemies.Each(func(e *enemy.Enemy) {
		alive = append(alive, e)
	})
	if len(alive) == 0 {
		return nil
	}

	var best *enemy.Enemy
	bestCount := -1

	for _, candidate := range alive {
		count := 0
		for _, other := range alive {
			if math.Hypot(other.X-candidate.X, other.Y-candidate.Y) <= radius {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			best = candidate
		}
	}
	return best
}
