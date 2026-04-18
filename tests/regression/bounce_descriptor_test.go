// bounce_descriptor_test.go — 回归测试：bounce 描述符能力必须返回 HitResult.Bounce。
//
// Bug: bounce 描述符用 ChainSelector，但 onHit 上下文没有 Enemies 池，
// selector 返回 nil，弹射从未触发。同 splash 的模式。
package regression_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// TestRegression_BounceDescriptorReturnsBounceEffect 验证 bounce 描述符返回 HitResult.Bounce。
func TestRegression_BounceDescriptorReturnsBounceEffect(t *testing.T) {
	ab, ok := tower.Registry["bounce"]
	if !ok {
		t.Fatal("bounce ability not registered")
	}

	tw := &tower.Tower{
		Damage: 100,
		Range:  200,
	}
	tw.Abilities = []string{"bounce"}
	p := &projectile.Projectile{Damage: 100}
	e := &enemy.Enemy{HP: 500, MaxHP: 500, X: 100, Y: 100}
	e.Active = true

	hr := ab.OnHit(tw, p, e)
	if hr == nil {
		t.Fatal("bounce OnHit returned nil HitResult")
	}
	if hr.Bounce == nil {
		t.Fatal("bounce OnHit returned HitResult without Bounce effect")
	}
	if hr.Bounce.MaxBounces < 1 {
		t.Errorf("Bounce.MaxBounces = %d, want >= 1", hr.Bounce.MaxBounces)
	}
	if hr.Bounce.Range <= 0 {
		t.Errorf("Bounce.Range = %v, want > 0", hr.Bounce.Range)
	}
	if hr.Bounce.DamageRatio <= 0 || hr.Bounce.DamageRatio > 1 {
		t.Errorf("Bounce.DamageRatio = %v, want (0, 1]", hr.Bounce.DamageRatio)
	}
	if hr.Bounce.SrcDamage != 100 {
		t.Errorf("Bounce.SrcDamage = %v, want 100 (tower damage)", hr.Bounce.SrcDamage)
	}
}
