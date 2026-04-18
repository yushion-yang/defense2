// splash_descriptor_test.go — 回归测试：splash 描述符能力必须返回 HitResult.Splash。
//
// Bug: splash 描述符用 aoeRadius selector + damage effect，但 onHit 上下文
// 没有 Enemies 池，selector 返回 nil，导致溅射从未触发。
// 修复方案：检测 splash 模式后直接合成 HitResult.Splash。
package regression_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// TestRegression_SplashDescriptorReturnsSplashEffect 验证 splash 描述符能力返回 HitResult.Splash。
func TestRegression_SplashDescriptorReturnsSplashEffect(t *testing.T) {
	ab, ok := tower.Registry["splash"]
	if !ok {
		t.Fatal("splash ability not registered")
	}

	tw := &tower.Tower{
		Damage: 100,
		Range:  200,
	}
	tw.Abilities = []string{"splash"}
	p := &projectile.Projectile{Damage: 100}
	e := &enemy.Enemy{
		HP:    500,
		MaxHP: 500,
		X:     100,
		Y:     100,
	}
	e.Active = true

	hr := ab.OnHit(tw, p, e)
	if hr == nil {
		t.Fatal("splash OnHit returned nil HitResult")
	}
	if hr.Splash == nil {
		t.Fatal("splash OnHit returned HitResult without Splash effect")
	}
	if hr.Splash.Radius <= 0 {
		t.Errorf("Splash.Radius = %v, want > 0", hr.Splash.Radius)
	}
	if hr.Splash.Ratio <= 0 || hr.Splash.Ratio > 1 {
		t.Errorf("Splash.Ratio = %v, want (0, 1]", hr.Splash.Ratio)
	}
}
