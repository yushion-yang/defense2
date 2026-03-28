// economy.go — 经济类能力实现。
// 提供金币生成和费用减免等经济效果。命中时不触发效果，由经济子系统 tick 处理。
package abilities

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&GoldOnKill{})
	tower.Register(&GoldPassive{})
	tower.Register(&Interest{})
}

// GoldOnKill 击杀赏金：该塔击杀敌人时额外获得金币。
type GoldOnKill struct{}
func (a *GoldOnKill) Name() string { return "goldOnKill" }
func (a *GoldOnKill) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 击杀时由经济管线处理
}

// GoldPassive 被动产金：每隔一段时间自动获得少量金币。
type GoldPassive struct{}
func (a *GoldPassive) Name() string { return "goldPassive" }
func (a *GoldPassive) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// Interest 利息：波次结算时根据当前金币获得额外收入。
type Interest struct{}
func (a *Interest) Name() string { return "interest" }
func (a *Interest) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
