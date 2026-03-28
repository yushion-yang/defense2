# Visual Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Faithfully replicate the JS original's visual style in the Go/Ebitengine port — dark-blue theme, polished HUD, detailed panels, animations.

**Architecture:** Create a centralized Design System (theme package) and drawing primitives library, then rewrite each renderer and HUD component to match JS original parameters. FontManager becomes global singleton replacing all DebugPrintAt usage.

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9 (vector, text/v2), zpix.ttf

**Reference Source:** `/Users/yushion/Games/tower-defense/src/renderer/canvas/`

**Design Doc:** `docs/plans/2026-03-28-visual-overhaul-design.md`

---

## Phase 1: Infrastructure (Foundation)

### Task 1: Design System — Theme Colors

**Files:**
- Create: `internal/render/theme/colors.go`

**Step 1: Create theme package with all color constants**

```go
// colors.go — Centralized color constants matching JS uiConstants.js
package theme

import "image/color"

// Hex helper
func hex(hexStr uint32) color.RGBA {
	return color.RGBA{
		R: uint8(hexStr >> 16),
		G: uint8(hexStr >> 8),
		B: uint8(hexStr),
		A: 255,
	}
}

func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// ── Canvas ──
var (
	CanvasBg = hex(0x0f172a) // Main background
)

// ── Panel ──
var (
	PanelBg     = rgba(15, 23, 42, 224)  // 0.88 alpha
	PanelBorder = rgba(148, 163, 184, 51) // 0.2 alpha
)

// ── Text ──
var (
	TextTitle  = hex(0xe2e8f0) // Primary headings
	TextBody   = hex(0xcbd5e1) // Body text
	TextMuted  = hex(0x94a3b8) // Secondary/muted
	TextLocked = hex(0x4b5563) // Disabled
)

// ── Status ──
var (
	ColorStrUp     = hex(0x4ade80) // Buff/growth (green)
	ColorStrDown   = hex(0xf87171) // Debuff/danger (red)
	ColorStrNormal = hex(0x94a3b8) // Neutral
	ColorSkill     = hex(0x60a5fa) // Skill accent (blue)
	ColorExclusive = hex(0x38bdf8) // Exclusive
	ColorWardenGold = hex(0xfbbf24) // Warden gold
	ColorGrowth    = hex(0x4ade80) // Growth
)

// ── Buttons ──
var (
	BtnPrimary   = hex(0x3b82f6) // Blue primary
	BtnDanger    = hex(0xef4444) // Red danger
	BtnSecondary = hex(0x475569) // Gray secondary
	BtnMuted     = hex(0x334155) // Dark muted
)

// ── Button tones (with alpha) ──
var (
	BtnTonePrimary   = rgba(34, 197, 94, 245)  // Green (start wave)
	BtnToneDanger    = rgba(239, 68, 68, 230)   // Red
	BtnToneAccent    = rgba(59, 130, 246, 235)   // Blue (speed)
	BtnToneSecondary = rgba(15, 23, 42, 217)     // Dark (menu)
	BtnToneDisabled  = rgba(30, 41, 59, 97)      // Dim
)

// ── Faction ──
var (
	FactionBase    = hex(0x38bdf8) // Cyan
	FactionOutput  = hex(0xf87171) // Red
	FactionControl = hex(0x60a5fa) // Blue
	FactionSupport = hex(0x4ade80) // Green
)

// ── Resources ──
var (
	ColorHearts = hex(0xf87171) // Lives
	ColorGold   = hex(0xfbbf24) // Currency
	ColorWaves  = hex(0x93c5fd) // Wave counter
)

// ── Map ──
var (
	MapGradientTop    = hex(0x193549)
	MapGradientBottom = hex(0x1b4332)
	MapDotGrid        = rgba(255, 255, 255, 8) // 0.03 alpha
	PathStroke        = hex(0xd6d3d1)
	PathDash          = hex(0x9ca3af)
	PathLabel         = rgba(255, 255, 255, 46) // 0.18 alpha
)

// ── Tower Slots ──
var (
	SlotEmpty      = rgba(255, 255, 255, 20) // 0.08 alpha
	SlotOccupied   = rgba(34, 197, 94, 36)    // 0.14 alpha
	SlotBuildPulse = rgba(251, 191, 36, 255)   // Gold pulse (alpha modulated at runtime)
	SlotHintPulse  = rgba(96, 165, 250, 255)   // Blue hint pulse
	SlotPlusSign   = hex(0xfde68a)
	SlotHintLabel  = hex(0xbfdbfe)
)

// ── Tower ──
var (
	TowerSelectionRing = rgba(253, 224, 71, 184)  // 0.72 alpha
	TowerRangeFill     = rgba(245, 158, 11, 20)   // 0.08 alpha
	TowerRangeStroke   = rgba(245, 158, 11, 89)   // 0.35 alpha
	TowerFallbackSel   = hex(0xf59e0b)
	TowerFallbackDef   = hex(0x38bdf8)
	TowerBarrel        = hex(0x082f49)
	TowerNameLabel     = rgba(255, 255, 255, 217)  // 0.85 alpha
)

// ── Tower Buff Dots ──
var (
	BuffDamage  = hex(0xffd700)
	BuffAtkSpd  = hex(0x60a5fa)
	BuffRange   = hex(0x4ade80)
	BuffCrit    = hex(0xc084fc)
	BuffStr     = hex(0x22d3ee)
)

// ── Enemy ──
var (
	HPBarBorder  = hex(0x0f172a)
	HPBarBg      = hex(0x1e293b)
	HPBarTrail   = hex(0xfb923c) // Damage flash
	HPFillHigh   = hex(0xef4444) // > 60%
	HPFillMid    = hex(0xdc2626) // > 30%
	HPFillLow    = hex(0x991b1b) // <= 30%
	HPSegmentDiv = rgba(0, 0, 0, 102) // 0.4 alpha

	BossRingInner = rgba(244, 63, 94, 255)  // Alpha modulated at runtime
	BossRingOuter = rgba(251, 113, 133, 255)
	RunnerPulse   = rgba(251, 146, 60, 115)  // 0.45 alpha
	SwarmTriangle = rgba(254, 240, 138, 184) // 0.72 alpha
	FlyingShadow  = rgba(15, 23, 42, 51)      // 0.2 alpha
)

// ── Projectile ──
var (
	ProjDefault = hex(0xfde68a)
	ProjSniper  = hex(0xfb923c)
	ProjRapid   = hex(0x5eead4)
	ProjFreeze  = hex(0xa5f3fc)
	ProjWind    = hex(0xa3e635)
	ProjWindTrail = hex(0x86efac)
)

// ── Hero ──
var (
	HeroLeashSelected = rgba(192, 132, 252, 133)  // 0.52 alpha
	HeroLeashDefault  = rgba(125, 211, 252, 46)   // 0.18 alpha
	HeroBodyFallback  = hex(0x4c1d95)
	HeroStroke        = hex(0xc4b5fd)
	HeroXPBarBg       = rgba(255, 255, 255, 20)   // 0.08 alpha
	HeroXPBarFill     = hex(0xa78bfa)
)

// ── HUD ──
var (
	TopBarBg      = rgba(15, 23, 42, 194)    // 0.76 alpha
	TopBarBorder  = rgba(255, 255, 255, 20)  // 0.08 alpha
	TopBarDivider = rgba(255, 255, 255, 18)  // 0.07 alpha
	ToastBg       = rgba(15, 23, 42, 224)    // 0.88 alpha
	PauseOverlay  = rgba(2, 6, 23, 107)      // 0.42 alpha
	PauseMenuBg   = rgba(15, 23, 42, 235)    // 0.92 alpha
	GameOverlay   = rgba(0, 0, 0, 102)       // 0.4 alpha
	VictoryColor  = hex(0x22c55e)
	DefeatColor   = hex(0xef4444)
)

// ── Build Menu ──
var (
	BuildMenuBg       = rgba(15, 23, 42, 224) // 0.88 alpha
	BuildCardNormal   = rgba(255, 255, 255, 13) // 0.05 alpha
	BuildCardSelected = rgba(34, 197, 94, 56)  // 0.22 alpha
	BuildCardSelBorder = rgba(74, 222, 128, 168) // 0.66 alpha
	BuildCostColor    = hex(0xfbbf24)
	BuildTooltipBg    = rgba(15, 23, 42, 235) // 0.92 alpha
	BuildTooltipBorder = rgba(79, 140, 255, 102) // 0.4 alpha
)

// ── Info Panel ──
var (
	InfoPanelBorder      = rgba(148, 163, 184, 51) // tower
	InfoPanelWardenBorder = rgba(251, 191, 36, 77)  // warden 0.3 alpha
	AttrDamageColor      = hex(0xf87171)
	AttrAtkSpdColor      = hex(0xfb923c)
	AttrRangeColor       = hex(0x38bdf8)
)

// ── Wave Panel ──
var (
	WavePanelBg     = rgba(15, 23, 42, 184) // 0.72 alpha
	ThreatDanger    = hex(0xfca5a5)
	ThreatPressure  = hex(0xfde68a)
	ThreatCalm      = hex(0x86efac)
)

// ── Result Screen ──
var (
	ResultBg        = hex(0x0f172a)
	ResultStatsBg   = rgba(255, 255, 255, 10) // 0.04 alpha
	SelectGradTop   = hex(0x0b1220)
	SelectGradBot   = hex(0x172554)
)
```

**Step 2: Verify it compiles**

Run: `cd /Users/yushion/Games/defense2 && go build ./internal/render/theme/`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/render/theme/colors.go
git commit -m "feat: add centralized theme color constants from JS design system"
```

---

### Task 2: Design System — Layout & DS Constants

**Files:**
- Create: `internal/render/theme/layout.go`
- Create: `internal/render/theme/ds.go`

**Step 1: Create layout constants**

```go
// layout.go — HUD layout constants matching JS LAYOUT object.
package theme

// Canvas dimensions
const (
	CanvasW = 1200
	CanvasH = 540
)

// Top bar
const (
	TopBarY             = 6
	TopBarH             = 40
	TopBarW             = 760
	TopBarDividerOffset = 290
	TopBarBtnH          = 30
	TopBarBtnGap        = 6
	TopBarRadius        = 16
)

// Top bar button widths
const (
	BtnBuildW = 66
	BtnStartW = 82
	BtnSpeedW = 78
	BtnMenuW  = 54
	BtnRadius = 14
)

// Badge
const (
	BadgeW   = 200
	BadgeH   = 24
	BadgeGap = 4
)

// Toast
const (
	ToastH            = 34
	ToastW            = 500
	ToastFadeDuration = 0.3 // seconds
	ToastRadius       = 10
)

// Info panel (wave preview, left side)
const (
	InfoPanelX          = 14
	InfoPanelW          = 220
	InfoPanelCollapsedW = 54
	InfoPanelLineH      = 18
)

// Center panel (tower detail)
const (
	CenterPanelW = 500
	CenterPanelX = (CanvasW - CenterPanelW) / 2 // 350
	CenterPanelRadius = 14
	CenterPanelMinH   = 100
	CenterPanelInnerPad = 14
)

// Bottom margin
const (
	BottomMargin = 14
)

// Tower visual constants
const (
	TowerBaseSize          = 64
	TowerSelectionRingR    = 24
	TowerSummonCircleR     = 28
	TowerNameLabelY        = 36  // Below center
	TowerBuffDotBaseY      = 26  // Above center
	TowerBuffDotSpacing    = 8
	TowerBuffDotRadius     = 3
	TowerRangeStrokeWidth  = 1.5
	TowerSelectionWidth    = 2
	TowerFallbackRadius    = 18
)

// Enemy visual constants
const (
	EnemySpriteScale   = 3.15
	EnemyHPBarW        = 36
	EnemyHPBarH        = 5
	EnemyHPBarOffsetY  = 26
	BossHPBarW         = 48
	BossHPBarH         = 7
	BossHPBarOffsetY   = 30
	ShieldBarH         = 3
)

// Build menu
const (
	BuildMenuRadius  = 18
	BuildCardW       = 88
	BuildCardH       = 56
	BuildCardGap     = 6
	BuildCardRadius  = 10
	BuildCardCols    = 5
	BuildCardsPerPage = 10
	FactionTabH      = 20
	FactionTabGap    = 3
	FactionTabRadius = 6
)

// Tooltip
const (
	TooltipBuildW  = 260
	TooltipHoverW  = 280
	TooltipLineH   = 16
	TooltipPad     = 10
	TooltipRadius  = 10
)

// Wave panel
const (
	WavePanelRadius = 16
)

// Map drawing
const (
	PathStrokeWidth = 52
	PathDashWidth   = 4
	PathDashOn      = 10
	PathDashOff     = 10
	SlotRadius      = 26
	DotGridSpacing  = 40
	DotGridSize     = 2
)

// Projectile
const (
	ProjDefaultR    = 4
	ProjDefaultGlow = 8
	ProjSniperR     = 5
	ProjSniperGlow  = 10
)
```

**Step 2: Create DS tokens**

```go
// ds.go — Design System spacing, font sizes, and line heights.
package theme

// Spacing
const (
	PadXS = 4
	PadSM = 8
	PadMD = 12
	PadLG = 16
)

// Font sizes (pixels)
const (
	FontXS = 10
	FontSM = 11
	FontMD = 12
	FontLG = 14
	FontXL = 16
)

// Special font sizes
const (
	FontTopBar     = 17
	FontMapLabel   = 16
	FontTowerName  = 10
	FontGameOver   = 52
	FontResultTitle = 42
	FontWardenTitle = 24
	FontSubtitle   = 13
)

// Line heights
const (
	LineHAbility = 16
	LineHAttr    = 18
	LineHTitle   = 22
	LineHBtn     = 36
)

// Panel
const (
	PanelRadius   = 14
	PanelMinH     = 100
	PanelInnerPad = 14
)

// Button
const (
	ButtonH      = 30
	ButtonRadius = 12
	ButtonGap    = 8
)

// Detail panel sections
const (
	DetailPad    = 14
	DetailTopPad = 10
	DetailBotPad = 10
	DetailGap    = 6
	DetailTitleH = 18
	DetailAttrH  = 16
	DetailRowH   = 15
	DetailBtnH   = 34
)
```

**Step 3: Verify**

Run: `go build ./internal/render/theme/`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/render/theme/layout.go internal/render/theme/ds.go
git commit -m "feat: add layout and design system constants for visual overhaul"
```

---

### Task 3: Drawing Primitives Library

**Files:**
- Create: `internal/render/draw/roundrect.go`
- Create: `internal/render/draw/gradient.go`
- Create: `internal/render/draw/dashed.go`
- Create: `internal/render/draw/circle.go`

**Step 1: Create roundrect.go — rounded rectangle fill and stroke**

```go
// roundrect.go — Rounded rectangle drawing for Ebitengine.
package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// RoundRect draws a filled rounded rectangle.
func RoundRect(screen *ebiten.Image, x, y, w, h, radius float32, clr color.Color) {
	if radius <= 0 {
		vector.DrawFilledRect(screen, x, y, w, h, clr, true)
		return
	}
	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}

	var path vector.Path
	path.MoveTo(x+radius, y)
	path.LineTo(x+w-radius, y)
	path.ArcTo(x+w, y, x+w, y+radius, radius)
	path.LineTo(x+w, y+h-radius)
	path.ArcTo(x+w, y+h, x+w-radius, y+h, radius)
	path.LineTo(x+radius, y+h)
	path.ArcTo(x, y+h, x, y+h-radius, radius)
	path.LineTo(x, y+radius)
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	r32, g32, b32, a32 := clr.RGBA()
	rf := float32(r32) / 0xffff
	gf := float32(g32) / 0xffff
	bf := float32(b32) / 0xffff
	af := float32(a32) / 0xffff
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = rf
		vs[i].ColorG = gf
		vs[i].ColorB = bf
		vs[i].ColorA = af
	}

	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	screen.DrawTriangles(vs, is, whitePixel(), op)
}

// StrokeRoundRect draws a rounded rectangle outline.
func StrokeRoundRect(screen *ebiten.Image, x, y, w, h, radius, strokeWidth float32, clr color.Color) {
	if radius <= 0 {
		StrokeRect(screen, x, y, w, h, strokeWidth, clr)
		return
	}

	var path vector.Path
	path.MoveTo(x+radius, y)
	path.LineTo(x+w-radius, y)
	path.ArcTo(x+w, y, x+w, y+radius, radius)
	path.LineTo(x+w, y+h-radius)
	path.ArcTo(x+w, y+h, x+w-radius, y+h, radius)
	path.LineTo(x+radius, y+h)
	path.ArcTo(x, y+h, x, y+h-radius, radius)
	path.LineTo(x, y+radius)
	path.ArcTo(x, y, x+radius, y, radius)
	path.Close()

	sop := &vector.StrokeOptions{Width: strokeWidth}
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, sop)
	r32, g32, b32, a32 := clr.RGBA()
	rf := float32(r32) / 0xffff
	gf := float32(g32) / 0xffff
	bf := float32(b32) / 0xffff
	af := float32(a32) / 0xffff
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = rf
		vs[i].ColorG = gf
		vs[i].ColorB = bf
		vs[i].ColorA = af
	}

	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	screen.DrawTriangles(vs, is, whitePixel(), op)
}

// StrokeRect draws a simple rectangle outline.
func StrokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.Color) {
	vector.StrokeLine(screen, x, y, x+w, y, width, clr, true)
	vector.StrokeLine(screen, x+w, y, x+w, y+h, width, clr, true)
	vector.StrokeLine(screen, x+w, y+h, x, y+h, width, clr, true)
	vector.StrokeLine(screen, x, y+h, x, y, width, clr, true)
}

var whitePixelImage *ebiten.Image

func whitePixel() *ebiten.Image {
	if whitePixelImage != nil {
		return whitePixelImage
	}
	whitePixelImage = ebiten.NewImage(3, 3)
	whitePixelImage.Fill(color.White)
	return whitePixelImage
}

// fillPath is a reusable helper to fill a vector.Path with a given color.
func fillPath(screen *ebiten.Image, path *vector.Path, clr color.Color) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	r32, g32, b32, a32 := clr.RGBA()
	rf := float32(r32) / 0xffff
	gf := float32(g32) / 0xffff
	bf := float32(b32) / 0xffff
	af := float32(a32) / 0xffff
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = rf
		vs[i].ColorG = gf
		vs[i].ColorB = bf
		vs[i].ColorA = af
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	screen.DrawTriangles(vs, is, whitePixel(), op)
}

// Pill draws a pill shape (fully rounded rectangle where radius = h/2).
func Pill(screen *ebiten.Image, x, y, w, h float32, clr color.Color) {
	RoundRect(screen, x, y, w, h, h/2, clr)
}

// unused import guard
var _ = math.Pi
```

**Step 2: Create gradient.go**

```go
// gradient.go — Linear gradient rendering for Ebitengine.
package draw

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LinearGradientV draws a vertical linear gradient filling the given rectangle.
func LinearGradientV(screen *ebiten.Image, x, y, w, h int, top, bottom color.RGBA) {
	if h <= 0 || w <= 0 {
		return
	}
	img := ebiten.NewImage(1, h)
	for row := 0; row < h; row++ {
		t := float64(row) / float64(h-1)
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		a := uint8(float64(top.A)*(1-t) + float64(bottom.A)*t)
		img.Set(0, row, color.RGBA{R: r, G: g, B: b, A: a})
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), 1)
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, op)
}

// CachedGradient pre-renders a gradient image for reuse across frames.
type CachedGradient struct {
	img    *ebiten.Image
	w, h   int
	top    color.RGBA
	bottom color.RGBA
}

// NewCachedGradient creates a reusable gradient image.
func NewCachedGradient(w, h int, top, bottom color.RGBA) *CachedGradient {
	cg := &CachedGradient{w: w, h: h, top: top, bottom: bottom}
	cg.img = ebiten.NewImage(w, h)
	pixels := make([]byte, w*h*4)
	for row := 0; row < h; row++ {
		t := float64(row) / float64(h-1)
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		a := uint8(float64(top.A)*(1-t) + float64(bottom.A)*t)
		for col := 0; col < w; col++ {
			idx := (row*w + col) * 4
			pixels[idx] = r
			pixels[idx+1] = g
			pixels[idx+2] = b
			pixels[idx+3] = a
		}
	}
	cg.img.WritePixels(pixels)
	return cg
}

// Draw renders the cached gradient at (x, y).
func (cg *CachedGradient) Draw(screen *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(cg.img, op)
}

// Image returns the underlying image for sub-region drawing.
func (cg *CachedGradient) Image() *ebiten.Image {
	return cg.img
}

var _ = image.Rect // unused import guard
```

**Step 3: Create dashed.go**

```go
// dashed.go — Dashed line rendering for Ebitengine.
package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DashedLine draws a dashed line between two points.
func DashedLine(screen *ebiten.Image, x1, y1, x2, y2, width, dashLen, gapLen float32, clr color.Color) {
	dx := x2 - x1
	dy := y2 - y1
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}

	nx := dx / totalLen // normalized direction
	ny := dy / totalLen
	segLen := dashLen + gapLen
	pos := float32(0)

	for pos < totalLen {
		end := pos + dashLen
		if end > totalLen {
			end = totalLen
		}
		sx := x1 + nx*pos
		sy := y1 + ny*pos
		ex := x1 + nx*end
		ey := y1 + ny*end
		vector.StrokeLine(screen, sx, sy, ex, ey, width, clr, true)
		pos += segLen
	}
}

// DashedCircle draws a dashed circle outline.
func DashedCircle(screen *ebiten.Image, cx, cy, r, width, dashLen, gapLen float32, clr color.Color) {
	circumference := float32(2 * math.Pi * float64(r))
	segLen := dashLen + gapLen
	numSegs := int(circumference / segLen)
	if numSegs < 1 {
		numSegs = 1
	}

	dashAngle := float64(dashLen) / float64(r)
	gapAngle := float64(gapLen) / float64(r)
	angle := float64(0)

	for i := 0; i < numSegs && angle < 2*math.Pi; i++ {
		endAngle := angle + dashAngle
		if endAngle > 2*math.Pi {
			endAngle = 2 * math.Pi
		}
		// Draw arc segment as line segments
		steps := 8
		for j := 0; j < steps; j++ {
			a1 := angle + float64(j)*(endAngle-angle)/float64(steps)
			a2 := angle + float64(j+1)*(endAngle-angle)/float64(steps)
			x1 := cx + r*float32(math.Cos(a1))
			y1 := cy + r*float32(math.Sin(a1))
			x2 := cx + r*float32(math.Cos(a2))
			y2 := cy + r*float32(math.Sin(a2))
			vector.StrokeLine(screen, x1, y1, x2, y2, width, clr, true)
		}
		angle = endAngle + gapAngle
	}
}
```

**Step 4: Create circle.go — enhanced circle drawing**

```go
// circle.go — Enhanced circle drawing utilities.
package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// CircleOutline draws a circle outline using line segments.
func CircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	const segments = 48
	for i := 0; i < segments; i++ {
		a1 := float64(i) * 2 * math.Pi / segments
		a2 := float64(i+1) * 2 * math.Pi / segments
		x1 := cx + r*float32(math.Cos(a1))
		y1 := cy + r*float32(math.Sin(a1))
		x2 := cx + r*float32(math.Cos(a2))
		y2 := cy + r*float32(math.Sin(a2))
		vector.StrokeLine(screen, x1, y1, x2, y2, width, clr, true)
	}
}

// FilledCircle draws a filled circle (convenience wrapper).
func FilledCircle(screen *ebiten.Image, cx, cy, r float32, clr color.Color) {
	vector.DrawFilledCircle(screen, cx, cy, r, clr, true)
}

// Glow draws a soft glow effect (outer transparent circle + inner solid).
func Glow(screen *ebiten.Image, cx, cy, innerR, outerR float32, clr color.RGBA) {
	// Outer glow
	glowClr := color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 4}
	vector.DrawFilledCircle(screen, cx, cy, outerR, glowClr, true)
	// Inner solid
	vector.DrawFilledCircle(screen, cx, cy, innerR, clr, true)
}

// Diamond draws a diamond (rotated square) outline.
func Diamond(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	vector.StrokeLine(screen, cx, cy-r, cx+r, cy, width, clr, true)
	vector.StrokeLine(screen, cx+r, cy, cx, cy+r, width, clr, true)
	vector.StrokeLine(screen, cx, cy+r, cx-r, cy, width, clr, true)
	vector.StrokeLine(screen, cx-r, cy, cx, cy-r, width, clr, true)
}

// ThickLine draws a line using a filled path (for rounded caps).
func ThickLine(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color) {
	var path vector.Path

	dx := x2 - x1
	dy := y2 - y1
	l := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if l < 0.001 {
		return
	}
	nx := -dy / l * width / 2
	ny := dx / l * width / 2

	path.MoveTo(x1+nx, y1+ny)
	path.LineTo(x2+nx, y2+ny)
	path.LineTo(x2-nx, y2-ny)
	path.LineTo(x1-nx, y1-ny)
	path.Close()

	fillPath(screen, &path, clr)

	// Round caps
	FilledCircle(screen, x1, y1, width/2, clr)
	FilledCircle(screen, x2, y2, width/2, clr)
}
```

**Step 5: Verify**

Run: `go build ./internal/render/draw/`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add internal/render/draw/
git commit -m "feat: add drawing primitives library (roundrect, gradient, dashed, circle)"
```

---

### Task 4: FontManager Global Singleton

**Files:**
- Modify: `internal/render/font.go`

**Step 1: Add global singleton and right-aligned text**

Add to `font.go`:
- `var defaultFM *FontManager` + `InitGlobalFont(ttfData)` + `GlobalFont() *FontManager`
- `DrawRightText` method for right-aligned text

**Step 2: Verify**

Run: `go build ./internal/render/`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/render/font.go
git commit -m "feat: make FontManager a global singleton for all scenes"
```

---

### Task 5: Wire Warden PNGs into embed

**Files:**
- Modify: `data.go`

**Step 1: Add warden PNG embed**

Change the AssetFS embed directive to include `assets/wardens/*.png`:

```go
//go:embed assets/towers/*/*.png assets/enemies/*.png assets/wardens/*.png assets/audio/*.wav assets/fonts/*.ttf
var AssetFS embed.FS
```

**Step 2: Verify**

Run: `go build .`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add data.go
git commit -m "feat: embed warden PNG sprites for rendering"
```

---

## Phase 2: Map Rendering

### Task 6: Rewrite draw_map.go

**Files:**
- Rewrite: `internal/render/draw_map.go`

**Step 1: Rewrite with gradient background, thick path, slot circles**

The new `draw_map.go` should:
1. Draw vertical gradient background (`MapGradientTop` → `MapGradientBottom`) using `CachedGradient`
2. Draw dot grid overlay (2x2 dots every 40px)
3. Draw path as 52px thick rounded line (using `ThickLine` for each segment)
4. Draw center dashed line on path (`PathDash` color, 4px wide, [10,10] pattern)
5. Draw "入口" / "基地" labels at path start/end
6. Draw tower slots as r=26 circles (`SlotEmpty` color)
7. Occupied slots get `SlotOccupied` fill
8. Remove old per-cell grid fill entirely

Key: `DrawMap` needs access to `FontManager` for labels and `animTime` for pulse animations. Change signature to: `DrawMap(screen, gm, fm *FontManager, animTime float64)` or pass a `MapDrawContext`.

**Step 2: Verify**

Run: `go build ./internal/render/`
Expected: SUCCESS (may have unused import warnings to fix in stage.go)

**Step 3: Commit**

```bash
git add internal/render/draw_map.go
git commit -m "feat: rewrite map rendering — gradient bg, thick path, slot circles"
```

---

## Phase 3: Entity Rendering

### Task 7: Rewrite draw_tower.go

**Files:**
- Rewrite: `internal/render/draw_tower.go`

**Step 1: Rewrite with selection ring, buff dots, aura effects**

Changes from current:
- `towerSpriteSize` → 64 (was 40)
- Add `animTime float64` parameter to `DrawTowers`
- Selected tower: gold selection ring (r=24, `TowerSelectionRing`)
- Range indicator: fill + stroke (only when selected)
- Fallback: circle r=18 + barrel rectangle, not square
- Name label below tower (10px font via FontManager)
- Buff dots above tower (colored by type)
- Aura pulse animation for aura-type towers
- Charge visual for charging towers
- `drawCircleOutline` moves to `draw` package, remove from here

**Step 2: Verify**

Run: `go build ./internal/render/`

**Step 3: Commit**

```bash
git add internal/render/draw_tower.go
git commit -m "feat: rewrite tower rendering — selection ring, buff dots, aura effects"
```

---

### Task 8: Rewrite draw_enemy.go

**Files:**
- Rewrite: `internal/render/draw_enemy.go`

**Step 1: Rewrite with proper HP bars, boss effects, status visuals**

Changes:
- `enemySpriteSize` → use `EnemySpriteScale * baseSize` logic
- HP bar: styled with border (`HPBarBorder`), background (`HPBarBg`), trail (`HPBarTrail`), color gradient by HP%
- Boss: wider bar (48x7), segment lines every 20%, dual pulsing rings
- Shield bar above HP bar, white, h=3
- Runner: pulsing orange ring
- Swarm: jittering yellow triangles
- Tank: white 14x14 square overlay
- Flying: ground shadow ellipse
- Status effect rings/dots with proper colors

**Step 2: Verify**

Run: `go build ./internal/render/`

**Step 3: Commit**

```bash
git add internal/render/draw_enemy.go
git commit -m "feat: rewrite enemy rendering — HP bars, boss aura, status effects"
```

---

### Task 9: Rewrite draw_projectile.go

**Files:**
- Rewrite: `internal/render/draw_projectile.go`

**Step 1: Per-type projectile rendering with glow**

Changes:
- Dispatch by `Projectile.Type` or `Projectile.TowerKey`
- Default: r=4 + glow r=8, `ProjDefault`
- Sniper: r=5 + glow, `ProjSniper`
- Rapid: `ProjRapid`
- Freeze: diamond shape `ProjFreeze`
- Wind: crescent shape (can approximate with arc)
- Glow = larger transparent circle behind

**Step 2: Verify & commit**

---

### Task 10: Rewrite draw_hero.go

**Files:**
- Rewrite: `internal/render/draw_hero.go`

**Step 1: Dashed leash, proper XP pill bar**

Changes:
- Leash: dashed circle (selected vs default colors)
- Body fallback: `HeroBodyFallback` + stroke `HeroStroke`
- XP bar: pill shape with `HeroXPBarBg` and `HeroXPBarFill`
- Level label via FontManager (not DebugPrintAt)

**Step 2: Verify & commit**

---

### Task 11: Rewrite draw_warden.go

**Files:**
- Rewrite: `internal/render/draw_warden.go`

**Step 1: Wire PNG sprites + enhanced fallback effects**

Changes:
- Add sprite loading (similar to TowerRenderer pattern) for `assets/wardens/warden-{type}.png`
- Prince: diamond fallback `#f97316`/`#fbbf24`, dash trail
- Envoy: golden hexagram, possess glow
- Core: blue triangle with radial glow
- Keep existing geometric fallbacks but update colors to match JS palette

**Step 2: Verify & commit**

---

## Phase 4: HUD System

### Task 12: Rewrite HUD layout.go

**Files:**
- Rewrite: `internal/render/hud/layout.go`

**Step 1: Replace with theme constants**

Replace all hardcoded values with references to `theme` package. The layout struct can be simplified or removed since `theme/layout.go` now holds all constants.

**Step 2: Verify & commit**

---

### Task 13: Rewrite top_bar.go

**Files:**
- Rewrite: `internal/render/hud/top_bar.go`

**Step 1: Pill-shaped top bar with icons and button group**

New `DrawTopBar` should:
1. Draw centered rounded pill (760x40, y=6, radius=16, `TopBarBg`)
2. Draw border (`TopBarBorder`, 1px)
3. Left section: heart icon (two arcs filled `ColorHearts`) + lives text, coin icon (circle `ColorGold`) + gold text, waves icon + wave text — all bold 17px
4. Divider line at x-offset 290
5. Right section: button group:
   - "造塔" (green `BtnTonePrimary`)
   - "开波" (green)
   - "x1"/"x2" (blue `BtnToneAccent`)
   - "菜单" (dark `BtnToneSecondary`)
   - Each with rounded rect (radius=14), white text bold 12px
6. Buttons need hit test support (return which button was clicked)

`TopBarData` needs additional fields: `SpeedMultiplier int`, `BuildMode bool`, etc.

**Step 2: Verify & commit**

---

### Task 14: Rewrite build_menu.go

**Files:**
- Rewrite: `internal/render/hud/build_menu.go`

**Step 1: Card-style build menu with faction tabs**

New `DrawBuildMenu`:
1. Panel background with rounded corners (radius=18)
2. Tower cards (88x56) with rounded corners (radius=10)
3. Selected card: green tint + green border
4. Tower sprite icon in card center
5. Name (bold 12px white) + cost (`BuildCostColor` 11px)
6. Faction tabs at top (colored by faction)
7. Tooltip on hover (260px wide, rounded corners)
8. Update `BuildMenuHitTest` for new layout

**Step 2: Verify & commit**

---

### Task 15: Rewrite info_panel.go (tower detail)

**Files:**
- Rewrite: `internal/render/hud/info_panel.go`

**Step 1: Bottom-center detail panel**

New `DrawInfoPanel`:
1. Position: bottom-center (x=350, y=540-H-14)
2. Width=500, rounded corners (14), `PanelBg`, border `InfoPanelBorder`
3. Title row: tower name `TextTitle` bold 14px + strength value right-aligned
4. Attribute row: 3 columns (DMG red, ATK SPD orange, RNG blue) — 12px
5. Ability rows: 15px each
6. Buff rows: max 6, colored dots
7. Skill row: `ColorSkill`
8. Button row: sell button (red, h=34, radius=12)
9. Auto-calculate height based on content

**Step 2: Verify & commit**

---

### Task 16: Create wave_panel.go

**Files:**
- Create: `internal/render/hud/wave_panel.go`

**Step 1: Wave preview panel**

```go
// wave_panel.go — Left-side wave preview panel.
package hud
```

Features:
1. Position: left x=14, bottom-aligned
2. Width=220, radius=16, bg `WavePanelBg`
3. Wave number text `ColorWaves`
4. Threat level indicator (danger/pressure/calm colors)
5. Enemy type list with counts

`WavePanelData` struct: `WaveNum int`, `ThreatLevel string`, `Enemies []WaveEnemyInfo`

**Step 2: Verify & commit**

---

### Task 17: Create toast.go

**Files:**
- Create: `internal/render/hud/toast.go`

**Step 1: Toast message system**

```go
// toast.go — Toast message overlay.
package hud
```

Features:
1. 500x34, centered, `ToastBg`, radius=10
2. 300ms fade in/out
3. `Toast` struct with message, timer, alpha
4. `ShowToast(msg string)`, `UpdateToast(dt float64)`, `DrawToast(screen)`

**Step 2: Verify & commit**

---

## Phase 5: Scene Integration

### Task 18: Update stage.go — wire everything together

**Files:**
- Modify: `internal/scene/stage.go`

**Step 1: Add animTime, FontManager, update Draw()**

Changes to `StageScene`:
1. Add `animTime float64` field, increment in `Update()` by `1.0/60.0`
2. Add `fontMgr *render.FontManager` field, initialize from global
3. Update `Draw()`:
   - Remove `screen.Fill(...)` green background
   - Call `render.DrawMap(screen, s.gameMap, s.fontMgr, s.animTime)`
   - Pass `animTime` to tower/enemy renderers
   - Replace all `ebitenutil.DebugPrintAt` with FontManager calls
   - Wire toast system
   - Wire wave panel
   - Update victory/defeat overlay with styled text

**Step 2: Verify**

Run: `make run` (visual check)

**Step 3: Commit**

```bash
git add internal/scene/stage.go
git commit -m "feat: wire new visual system into stage scene"
```

---

### Task 19: Update select.go — tune to match JS params

**Files:**
- Modify: `internal/scene/select.go`

**Step 1: Align gradient, card sizes, button style to JS**

Changes:
- Background gradient: `SelectGradTop` → `SelectGradBot`
- Card dimensions: 180x120 (if different), radius=14
- Start button: `#22c55e`, radius=14, text `#052e16`
- Import theme package for colors

**Step 2: Verify & commit**

---

### Task 20: Rewrite result.go

**Files:**
- Rewrite: `internal/scene/result.go`

**Step 1: Dark blue theme + FontManager + stats panel**

Changes:
- Background: `ResultBg` (`#0f172a`)
- Victory text: `VictoryColor` bold 42px / Defeat: `DefeatColor` bold 42px
- Stats panel: 500x220, `ResultStatsBg`, radius=16
- All text via FontManager
- Remove all `ebitenutil.DebugPrintAt`

**Step 2: Verify & commit**

---

## Phase 6: Testing & Polish

### Task 21: Write tests for drawing primitives

**Files:**
- Create: `tests/render/draw_test.go`

**Step 1: Test roundrect, gradient, dashed line**

Table-driven tests verifying:
- RoundRect doesn't panic with edge cases (zero size, large radius)
- CachedGradient produces correct pixel colors
- DashedLine with various dash/gap lengths
- CircleOutline basic rendering

**Step 2: Run tests**

Run: `go test -race ./tests/render/...`

**Step 3: Commit**

---

### Task 22: Write tests for theme constants

**Files:**
- Create: `tests/render/theme_test.go`

**Step 1: Test color hex helper and key constants**

Verify hex() produces correct RGBA values for known inputs. Spot-check key colors.

**Step 2: Run & commit**

---

### Task 23: Visual smoke test

**Step 1: Run the game**

Run: `make run`

**Step 2: Verify visually**
- [ ] Background is dark blue gradient (not green)
- [ ] Path is thick with dashed center line
- [ ] Tower slots are semi-transparent circles
- [ ] Text uses custom font (not green debug text)
- [ ] Top bar is pill-shaped with colored icons
- [ ] Build menu has card-style slots
- [ ] Tower info panel appears bottom-center
- [ ] HP bars have styled colors
- [ ] Victory/defeat uses styled text

**Step 3: Fix any visual issues found**

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete visual overhaul — full JS parity"
```

---

## Implementation Order Summary

```
Phase 1: Infrastructure (Tasks 1-5) — no visual changes, foundation only
Phase 2: Map Rendering (Task 6) — first visible change
Phase 3: Entity Rendering (Tasks 7-11) — towers, enemies, projectiles, hero, warden
Phase 4: HUD System (Tasks 12-17) — top bar, build menu, info panel, wave panel, toast
Phase 5: Scene Integration (Tasks 18-20) — wire into scenes, remove DebugPrintAt
Phase 6: Testing & Polish (Tasks 21-23) — tests, visual verification
```

Each phase builds on the previous. Phase 1 is pure infrastructure with no breaking changes. Phase 2-4 can cause temporary visual regression in stage scene until Phase 5 wires everything together.

**Critical path**: Tasks 1-3 (theme + primitives) must complete before any rendering work.
