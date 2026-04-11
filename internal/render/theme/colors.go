package theme

import "image/color"

// hex converts a 0xRRGGBB integer to an opaque color.RGBA.
func hex(rgb uint32) color.RGBA {
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8(rgb >> 8),
		B: uint8(rgb),
		A: 0xff,
	}
}

// rgba builds a color.RGBA from individual 0-255 components.
func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// ---------------------------------------------------------------------------
// Canvas / Panel
// ---------------------------------------------------------------------------

var (
	CanvasBg    = hex(0x0f172a)
	PanelBg     = rgba(15, 23, 42, 224)   // 0.88 * 255 ≈ 224
	PanelBorder = rgba(148, 163, 184, 51) // 0.2  * 255 ≈ 51
)

// ---------------------------------------------------------------------------
// Text
// ---------------------------------------------------------------------------

var (
	TextTitle  = hex(0xf1f5f9) // 更亮的标题白
	TextBody   = hex(0xe2e8f0) // 正文提亮
	TextMuted  = hex(0xb0bec5) // 辅助文字提亮
	TextLocked = hex(0x4b5563)
)

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

var (
	StatusStrUp   = hex(0x4ade80)
	StatusStrDown = hex(0xfca5a5) // 红300，暗背景更醒目
	StatusStrNorm = hex(0x94a3b8)
	StatusExcl    = hex(0x38bdf8)
	StatusWarden  = hex(0xfbbf24)
	StatusGrowth  = hex(0x4ade80)
	StatusSkill   = hex(0xa78bfa) // 紫400，技能/事件高阶标签
)

// ---------------------------------------------------------------------------
// Buttons (flat)
// ---------------------------------------------------------------------------

var (
	BtnPrimary   = hex(0x3b82f6)
	BtnDanger    = hex(0xef4444)
	BtnSecondary = hex(0x475569)
	BtnMuted     = hex(0x334155)
)

// ---------------------------------------------------------------------------
// Button Tones (with alpha)
// ---------------------------------------------------------------------------

var (
	TonePrimary   = rgba(34, 197, 94, 245)  // 0.96 * 255 ≈ 245
	ToneAccent    = rgba(59, 130, 246, 235) // 0.92 * 255 ≈ 235
	ToneSecondary = rgba(15, 23, 42, 217)   // 0.85 * 255 ≈ 217
	ToneDisabled  = rgba(30, 41, 59, 97)    // 0.38 * 255 ≈ 97
)


// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

var (
	ResHearts = hex(0xfca5a5)
	ResGold   = hex(0xfbbf24)
	ResWaves  = hex(0x93c5fd)
)

// ---------------------------------------------------------------------------
// Map
// ---------------------------------------------------------------------------

var (
	MapGradientTop = hex(0x193549)
	MapGradientBot = hex(0x1b4332)
	MapDotGrid     = rgba(255, 255, 255, 8) // 0.03 * 255 ≈ 8
	MapPathStroke  = hex(0xd6d3d1)
	MapPathLabel   = rgba(255, 255, 255, 46) // 0.18 * 255 ≈ 46
)

// ---------------------------------------------------------------------------
// Map Themes
// ---------------------------------------------------------------------------

// MapTheme holds per-map color palette for visual variety.
type MapTheme struct {
	GradientTop color.RGBA
	GradientBot color.RGBA
	PathColor   color.RGBA
	DotColor    color.RGBA
}

// MapThemes maps theme name → palette.
var MapThemes = map[string]MapTheme{
	"desert": {hex(0x2a1a0a), hex(0x1a1208), hex(0xd4a574), rgba(255, 220, 180, 8)},
	"forest": {hex(0x0a1a0f), hex(0x051208), hex(0x8fbc8f), rgba(180, 255, 180, 8)},
	"tech":   {hex(0x0a0f1a), hex(0x080a15), hex(0x88aacc), rgba(150, 200, 255, 8)},
	"stone":  {hex(0x1a150f), hex(0x0f0a08), hex(0xc8b090), rgba(220, 200, 170, 8)},
	"ice":    {hex(0x0a1520), hex(0x051018), hex(0xa8d8ea), rgba(200, 230, 255, 8)},
	"dark":   {hex(0x0a0a10), hex(0x050508), hex(0x9080b0), rgba(180, 160, 220, 8)},
	"lava":   {hex(0x200a0a), hex(0x150505), hex(0xe07050), rgba(255, 150, 100, 8)},
	"void":   {hex(0x0a0520), hex(0x050315), hex(0xb090e0), rgba(200, 180, 255, 8)},
}

// MapThemeFor returns the theme for a given name, falling back to default.
func MapThemeFor(name string) MapTheme {
	if t, ok := MapThemes[name]; ok {
		return t
	}
	return MapTheme{MapGradientTop, MapGradientBot, MapPathStroke, MapDotGrid}
}

// ---------------------------------------------------------------------------
// Tower Slots
// ---------------------------------------------------------------------------

var (
	SlotEmpty      = rgba(255, 255, 255, 12) // 空闲态内部微填充
	SlotIdleRing   = rgba(180, 200, 220, 50) // 空闲态淡灰轮廓
	SlotBuildRing  = rgba(220, 180, 60, 180) // 建造态金黄轮廓
	SlotBuildPulse = rgba(251, 191, 36, 255) // 建造态脉冲外圈
	SlotPlusSign   = rgba(250, 220, 120, 220) // "+" 号颜色
	SlotHintLabel  = hex(0xbfdbfe)
)

// ---------------------------------------------------------------------------
// Tower
// ---------------------------------------------------------------------------

var (
	TowerSelectionRing = rgba(253, 224, 71, 184) // 0.72 * 255 ≈ 184
	TowerRangeFill     = rgba(245, 158, 11, 20)  // 0.08 * 255 ≈ 20
	TowerRangeStroke   = rgba(245, 158, 11, 89)  // 0.35 * 255 ≈ 89
	TowerFallbackSel   = hex(0xf59e0b)
	TowerFallbackDef   = hex(0x38bdf8)
	TowerBarrel        = hex(0x082f49)
	TowerNameLabel     = rgba(255, 255, 255, 217) // 0.85 * 255 ≈ 217
)

// ---------------------------------------------------------------------------
// Buff Dots
// ---------------------------------------------------------------------------

var (
	BuffDamage = hex(0xffd700)
	BuffAtkSpd = hex(0x60a5fa)
	BuffRange  = hex(0x4ade80)
)

// ---------------------------------------------------------------------------
// Enemy
// ---------------------------------------------------------------------------

var (
	EnemyHPBarBorder = hex(0x0f172a)
	EnemyHPBarBg     = hex(0x1e293b)
	EnemyHPBarTrail  = hex(0xfb923c)
	EnemyHPFillHigh  = hex(0xef4444)      // >60%
	EnemyHPFillMid   = hex(0xdc2626)      // >30%
	EnemyHPFillLow   = hex(0x991b1b)      // <=30%
	EnemyHPSegDiv    = rgba(0, 0, 0, 102) // 0.4 * 255 ≈ 102
	EnemyBossInner   = rgba(244, 63, 94, 255)
	EnemyBossOuter   = rgba(251, 113, 133, 255)
	EnemyRunnerPulse = rgba(251, 146, 60, 115)  // 0.45 * 255 ≈ 115
)

// ---------------------------------------------------------------------------
// Projectile
// ---------------------------------------------------------------------------

var (
	ProjDefault   = hex(0xfde68a)
	ProjSniper    = hex(0xfb923c)
	ProjRapid     = hex(0x5eead4)
	ProjFreeze    = hex(0xa5f3fc)
	ProjWind      = hex(0xa3e635)
	ProjWindTrail = hex(0x86efac)
)

// ---------------------------------------------------------------------------
// HUD
// ---------------------------------------------------------------------------

var (
	HUDTopBarBg      = rgba(15, 23, 42, 194)   // 0.76 * 255 ≈ 194
	HUDTopBarBorder  = rgba(255, 255, 255, 20) // 0.08 * 255 ≈ 20
	HUDTopBarDivider = rgba(255, 255, 255, 18) // 0.07 * 255 ≈ 18
	HUDToastBg       = rgba(15, 23, 42, 224)   // 0.88 * 255 ≈ 224
	HUDPauseOverlay  = rgba(2, 6, 23, 107)     // 0.42 * 255 ≈ 107
	HUDPauseMenuBg   = rgba(15, 23, 42, 235)   // 0.92 * 255 ≈ 235
	HUDGameOverlay   = rgba(0, 0, 0, 102)      // 0.4  * 255 ≈ 102
	HUDVictoryColor  = hex(0x22c55e)
	HUDDefeatColor   = hex(0xef4444)
)

// ---------------------------------------------------------------------------
// Build Menu
// ---------------------------------------------------------------------------

var (
	BuildMenuBg        = rgba(15, 23, 42, 224)   // 0.88 * 255 ≈ 224
	BuildCardNormal    = rgba(255, 255, 255, 13) // 0.05 * 255 ≈ 13
	BuildCardSelected  = rgba(34, 197, 94, 56)   // 0.22 * 255 ≈ 56
	BuildCardSelBorder = rgba(74, 222, 128, 168) // 0.66 * 255 ≈ 168
	BuildCardVariant   = rgba(30, 40, 60, 120)   // variant preview card bg
	BuildCostColor     = hex(0xfbbf24)
	BuildTooltipBg     = rgba(15, 23, 42, 235)   // 0.92 * 255 ≈ 235
	BuildTooltipBorder = rgba(79, 140, 255, 102) // 0.4  * 255 ≈ 102
)

// ---------------------------------------------------------------------------
// ActionBar
// ---------------------------------------------------------------------------

var (
	ActionBarBg     = rgba(15, 23, 42, 200)
	ActionBarBorder = rgba(255, 255, 255, 20)
)

// ---------------------------------------------------------------------------
// Info Panel
// ---------------------------------------------------------------------------

var (
	InfoBorder     = rgba(148, 163, 184, 51) // 0.2  * 255 ≈ 51
	InfoWardenBdr  = rgba(251, 191, 36, 77)  // 0.3  * 255 ≈ 77
	InfoAttrDamage = hex(0xfca5a5)           // 红300，更亮
	InfoAttrAtkSpd = hex(0xfdba74)           // 橙300，更亮
	InfoAttrRange  = hex(0x7dd3fc)           // 蓝300，更亮
)

// ---------------------------------------------------------------------------
// Wave Panel
// ---------------------------------------------------------------------------

var (
	WavePanelBg      = rgba(15, 23, 42, 220) // 0.86 * 255 ≈ 220
	WaveThreatDanger = hex(0xfca5a5)
	WaveThreatPress  = hex(0xfde68a)
	WaveThreatCalm   = hex(0x86efac)
)

// ---------------------------------------------------------------------------
// Result / Select
// ---------------------------------------------------------------------------

var (
	ResultBg      = hex(0x0f172a)
	ResultStatsBg = rgba(255, 255, 255, 10) // 0.04 * 255 ≈ 10
	SelectGradTop = hex(0x0b1220)
	SelectGradBot = hex(0x172554)
)

// ---------------------------------------------------------------------------
// Debug / Overlay
// ---------------------------------------------------------------------------

var (
	DebugTextClr  = rgba(180, 190, 210, 200)
	DebugStatsClr = rgba(160, 180, 200, 220)
	OverlayHeavy  = rgba(0, 0, 0, 180)
)
