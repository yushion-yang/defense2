// vfx_level.go — 帧时间驱动的 VFX 细节等级。
// 核心思路：不管什么原因导致卡顿（实体多、强度高、VFX 复杂），
// 只要上一帧慢了，本帧立刻降级。恢复也快（连续数帧快就升级）。
package render

// VFXLevel VFX 渲染细节等级。
type VFXLevel int

const (
	VFXFull    VFXLevel = 0 // 所有 VFX 完整渲染
	VFXReduced VFXLevel = 1 // 跳过装饰性 VFX
	VFXMinimal VFXLevel = 2 // 仅精灵+HP条
)

// CurrentVFXLevel 当前帧的 VFX 细节等级（单线程，无需加锁）。
var CurrentVFXLevel VFXLevel

const (
	vfxSlowMs = 13.0 // 帧时间超过此值视为慢帧
	vfxFastMs = 8.0  // 帧时间低于此值视为快帧
	vfxUpgradeFrames = 30 // 连续快帧数达到此值才升级（防闪烁）
)

var vfxFastCount int // 连续快帧计数

// UpdateVFXLevel 根据上一帧耗时和实体负载计算本帧 VFX 细节等级。
// lastFrameMs: 上一帧 Update+Draw 总耗时（毫秒）。
// 响应策略：降级立即（1 帧延迟），升级保守（连续 30 快帧）。
func UpdateVFXLevel(lastFrameMs float64, enemyCount, projectileCount, towerCount int) {
	// 实体负载（用于在帧时间正常时预防性降级）
	load := enemyCount + projectileCount + towerCount*5

	// ── 帧时间驱动（最高优先级）──
	if lastFrameMs > vfxSlowMs {
		// 慢帧：立即降一级
		vfxFastCount = 0
		if CurrentVFXLevel < VFXMinimal {
			CurrentVFXLevel++
		}
		return
	}

	if lastFrameMs < vfxFastMs {
		vfxFastCount++
	} else {
		vfxFastCount = 0
	}

	// 快帧足够多：尝试升一级
	if vfxFastCount >= vfxUpgradeFrames && CurrentVFXLevel > VFXFull {
		CurrentVFXLevel--
		vfxFastCount = 0
	}

	// ── 实体负载兜底（即使帧时间还行，负载极高时预防性降级）──
	if load > 800 && CurrentVFXLevel < VFXMinimal {
		CurrentVFXLevel = VFXMinimal
	} else if load > 400 && CurrentVFXLevel < VFXReduced {
		CurrentVFXLevel = VFXReduced
	}
}
