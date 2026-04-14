// draw_tower.go — 塔渲染模块。
//
// 负责将 core 层的 tower.Tower 转为屏幕绘制，包括：
// - 精灵本体（64px PNG + 帧动画），加载失败回退为几何图形（圆形底座+矩形炮管）
// - 建造/出售动画（缩放+淡入淡出）
// - 选中塔的视觉反馈（选中环 + 射程指示圈）
// - 力量溢出光晕（Strength overflow 强度外溢时脚下发光）
// - 攻击方式特有 VFX（spin_aoe 旋转刀刃弧）
// - Buff 指示点 + 名称标签
// - 光环/区域能力的视觉效果（通过 auraRegistry 注册制分发）
//
// 性能关键设计：
//   - 所有塔 VFX 的线段绘制被 BeginLineBatch/FlushLineBatch 包裹，
//     将 Diamond/Arc/DashedCircle/ThickLine 合并为一次 DrawTriangles 调用
//   - vfxRadius() 钳制极端射程（>400px），防止 VFX 视觉爆炸
//   - 视口裁剪跳过屏幕外塔（选中塔例外，始终绘制以保持 UI 一致性）
package render

import (
	"fmt"
	"log"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/tower"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// TowerRenderer 管理塔的 PNG 精灵渲染与帧动画。
// animators 按 "instanceKey:spriteKey" 索引，确保同一塔选择能力后
// SpriteKey 变更（如 basic→sentinel）时重新加载对应动画资源。
type TowerRenderer struct {
	cache     *sprite.Cache
	assetFS   AssetReader
	animators map[string]*anim.Animator // 每塔实例惰性加载的动画器
}

// AssetReader reads embedded asset files.
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// NewTowerRenderer creates a tower renderer.
func NewTowerRenderer(assetFS AssetReader) *TowerRenderer {
	return &TowerRenderer{
		cache:     sprite.NewCache(),
		assetFS:   assetFS,
		animators: make(map[string]*anim.Animator),
	}
}

const towerSpriteSize = 64 // 塔精灵逻辑显示尺寸（像素），与 theme.TowerBaseSize 匹配

// vfxRadius 将游戏 Range 转为 VFX 绘制半径。
// DashedCircle 已内置自适应（大圆自动放大 dash 间距保证段数不爆炸），
// 因此此处不再钳制上限，VFX 圈与实际射程一致。
func vfxRadius(r float64) float32 {
	return float32(r)
}

// towerAnimScaleAlpha 计算建造/出售动画的缩放系数和透明度。
// 返回 (scaleMul, alpha)，无动画时均为 1.0。
// 建造动画：从小变大+淡入；出售动画：从大变小+淡出。
func towerAnimScaleAlpha(t *tower.Tower) (float64, float64) {
	if t.Selling && t.SellAnim > 0 {
		return vfx.SellAnimParams(t.SellAnim)
	}
	return vfx.BuildAnimParams(t.BuildAnim)
}

// DrawTowers 渲染所有已放置的塔。
//
// 流程概览：
//  1. BeginLineBatch 开启线段批量收集模式
//  2. 遍历塔池，逐塔绘制：建造涟漪 → 选中环/射程 → 脚底光晕/光环 → 精灵本体
//     → spin_aoe 旋转弧 → buff 点 → 名称标签
//  3. FlushLineBatch 将所有 VFX 线段合并为一次 DrawTriangles
//
// 线段批量化是关键优化：一局游戏 6 塔，每塔可能有 DashedCircle（64段）+ Diamond
// + 光环圈等，不批量化则每帧数百次 draw call。
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool, selectedTower *tower.Tower, animTime float64) {
	// 开启线段批量模式：后续所有 draw.Diamond/Arc/DashedCircle/ThickLine
	// 调用被拦截并收集到顶点缓冲，FlushLineBatch 时一次性提交 GPU
	draw.BeginLineBatch(screen)
	defer draw.FlushLineBatch()

	pool.Each(func(t *tower.Tower) {
		// Viewport culling: skip off-screen towers (selected tower always rendered)
		selected := selectedTower != nil && t == selectedTower
		if !selected && !IsInView(t.X, t.Y) {
			return
		}

		cx := float32(t.X)
		cy := float32(t.Y)

		// Compute animation scale and alpha
		animScale, animAlpha := towerAnimScaleAlpha(t)

		// --- Build ripple effect ---
		if t.BuildAnim > 0 {
			vfx.DrawBuildRipple(screen, cx, cy, 1.0-t.BuildAnim/0.3)
		}

		// --- Selection ring & range indicator (selected tower only, skip during sell) ---
		if selected && !t.Selling {
			vfx.DrawSelectionRing(screen, cx, cy,
				theme.TowerSelectionRingR, theme.TowerSelectionWidth, theme.TowerSelectionRing,
				vfxRadius(t.Range), theme.TowerRangeStrokeWidth, theme.TowerRangeStroke)
		}

		// --- Under-body VFX (drawn BEFORE sprite so they don't obscure it) ---
		if !t.Selling && t.BuildAnim <= 0 && t.Strength != nil {
			vfx.DrawStrengthGlow(screen, cx, cy, t.Strength.Overflow(), animTime)
		}
		if !t.Selling && t.BuildAnim <= 0 {
			drawTowerAuras(screen, t, cx, cy, animTime)
		}

		// --- 塔本体（帧动画或静态精灵，朝向目标旋转） ---
		// spin_aoe（旋风攻击）是 360° 旋转攻击，不跟踪目标朝向
		// 其他攻击方式的 Angle 是炮管指向目标的弧度，+π/2 是因为精灵默认朝上
		rotation := t.Angle + math.Pi/2
		if t.AttackStyleID == tower.StyleSpinAoE {
			rotation = 0
		}

		img := tr.getTowerFrame(t, 1.0/60.0)
		if img != nil {
			logicalScale := float64(towerSpriteSize) / float64(img.Bounds().Dx())
			// 射击缩放脉冲（skip during build/sell anim）
			if t.FireAnim > 0 && t.BuildAnim <= 0 && !t.Selling {
				logicalScale *= vfx.FirePulseScale(t.FireAnim)
			}
			// Apply build/sell animation scale
			logicalScale *= animScale
			if animAlpha < 1.0 {
				draw.SpriteScaledRotatedAlpha(screen, img, float64(cx), float64(cy), logicalScale, rotation, animAlpha)
			} else {
				draw.SpriteScaledRotated(screen, img, float64(cx), float64(cy), logicalScale, rotation)
			}
		} else {
			// Fallback: circle body + barrel rectangle
			bodyClr := theme.TowerFallbackDef
			if selected {
				bodyClr = theme.TowerFallbackSel
			}
			// Apply alpha to fallback colors
			if animAlpha < 1.0 {
				bodyClr.A = uint8(float64(bodyClr.A) * animAlpha)
			}
			draw.FilledCircle(screen, cx, cy, float32(float64(theme.TowerFallbackRadius)*animScale), bodyClr)
			barrelClr := theme.TowerBarrel
			if animAlpha < 1.0 {
				barrelClr.A = uint8(float64(barrelClr.A) * animAlpha)
			}
			barrelLen := 14.0 * animScale
			bx2 := float64(cx) + math.Cos(t.Angle)*barrelLen
			by2 := float64(cy) + math.Sin(t.Angle)*barrelLen
			draw.ThickLine(screen, float32(cx), float32(cy), float32(bx2), float32(by2), float32(8*animScale), barrelClr)
		}

		// 射击反馈已由 shoot pulse（精灵放大 15%）+ muzzle flash 粒子提供，
		// 不再叠加白色圆——高攻速塔会导致持续白圈。

		// --- Spin AoE visual: rotating blade arcs ---
		if !t.Selling && t.BuildAnim <= 0 && t.AttackStyleID == tower.StyleSpinAoE && t.SpinActive > 0 {
			vfx.DrawSpinBlades(screen, cx, cy, vfxRadius(t.Range), t.SpinAngle, t.SpinActive/0.3)
		}

		// --- Buff indicator dots ---
		if !t.Selling && t.Buffs.Count() > 0 {
			vfx.DrawBuffDots(screen, cx, cy, t.Buffs.Count())
		}

		// --- Name label (skip during sell animation) ---
		if !t.Selling {
			if fm := GlobalFont(); fm != nil {
				fm.DrawCenteredText(screen, t.Label,
					float64(cx), float64(cy)+theme.TowerNameLabelY,
					theme.FontTowerName, theme.TowerNameLabel)
			}
		}
	})
}

// loadTowerImage 加载塔的 PNG 精灵。
// 命名约定：assets/towers/{key}/tower-{key}.png
// SpriteKey 优先于 Key：选择攻击能力后 SpriteKey 会变（如 basic→sentinel），
// 此时加载新的精灵资源。
func (tr *TowerRenderer) loadTowerImage(t *tower.Tower) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	key := t.SpriteKey
	if key == "" {
		key = t.Key
	}
	path := fmt.Sprintf("assets/towers/%s/tower-%s.png", key, key)
	cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize)
	if cached != nil {
		return cached
	}
	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// GetSprite 按 key 获取塔精灵，用于图标/缩略图（如建塔菜单）。
// 优先取 idle-0 帧（与游戏中渲染一致），回退到静态 PNG。
func (tr *TowerRenderer) GetSprite(key string) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	// Try idle-0 first (matches map rendering)
	for _, path := range []string{
		fmt.Sprintf("assets/towers/%s/tower-%s-idle-0.png", key, key),
		fmt.Sprintf("assets/towers/%s/tower-%s.png", key, key),
	} {
		if img := tr.loadPNG(path); img != nil {
			return img
		}
	}
	return nil
}

func (tr *TowerRenderer) loadPNG(path string) *ebiten.Image {
	if cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize); cached != nil {
		return cached
	}
	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	if err != nil {
		log.Printf("[render] sprite parse failed: %v", err)
	}
	return img
}

// getTowerFrame 返回塔当前的动画帧，无帧动画时回退到静态精灵。
// 动画器按 "instanceKey:spriteKey" 惰性加载，SpriteKey 变更时自动加载新动画。
func (tr *TowerRenderer) getTowerFrame(t *tower.Tower, dt float64) *ebiten.Image {
	// 惰性加载动画器（按 SpriteKey 索引以支持能力选择后视觉切换）
	sprKey := t.SpriteKey
	if sprKey == "" {
		sprKey = t.Key
	}
	// animKey 包含 sprKey 以确保 SpriteKey 变更（如选择能力后）重新加载动画
	animKey := t.InstanceKey + ":" + sprKey
	if t.InstanceKey == "" {
		animKey = sprKey
	}
	a, ok := tr.animators[animKey]
	if !ok {
		a = anim.LoadTowerAnimator(tr.assetFS, sprKey)
		tr.animators[animKey] = a
	}

	// Choose animation state
	if t.FireAnim > 0 && a.HasAnim("attack") {
		a.Play("attack")
	} else if a.HasAnim("idle") {
		a.Play("idle")
	}
	a.Update(dt)

	img := a.CurrentImage()
	if img != nil {
		return img
	}
	// Fallback to static sprite cache
	return tr.loadTowerImage(t)
}

// DrawTowerRangePreview 绘制建塔放置预览（射程圈 + 合法性颜色指示）。
// valid=true 时绿色圈表示可放置，false 时红色圈表示无效位置。
func DrawTowerRangePreview(screen *ebiten.Image, cx, cy float32, r float64, valid bool) {
	fr := float32(r)

	strokeClr := theme.TowerRangeInvalid
	if valid {
		strokeClr = theme.TowerRangeValid
	}

	// Range circle (outline only)
	draw.CircleOutline(screen, cx, cy, fr, 1.5, strokeClr)
}

// AuraDrawFunc 单个 buff/zone 能力的自定义渲染函数。
// 参数: screen, 塔中心(cx,cy), 效果半径, 动画时间。
// 每种 buff 可实现完全独立的视觉效果（圈、脉冲、粒子等）。
type AuraDrawFunc func(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64)

// auraEntry 注册一个 buff 能力的视觉效果。
type auraEntry struct {
	DrawFunc      AuraDrawFunc // 自定义渲染函数
	UseTowerRange bool         // true: 用塔射程, false: 用 ability param
}

// auraRegistry 将 buff/zone 能力名映射到对应的 VFX 渲染函数。
// 注册制设计：新增光环视觉只需两步——
//  1. 在 vfx 包中实现 DrawXxx(screen, cx, cy, radius, animTime) 函数
//  2. 在此 map 中注册能力名 → DrawFunc
//
// 两种半径来源：
//   - buff 类光环（UseTowerRange=false）：半径从 ability 配置的 Param 读取
//   - zone 类效果（UseTowerRange=true）：半径等于塔的射程（已经过 vfxRadius 钳制）
var auraRegistry = map[string]auraEntry{
	// 增益光环（buff 类，param=半径）
	"damageUpAura":    {DrawFunc: vfx.DrawDamageAura},
	"attackSpeedAura": {DrawFunc: vfx.DrawSpeedAuraRing},
	"rangeAura":       {DrawFunc: vfx.DrawRangeAura},
	"critAura":        {DrawFunc: vfx.DrawCritAura},
	"soloBoost":       {DrawFunc: vfx.DrawSoloAura},
	// 区域效果（zone 类，用塔射程）
	"poisonZone":  {DrawFunc: vfx.DrawPoisonZone, UseTowerRange: true},
	"silenceZone": {DrawFunc: vfx.DrawSilenceZone, UseTowerRange: true},
	"curseZone":   {DrawFunc: vfx.DrawCurseZone, UseTowerRange: true},
	"weakenZone":  {DrawFunc: vfx.DrawWeakenZone, UseTowerRange: true},
}

// drawTowerAuras 渲染塔上所有 buff/zone 能力的视觉效果。
// 遍历塔的能力列表，查找 auraRegistry 中有注册视觉的能力，
// 根据 UseTowerRange 决定半径来源后委托给对应的 VFX 函数。
func drawTowerAuras(screen *ebiten.Image, t *tower.Tower, cx, cy float32, animTime float64) {
	table := config.GlobalAbilityTable()
	if table == nil {
		return
	}
	for _, abName := range t.Abilities {
		entry, ok := auraRegistry[abName]
		if !ok {
			continue
		}
		var radius float64
		if entry.UseTowerRange {
			radius = float64(vfxRadius(t.Range))
		} else if def, exists := table[abName]; exists && def.Param > 0 {
			radius = def.Param
		}
		if radius <= 0 {
			continue
		}
		entry.DrawFunc(screen, cx, cy, radius, animTime)
	}
}
