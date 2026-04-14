// draw_enemy.go — 敌人渲染模块。
//
// 本文件是渲染层中最复杂的绘制器，采用 3-pass 渲染策略将敌人按视觉层次分离：
//
//	Pass 1: 死亡动画（dying）— 先画，位于最底层，不会遮挡存活敌人
//	Pass 2: 存活敌人（active）— 精灵本体 + 行为 VFX + 状态特效 + 能力图标
//	Pass 3: HP 血条（deferred）— 收集后统一绘制，经过 Y 轴斥力避免重叠
//
// 设计决策：
// - 精灵优先 PNG，加载失败时回退到纯色圆形（theme.EnemyFallback*）
// - 帧动画通过 AnimLib 共享帧数据、Enemy 独立维护播放状态（AnimCur/AnimFrame/AnimTimer）
// - 路径缓存（spritePathCache）避免每帧 fmt.Sprintf 产生 GC 压力
// - HP 条采集-排序-斥力三步走，解决密集敌群血条重叠的可读性问题
// - 视口裁剪（IsInView）跳过屏幕外实体，大地图场景下显著降低 draw call
package render

import (
	"image/color"
	"log"
	"math"
	"slices"

	"defense2/internal/core/enemy"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// EnemyRenderer 管理敌人 PNG 精灵渲染，支持可选的帧动画。
// cache: 精灵图片缓存，按路径+尺寸去重
// animLibs: 按原型名共享的动画库（多个同原型敌人共用帧数据，各自维护播放进度）
type EnemyRenderer struct {
	cache    *sprite.Cache
	assetFS  AssetReader
	animLibs map[string]*anim.AnimLib // 每原型共享帧数据，惰性加载
}

// NewEnemyRenderer creates an enemy renderer.
func NewEnemyRenderer(assetFS AssetReader) *EnemyRenderer {
	return &EnemyRenderer{
		cache:    sprite.NewCache(),
		assetFS:  assetFS,
		animLibs: make(map[string]*anim.AnimLib),
	}
}

const enemySpriteSize = 32 // 敌人精灵的逻辑显示尺寸（像素），与资源 PNG 无关

// hpBarEntry 是 HP 血条的延迟渲染数据（Pass 3 使用）。
// 在 Pass 2 遍历存活敌人时收集，之后按 Y 坐标排序并施加斥力以避免重叠。
// 使用固定数组 [256] 在栈上分配，避免每帧堆分配。
type hpBarEntry struct {
	cx, cy     float32 // enemy center
	barW, barH float32
	barOffY    float32 // base Y offset above enemy center
	adjustY    float32 // additional Y offset from repulsion (negative = higher)
	hp, maxHP  float64
	displayHP  float64
	boss       bool
	// status dots
	slowed, stunned, rooted, bleeding, burning, poisoned, weakened bool
}

// spritePathCache 缓存精灵路径字符串，避免每帧 fmt.Sprintf 分配。
// 敌人种类有限（~75 种精灵变体），此 map 增长有界。
var spritePathCache = map[string]string{}

func cachedEnemySpritePath(dir string) string {
	if p, ok := spritePathCache[dir]; ok {
		return p
	}
	p := "assets/enemies/sprites/" + dir + "/" + dir + ".png"
	spritePathCache[dir] = p
	return p
}

func cachedShieldPath(name string) string {
	key := "shield:" + name
	if p, ok := spritePathCache[key]; ok {
		return p
	}
	p := "assets/enemies/shields/" + name + ".png"
	spritePathCache[key] = p
	return p
}

// DrawEnemies 渲染所有存活敌人，采用 3-pass 策略：
//
// 流程概览：
//
//	Pass 1: 遍历池中所有敌人，仅绘制 dying 状态的（缩小+淡出动画），位于最底层
//	Pass 2: 遍历池中所有敌人，绘制非 dying 的存活敌人，包括：
//	        - Boss 脉冲环 / Runner 速度环 / Root 地面效果 / Buffer 光环（精灵之下）
//	        - 精灵本体（含行走摆动、Boss 呼吸缩放、出生动画、隐身/相位透明度）
//	        - 行为 VFX（减伤盾/狂暴/再生，被沉默时隐藏）
//	        - 状态特效（减速冰霜/燃烧/中毒叠加、眩晕星星、受击闪白）
//	        - 能力常驻视觉（免疫脚环/盾牌图标/净化微光）
//	        - 飘字（伤害数字、免疫提示等）
//	        - 收集 HP 条数据到 hpBars 数组
//	Pass 3: 对收集到的 HP 条施加 Y 轴斥力后统一绘制（背景+橙色伤害轨迹+彩色填充+状态点）
func (er *EnemyRenderer) DrawEnemies(screen *ebiten.Image, pool *enemy.Pool, animTime float64) {
	// Pass 1: dying 敌人（先画，位于存活敌人之下）
	pool.Each(func(e *enemy.Enemy) {
		if !e.IsDying() {
			return
		}
		if e.DyingDuration <= 0 {
			return
		}
		// Viewport culling: skip dying enemies outside camera view
		if !IsInView(e.X, e.Y) {
			return
		}
		cx := float32(e.X)
		cy := float32(e.Y)
		scale, alphaF, offsetY := vfx.DyingAnimParams(e.DyingTimer, e.DyingDuration)
		alpha := float32(alphaF)

		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			displaySize := float64(enemySpriteSize) * scale
			if displaySize < 0.5 {
				return // too small to see
			}
			w := float64(img.Bounds().Dx())
			h := float64(img.Bounds().Dy())
			s := displaySize / w * draw.Scale
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(-w/2, -h/2)
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(float64(cx)*draw.Scale, (float64(cy)+offsetY)*draw.Scale)
			op.ColorScale.ScaleAlpha(alpha)
			screen.DrawImage(img, &op)
		}
	})

	// HP 条数据收集缓冲区：栈上分配 256 容量，避免堆分配。
	// 256 远超单屏可见敌人数（通常 30-50），溢出时 append 会自动堆逃逸。
	var hpBarsArr [256]hpBarEntry
	hpBars := hpBarsArr[:0]

	// Pass 2: 存活（非 dying）敌人
	pool.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}

		// Viewport culling: skip rendering for off-screen enemies
		if !IsInView(e.X, e.Y) {
			return
		}

		cx := float32(e.X)
		cy := float32(e.Y)

		r := float32(e.Radius)

		// --- Boss pulsing rings ---
		if e.Boss {
			vfx.DrawBossPulse(screen, cx, cy, r, animTime)
		}

		// --- Runner pulsing ring ---
		if e.Archetype == "runner" {
			vfx.DrawRunnerRing(screen, cx, cy, r, animTime)
		}

		// --- Root ground effect (drawn UNDER enemy body) ---
		if e.IsRooted() {
			vfx.DrawRootGround(screen, cx, cy, r)
		}

		// --- Buffer aura ring (drawn UNDER body, hidden when silenced) ---
		if e.Behavior == "buffer" && e.HasBufferAura() && !e.AbilitySilenced {
			if ba, ok := e.Buffs.Get("bufferAura"); ok {
				vfx.DrawBufferAura(screen, cx, cy, ba.Value2, animTime)
			}
		}

		// (旧 healer aura ring 已移到能力 VFX 系统)

		// --- Spawn animation modifiers ---
		spawnScale, spawnAlpha := 1.0, 1.0
		if e.IsSpawning() && e.SpawnDuration > 0 {
			spawnScale, spawnAlpha = vfx.SpawnAnimParams(e.SpawnTimer, e.SpawnDuration)
		}

		// --- 敌人本体渲染（帧动画优先，静态精灵兜底） ---
		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			// 行走摆动：X 位置 + 时间混合作为相位，让每个敌人有独立的摆动节奏。
			// wobbleY: ±1.5px 上下浮动；wobbleRot: ±0.05rad (~3°) 左右摇摆。
			// 眩晕/定身时禁用摆动（视觉上表示无法移动）。
			wobblePhase := e.X*0.05 + animTime*4
			wobbleY := math.Sin(wobblePhase) * 1.5
			wobbleRot := math.Sin(wobblePhase) * 0.05
			if e.IsStunned() || e.IsRooted() {
				wobbleY = 0
				wobbleRot = 0
			}
			// Boss/Elite 呼吸缩放：体型周期性脉冲
			displaySize := float64(enemySpriteSize)
			if e.Boss {
				displaySize *= 1.0 + 0.04*math.Sin(animTime*1.8)
			}
			// Apply spawn scale
			displaySize *= spawnScale

			// 计算 alpha（隐身/相位）
			bodyAlpha := 1.0
			if e.IsStealthed() {
				bodyAlpha = 0.15
			} else if e.PhaseActive && !e.AbilitySilenced {
				bodyAlpha = 0.35
			}
			// Apply spawn alpha
			bodyAlpha *= spawnAlpha

			if bodyAlpha < 1.0 || spawnScale != 1.0 {
				logicalScale := displaySize / float64(img.Bounds().Dx())
				draw.SpriteScaledRotatedAlpha(screen, img, float64(cx), float64(cy)+wobbleY,
					logicalScale, wobbleRot, bodyAlpha)
			} else {
				draw.SpriteRotated(screen, img, float64(cx), float64(cy), displaySize, wobbleRot, wobbleY)
			}
		} else {
			bodyColor := theme.EnemyFallback
			if e.Boss {
				bodyColor = theme.EnemyFallbackBoss
			}
			if e.IsStealthed() {
				bodyColor.A = 38
			} else if e.PhaseActive && !e.AbilitySilenced {
				bodyColor.A = 90
			}
			// Apply spawn alpha to fallback circle
			bodyColor.A = uint8(float64(bodyColor.A) * spawnAlpha)
			draw.FilledCircle(screen, cx, cy, r*float32(spawnScale), bodyColor)
		}

		// --- Buff behavior VFX (drawn over body, hidden when silenced) ---
		if !e.AbilitySilenced {
			if e.GetDamageReduce() > 0 {
				vfx.DrawDamageReduceShield(screen, cx, cy, float32(e.Radius), animTime)
			}
			if e.HasBerserk() && e.BerserkTriggered {
				vfx.DrawBerserkFlare(screen, cx, cy, float32(e.Radius), animTime)
			}
			if e.HasRegen() {
				vfx.DrawRegenAura(screen, cx, cy, float32(e.Radius), animTime)
			}
		}

		// --- Status effect body overlays (subtle, sprite-sized) ---
		spriteR := float32(enemySpriteSize) / 2
		if e.IsSlowed() {
			vfx.DrawSlowOverlay(screen, cx, cy, spriteR, animTime)
		}
		if e.IsBurning() {
			vfx.DrawBurnOverlay(screen, cx, cy, spriteR, animTime)
		}
		if e.Buffs != nil && e.Buffs.Has("poison") {
			vfx.DrawPoisonOverlay(screen, cx, cy, spriteR, animTime)
		}

		// --- Stun rotating stars ---
		if e.IsStunned() {
			vfx.DrawStunStars(screen, cx, cy, r, animTime)
		}

		// --- Hit flash overlay ---
		if e.HitFlash > 0 && !e.IsDying() {
			vfx.DrawHitFlash(screen, cx, cy, spriteR, e.HitFlash)
		}

		// (tank overlay removed — was debug placeholder)

		// Stealthed enemies: skip HP bar and status dots (nearly invisible)
		if e.IsStealthed() {
			return
		}

		// 收集 HP 条数据用于 Pass 3 延迟绘制（带斥力算法避免重叠）。
		// 优化：满血且无状态效果的敌人不显示血条（减少视觉噪声）。
		// Boss 始终显示（玩家需要实时掌握 Boss 血量）。
		hasStatus := e.IsSlowed() || e.IsStunned() || e.IsRooted() || e.IsBleeding() || e.IsBurning() || e.IsPoisoned() || e.IsWeakened()
		if e.HP < e.MaxHP || e.Boss || hasStatus {
			var barW, barH, barOffY float32
			if e.Boss {
				barW = theme.EnemyBossHPBarW
				barH = theme.EnemyBossHPBarH
				barOffY = theme.EnemyBossHPOffsetY
			} else {
				barW = theme.EnemyHPBarW
				barH = theme.EnemyHPBarH
				barOffY = theme.EnemyHPBarOffsetY
			}
			hpBars = append(hpBars, hpBarEntry{
				cx: cx, cy: cy,
				barW: barW, barH: barH, barOffY: barOffY,
				hp: e.HP, maxHP: e.MaxHP, displayHP: e.DisplayHP,
				boss:     e.Boss,
				slowed:   e.IsSlowed(),
				stunned:  e.IsStunned(),
				rooted:   e.IsRooted(),
				bleeding: e.IsBleeding(),
				burning:  e.IsBurning(),
				poisoned: e.IsPoisoned(),
				weakened: e.IsWeakened(),
			})
		}

		// --- 能力常驻视觉（被沉默时全部隐藏）---
		if !e.AbilitySilenced {
			// 免疫脚环（只显示天生能力，净化临时免疫用白色微光）
			footR := float32(e.Radius) + 2
			if hasAbility(e, enemy.AbilCCImmune) {
				vfx.DrawImmunityRing(screen, cx, cy, footR, theme.EnemyImmuneCC, animTime)
			} else if hasAbility(e, enemy.AbilSlowImmune) {
				vfx.DrawImmunityRing(screen, cx, cy, footR, theme.EnemyImmuneSlow, animTime)
			}

			// 盾牌叠加（能力对应颜色盾牌）
			shieldOffset := float32(e.Radius) * 0.6
			if e.ProjectileBlockChance > 0 {
				if img := er.loadShield("shield-white"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}
			if e.ArmorFlat > 0 {
				if img := er.loadShield("shield-blue"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}
			if e.DamageCap > 0 || e.DamageCapPercent > 0 {
				if img := er.loadShield("shield-orange"); img != nil {
					draw.Sprite(screen, img, float64(cx+shieldOffset), float64(cy), 14)
				}
			}

			// 净化免疫期白色微光
			if e.PurgeInterval > 0 && e.HasControlImmunity() {
				vfx.DrawPurgeGlow(screen, cx, cy, float32(e.Radius), animTime)
			}
		}

		// --- 飘字渲染 ---
		if e.FloatText != "" && e.FloatTextTimer > 0 {
			fm := GlobalFont()
			if fm != nil {
				progress := 1 - e.FloatTextTimer/0.6
				floatY := float64(cy) - float64(e.Radius) - 10 - progress*12
				alpha := uint8(255 * (1 - progress))
				fm.DrawCenteredText(screen, e.FloatText, float64(cx), floatY, 10,
					color.RGBA{R: e.FloatTextR, G: e.FloatTextG, B: e.FloatTextB, A: alpha})
			}
		}
	})

	// Pass 3: 统一绘制所有 HP 条，先施加 Y 轴斥力再渲染。
	if len(hpBars) > 0 {
		repulseHPBars(hpBars)
		drawHPBars(screen, hpBars, animTime)
	}
}

// repulseHPBars 对 HP 条施加简易 Y 轴斥力，使重叠的血条上下分散。
// 算法：先按血条屏幕 Y 排序，再线性扫描相邻对，发现 Y 重叠时将两者各推移一半。
// 仅在 X 方向有重叠时才计算斥力（远距离的血条互不干扰），复杂度 O(n log n + n)。
func repulseHPBars(bars []hpBarEntry) {
	if len(bars) < 2 {
		return
	}
	// Sort by the bar's screen Y position (enemy cy - barOffY).
	slices.SortFunc(bars, func(a, b hpBarEntry) int {
		ay := a.cy - a.barOffY
		by := b.cy - b.barOffY
		if ay < by {
			return -1
		}
		if ay > by {
			return 1
		}
		return 0
	})

	const minGap float32 = 2 // minimum vertical gap between bars

	for i := 1; i < len(bars); i++ {
		prev := &bars[i-1]
		cur := &bars[i]

		// Check horizontal overlap first — bars far apart in X don't need repulsion.
		maxW := prev.barW
		if cur.barW > maxW {
			maxW = cur.barW
		}
		dx := cur.cx - prev.cx
		if dx < 0 {
			dx = -dx
		}
		if dx > maxW {
			continue // no horizontal overlap
		}

		prevBottom := prev.cy - prev.barOffY + prev.adjustY + prev.barH
		curTop := cur.cy - cur.barOffY + cur.adjustY

		overlap := prevBottom + minGap - curTop
		if overlap > 0 {
			// Push current bar down, previous bar up (split evenly).
			half := overlap / 2
			prev.adjustY -= half
			cur.adjustY += half
		}
	}
}

// drawHPBars 渲染所有收集到的 HP 条。
// 每个血条由四层组成：
//  1. 黑色背景（1px 伪边框）
//  2. 橙色伤害轨迹（displayHP > hp 时可见，平滑追赶实际 HP 产生"掉血"视觉）
//  3. 彩色 HP 填充（>60% 绿 / >30% 黄 / ≤30% 红）
//  4. Boss 专用分段线（每 20% 一道分割线，便于估读血量）
//
// 血条上方绘制状态效果圆点（减速/眩晕/定身/流血/燃烧/中毒/削弱）。
func drawHPBars(screen *ebiten.Image, bars []hpBarEntry, animTime float64) {
	for i := range bars {
		b := &bars[i]
		barX := b.cx - b.barW/2
		barY := b.cy - b.barOffY + b.adjustY

		// Only draw HP bar if not full HP (status-only entries skip the bar).
		if b.hp < b.maxHP {
			// Background (1px pseudo-border)
			draw.FilledRect(screen, barX-1, barY-1, b.barW+2, b.barH+2,
				theme.EnemyHPBarBg, true)

			// Damage trail (orange)
			if b.displayHP > b.hp && b.displayHP > 0 {
				trailRatio := float32(b.displayHP / b.maxHP)
				if trailRatio > 1 {
					trailRatio = 1
				}
				draw.FilledRect(screen, barX, barY, b.barW*trailRatio, b.barH,
					theme.EnemyHPBarTrail, true)
			}

			// HP fill
			ratio := float32(b.hp / b.maxHP)
			if ratio < 0 {
				ratio = 0
			} else if ratio > 1 {
				ratio = 1
			}
			fillW := b.barW * ratio

			var fillClr color.RGBA
			switch {
			case ratio > 0.6:
				fillClr = theme.EnemyHPFillHigh
			case ratio > 0.3:
				fillClr = theme.EnemyHPFillMid
			default:
				fillClr = theme.EnemyHPFillLow
			}
			draw.FilledRect(screen, barX, barY, fillW, b.barH, fillClr, true)

			// Boss segment dividers
			if b.boss {
				for s := 1; s < 5; s++ {
					divX := barX + b.barW*float32(s)*0.2
					draw.FilledRect(screen, divX, barY, 1, b.barH, theme.EnemyHPSegDiv, true)
				}
			}
		}

		// Status effect dots (above the bar)
		dotY := barY - 4
		var dotsArr [7]vfx.StatusDot
		dots := dotsArr[:0]
		if b.slowed {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotSlowed})
		}
		if b.stunned {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotStunned})
		}
		if b.rooted {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotRooted})
		}
		if b.bleeding {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotBleeding})
		}
		if b.burning {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotBurning})
		}
		if b.poisoned {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotPoison})
		}
		if b.weakened {
			dots = append(dots, vfx.StatusDot{Color: theme.EnemyDotWeaken})
		}
		if len(dots) > 0 {
			vfx.DrawStatusDots(screen, b.cx, dotY, dots, animTime)
		}
	}
}

// hasAbility 检查敌人是否装配了指定能力。
func hasAbility(e *enemy.Enemy, abilityType string) bool {
	for _, id := range e.AbilityIDs {
		if id == abilityType {
			return true
		}
	}
	return false
}

// loadShield 加载盾牌 PNG（缓存）。
func (er *EnemyRenderer) loadShield(name string) *ebiten.Image {
	path := cachedShieldPath(name)
	if cached := er.cache.Get(path, 12, 16); cached != nil {
		return cached
	}
	if er.assetFS == nil {
		return nil
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, 12, 16)
	if err != nil {
		log.Printf("[render] sprite parse failed: %v", err)
	}
	return img
}

// loadEnemyImage 加载敌人 PNG 精灵。
// 命名约定：assets/enemies/sprites/{spriteDir}/{spriteDir}.png
// SpriteDir 字段允许同一原型使用不同精灵变体（如 tank 原型的 tank_elite 变体）。
func (er *EnemyRenderer) loadEnemyImage(e *enemy.Enemy) *ebiten.Image {
	spriteDir := e.SpriteDir
	if spriteDir == "" {
		spriteDir = e.Archetype
	}
	if er.assetFS == nil || spriteDir == "" {
		return nil
	}
	path := cachedEnemySpritePath(spriteDir)
	cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize)
	if cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// GetSprite 按原型名加载敌人静态精灵（供造怪菜单等外部使用）。
func (er *EnemyRenderer) GetSprite(archetype string) *ebiten.Image {
	if er.assetFS == nil || archetype == "" {
		return nil
	}
	path := cachedEnemySpritePath(archetype)
	if cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize); cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	if err != nil {
		log.Printf("[render] sprite parse failed: %v", err)
	}
	return img
}

// getEnemyFrame 返回敌人当前动画帧，无帧动画时回退到静态精灵。
// 动画状态分离设计：
//   - AnimLib: 按原型共享（同种敌人共享帧图片数据，节省内存）
//   - AnimCur/AnimFrame/AnimTimer/AnimDone: 每个 Enemy 独立维护（各自播放进度独立）
//
// 动画选择逻辑：受击时播放 "hit" 动画（如果存在），否则默认 "walk"。
func (er *EnemyRenderer) getEnemyFrame(e *enemy.Enemy, dt float64) *ebiten.Image {
	if e.Archetype == "" {
		return nil
	}

	sprKey := e.SpriteDir
	if sprKey == "" {
		sprKey = e.Archetype
	}
	lib, ok := er.animLibs[sprKey]
	if !ok {
		lib = anim.LoadEnemyAnimLib(er.assetFS, sprKey)
		er.animLibs[sprKey] = lib
	}

	// Determine target animation
	target := "walk"
	if e.HitFlash > 0 && lib.HasAnim("hit") {
		target = "hit"
	}

	// Switch animation if needed (reset on change or when finished non-loop replays)
	if e.AnimCur != target || (e.AnimDone && target != e.AnimCur) {
		e.AnimCur = target
		e.AnimFrame = 0
		e.AnimTimer = 0
		e.AnimDone = false
	}

	// Advance per-enemy timer
	a, exists := lib.Anims[e.AnimCur]
	if exists && !e.AnimDone && len(a.Frames) > 0 {
		e.AnimTimer += dt
		frameDur := 1.0 / a.FPS
		if e.AnimTimer >= frameDur {
			e.AnimTimer -= frameDur
			e.AnimFrame++
			if e.AnimFrame >= len(a.Frames) {
				if a.Loop {
					e.AnimFrame = 0
				} else {
					e.AnimFrame = len(a.Frames) - 1
					e.AnimDone = true
				}
			}
		}
	}

	img := lib.Frame(e.AnimCur, e.AnimFrame)
	if img != nil {
		return img
	}
	return er.loadEnemyImage(e)
}
