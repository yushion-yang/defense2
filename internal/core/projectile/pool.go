// pool.go — 弹射物环形缓冲区对象池：发射、追踪、回收。
//
// ═══════════════════════════════════════════════════════════════════
// 为什么用环形缓冲区而不是线性数组池？
// ═══════════════════════════════════════════════════════════════════
//
// 弹射物有极高的创建/销毁频率（gatling 塔每秒 12 发），特点：
//   - 生命周期短（2-3 秒）
//   - FIFO 特性强（先发射的先命中/消失）
//   - 不需要稳定指针（弹射物不被外部系统长期引用）
//
// 环形缓冲区优势：
//   - cursor 直接写入下一槽位，O(1) 发射（vs 线性扫描 O(n) 找空位）
//   - FIFO 回收：cursor 总是覆盖最旧的槽位
//   - 池满时自动覆盖最旧的弹射物（优雅降级，不丢帧）
//
// ═══════════════════════════════════════════════════════════════════
// ABA 问题防护
// ═══════════════════════════════════════════════════════════════════
//
// 当弹射物 A 锁定敌人 E1，E1 被杀后槽位复用给 E2，弹射物 A 可能误追踪 E2。
// 防护机制：发射时记录 TargetID（敌人递增 ID），每帧检查：
//
//	Target.Active && Target.ID == TargetID → 继续追踪
//	否则 → 弹射物立即消失
//
// ═══════════════════════════════════════════════════════════════════
// 弹射物生命周期（塔防模型）
// ═══════════════════════════════════════════════════════════════════
//
// 本游戏是塔防，弹射物行为遵循塔防惯例：
//
//  1. 发射时锁定目标（Target != nil），每帧重算朝向完美追踪
//  2. 碰撞检测只对锁定目标生效，穿过其他敌人（见 tick_combat.go）
//  3. 命中目标 → 触发能力 + 伤害 → 回收
//  4. 目标被其他弹先杀死 → 本弹直接消失（不继续飞行）
//
// 三种弹射物类型：
//   - 普通弹(Fire)：追踪目标，命中后回收，MaxLife=3s
//   - 弹射弹(FireBounce)：命中后弹向下一目标，携带已命中 ID 列表避免重复，MaxLife=2s
//   - 穿透弹(FirePenetrate)：直线飞行无追踪，对路径上所有敌人做碰撞检测，
//     MaxLife=飞行距离/速度*1.1
//
// 关联文件：
//   - projectile.go: Projectile struct 定义
//   - tick_combat.go: 碰撞检测与命中处理
//   - physics/grid.go: 穿透弹用空间网格加速碰撞检测
package projectile

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
)

const (
	normalProjectileMaxLife   = 3.0  // 普通弹射物最大存活时间(秒)
	bounceProjectileMaxLife   = 2.0  // 弹射弹射物最大存活时间(秒)
	penetrateProjectileRadius = 4    // 穿透弹射物碰撞半径(像素)
	penetrateLifeOvershoot    = 1.1  // 穿透弹射物寿命余量倍率
	projectileBoundaryMargin  = 50.0 // 弹射物屏幕外回收边距(像素)
)

// Pool 环形缓冲区弹射物对象池。
// cursor 指向下一个写入位置，每次 Fire 后 cursor = (cursor+1) % cap。
// Count 跟踪存活数量（Active=true），用于 HUD 显示和性能监控。
type Pool struct {
	projectiles []Projectile // 预分配的弹射物槽位数组（默认 1024）
	cursor      int          // 下一个写入位置（环形递增，到末尾回绕到 0）
	Count       int          // 当前存活弹射物数量

	// 地图边界（用于弹射物出界回收）
	mapWidth  float64 // 地图像素宽度（0=使用默认 2400）
	mapHeight float64 // 地图像素高度（0=使用默认 1200）
}

// 默认地图边界（足够容纳最大地图）
const (
	defaultMapWidth  = 2400.0
	defaultMapHeight = 1200.0
)

// NewPool 创建指定容量的弹射物对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		projectiles: make([]Projectile, cap),
	}
}

// DefaultPool 创建默认容量（MaxProjectiles=1024）的弹射物对象池。
func DefaultPool() *Pool {
	return NewPool(game.MaxProjectiles)
}

// SetMapBounds 设置地图像素边界（用于弹射物出界回收）。
func (p *Pool) SetMapBounds(width, height float64) {
	p.mapWidth = width
	p.mapHeight = height
}

// MapBounds 返回当前地图边界（0 值时返回默认值）。
func (p *Pool) MapBounds() (width, height float64) {
	w := p.mapWidth
	if w <= 0 {
		w = defaultMapWidth
	}
	h := p.mapHeight
	if h <= 0 {
		h = defaultMapHeight
	}
	return w, h
}

// Fire 从 (sx,sy) 向 (tx,ty) 发射一颗普通追踪弹射物。
// target 非 nil 时启用追踪（每帧重新计算朝向），否则按初始方向直线飞行。
// towerKey 用于碰撞时查找来源塔触发能力（格式 "key_row_col"）。
//
// 实现细节：cursor 位置如果已有活跃弹射物，会被覆盖（Count--），
// 这是环形缓冲区的优雅降级——极端情况下最旧的弹射物被牺牲。
func (p *Pool) Fire(sx, sy, tx, ty, damage, speed, radius float64, target *enemy.Enemy, towerKey string) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零所有旧字段，防止复用槽位残留

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = normalProjectileMaxLife
	proj.Life = proj.MaxLife
	proj.Target = target
	if target != nil {
		proj.TargetID = target.ID
	}
	proj.SourceTowerKey = towerKey

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FireWithExecute 发射带斩杀判定的弹射物（机甲战灵专用）。
// execHpPct > 0 时：命中时若目标非 Boss 且 HP < MaxHP*execHpPct，则秒杀（伤害=当前 HP）。
func (p *Pool) FireWithExecute(sx, sy, tx, ty, damage, speed, radius float64, target *enemy.Enemy, towerKey string, execHpPct float64) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{}

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = normalProjectileMaxLife
	proj.Life = proj.MaxLife
	proj.Target = target
	if target != nil {
		proj.TargetID = target.ID
	}
	proj.SourceTowerKey = towerKey
	proj.ExecuteHpPct = execHpPct

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FireBounce 发射一颗弹射子弹（从前一次命中位置飞向新目标）。
// bounceCount 记录已弹射次数（用于弹射次数上限检查），
// hitIDs 记录已命中的敌人 ID 列表（避免同一敌人被同一链弹射重复命中）。
func (p *Pool) FireBounce(sx, sy float64, target *enemy.Enemy, damage, speed, radius float64, towerKey string, bounceCount int, hitIDs []int) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零所有旧字段

	dx := target.X - sx
	dy := target.Y - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = bounceProjectileMaxLife
	proj.Life = proj.MaxLife
	proj.Target = target
	if target != nil {
		proj.TargetID = target.ID
	}
	proj.SourceTowerKey = towerKey
	proj.BounceCount = bounceCount
	proj.BounceHitIDs = hitIDs

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// Tick 每帧调用，处理所有弹射物的移动和生命周期。
//
// 每个活跃弹射物的处理流程：
//  1. 追踪：有目标 → 检查 ABA（Active && ID 匹配）→ 重算朝向 / 目标失效 → 消失
//  2. 记录拖尾位置（移动前的坐标存入 Trail 环形缓冲）
//  3. 移动：X += VX*dt, Y += VY*dt
//  4. 散射弹飞行距离限制：超过 MaxRange 则回收
//  5. 超时(Life<=0)或飞出屏幕边界(50px margin) → 回收
func (p *Pool) Tick(dt float64) {
	for i := range p.projectiles {
		proj := &p.projectiles[i]
		if !proj.Active {
			continue
		}

		// ── 追踪与 ABA 校验 ──
		// 所有有目标的弹（普通弹/弹射弹/蓄力弹）统一行为：
		//   - 目标存活且 ID 匹配 → 每帧重算朝向，完美追踪（塔防惯例：弹必中）
		//   - 目标死亡或 ID 不匹配（槽位被复用 = ABA 问题）→ 弹射物立即消失
		// 注意：穿透弹 Target=nil，不走此分支
		if proj.Target != nil {
			if proj.Target.Active && proj.Target.ID == proj.TargetID {
				dx := proj.Target.X - proj.X
				dy := proj.Target.Y - proj.Y
				dist := math.Hypot(dx, dy)
				if dist > 1 {
					proj.VX = (dx / dist) * proj.Speed
					proj.VY = (dy / dist) * proj.Speed
				}
			} else {
				proj.Active = false
				proj.Target = nil
				p.Count--
				continue
			}
		}

		// 记录拖尾位置（移动前）
		proj.Trail[proj.TrailCursor] = TrailPoint{X: proj.X, Y: proj.Y, Active: true}
		proj.TrailCursor = (proj.TrailCursor + 1) % TrailLen

		proj.X += proj.VX * dt
		proj.Y += proj.VY * dt
		proj.Life -= dt

		// 散射弹飞行距离限制
		if proj.MaxRange > 0 {
			dx := proj.X - proj.StartX
			dy := proj.Y - proj.StartY
			if math.Hypot(dx, dy) >= proj.MaxRange {
				proj.Active = false
				proj.Target = nil
				p.Count--
				continue
			}
		}

		// 超时或飞出地图边界则回收（使用动态地图尺寸而非固定 ScreenHeight）
		mapW, mapH := p.MapBounds()
		if proj.Life <= 0 || proj.X < -projectileBoundaryMargin || proj.X > mapW+projectileBoundaryMargin ||
			proj.Y < -projectileBoundaryMargin || proj.Y > mapH+projectileBoundaryMargin {
			proj.Active = false
			proj.Target = nil
			p.Count--
		}
	}
}

// Each 遍历所有存活弹射物并执行回调。
func (p *Pool) Each(fn func(proj *Projectile)) {
	for i := range p.projectiles {
		if p.projectiles[i].Active {
			fn(&p.projectiles[i])
		}
	}
}

// Release 释放弹射物（命中后回收）。
func (p *Pool) Release(proj *Projectile) {
	if proj.Active {
		proj.Active = false
		p.Count--
	}
}

// FirePenetrate 发射一颗直线穿透弹。
// 与普通弹的关键区别：
//   - Target=nil，按初始方向直线飞行，不追踪
//   - Penetrate=true，tick_combat.go 对路径上所有敌人做碰撞检测（用空间网格加速）
//   - MaxRange=发射距离，MaxLife=距离/速度*1.1（到终点后稍微多飞一点余量）
//   - 固定碰撞半径 4px（小于普通弹，因为穿透弹命中范围由 MaxRange 控制）
func (p *Pool) FirePenetrate(sx, sy, tx, ty, damage, speed float64, towerKey string) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = penetrateProjectileRadius
	proj.Active = true
	proj.MaxLife = dist / speed * penetrateLifeOvershoot // 飞到终点后稍微多一点余量
	proj.Life = proj.MaxLife
	proj.Target = nil // 直线飞行，不追踪
	proj.SourceTowerKey = towerKey
	proj.Penetrate = true
	proj.StartX = sx
	proj.StartY = sy
	proj.MaxRange = dist

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// ClearAll 清空所有弹射物（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.projectiles {
		p.projectiles[i].Active = false
		p.projectiles[i].Target = nil
	}
	p.Count = 0
	p.cursor = 0
}
