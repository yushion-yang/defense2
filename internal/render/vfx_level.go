// vfx_level.go — 实体数量驱动的 VFX 细节等级。
// 每帧 Draw 前调用 UpdateVFXLevel 根据当前实体负载自动选择细节等级。
// 正常游戏无降级，极端场景自动简化 VFX 保持 60fps。
package render

// VFXLevel VFX 渲染细节等级。
type VFXLevel int

const (
	VFXFull    VFXLevel = 0 // 所有 VFX 完整渲染
	VFXReduced VFXLevel = 1 // 跳过装饰性 VFX（高负载）
	VFXMinimal VFXLevel = 2 // 仅精灵+HP条（极端负载）
)

// CurrentVFXLevel 当前帧的 VFX 细节等级（单线程，无需加锁）。
var CurrentVFXLevel VFXLevel

// UpdateVFXLevel 根据实体数量计算本帧 VFX 细节等级。
// 塔权重 ×5 因为每塔 VFX 比单个敌人/弹道更昂贵。
func UpdateVFXLevel(enemyCount, projectileCount, towerCount int) {
	load := enemyCount + projectileCount + towerCount*5
	switch {
	case load > 800:
		CurrentVFXLevel = VFXMinimal
	case load > 400:
		CurrentVFXLevel = VFXReduced
	default:
		CurrentVFXLevel = VFXFull
	}
}
