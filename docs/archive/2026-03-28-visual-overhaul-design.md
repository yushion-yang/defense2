# Visual Overhaul Design — Full JS Parity

Date: 2026-03-28
Status: Approved

## Goal

Faithfully replicate the JS original's visual style (dark-blue theme, polished HUD,
detailed panels, animations) in the Go/Ebitengine port.

Reference: `/Users/yushion/Games/tower-defense` (JS original)

---

## 1. Infrastructure Layer

### 1.1 Design System (`internal/render/theme/`)

Centralized visual constants extracted from JS `uiConstants.js`:

**colors.go** — All colors as `color.RGBA`:
- Canvas bg: `#0f172a` / Panel bg: `rgba(15,23,42,0.88)`
- Text: title `#e2e8f0`, body `#cbd5e1`, muted `#94a3b8`, locked `#4b5563`
- Accent: btnPrimary `#3b82f6`, btnDanger `#ef4444`, gold `#fbbf24`, hearts `#f87171`
- Status: strUp `#4ade80`, strDown `#f87171`, skill `#60a5fa`
- Faction: base `#38bdf8`, output `#f87171`, control `#60a5fa`, support `#4ade80`

**layout.go** — from JS `LAYOUT`:
- Canvas: 1200x540
- TopBar: y=6, h=40, w=760, dividerOffset=290, btnH=30, btnGap=6
- Badge: w=200, h=24, gap=4
- InfoPanel: x=14, w=220
- CenterPanel: w=500 (x=350)
- Bottom margin: 14

**ds.go** — Design System tokens:
- Spacing: xs=4, sm=8, md=12, lg=16
- Font sizes: xs=10, sm=11, md=12, lg=14, xl=16
- Line heights: ability=16, attr=18, title=22, btn=36
- Panel: w=500, radius=14, minH=100, innerPad=14
- Button: h=30, radius=12, gap=8
- Tooltip: buildW=260, hoverW=280, lineH=16, pad=10, radius=10

### 1.2 Drawing Primitives (`internal/render/draw/`)

Ebitengine lacks Canvas2D-style APIs. Implement:
- `roundrect.go` — Rounded rectangle (fill + stroke)
- `gradient.go` — Linear gradient (row-by-row pixel fill)
- `dashed.go`   — Dashed line rendering (segment + gap)
- `circle.go`   — Enhanced circle (fill, outline, arc, glow)

### 1.3 FontManager Global Singleton

Current: only used in SelectScene. Change to global singleton shared by all scenes.
Replace ALL `ebitenutil.DebugPrintAt` calls with `FontManager.DrawText`.

---

## 2. Map Rendering

### Background
- Linear gradient: `#193549` (top) to `#1b4332` (bottom)
- Dot grid overlay: `rgba(255,255,255,0.03)`, 2x2 dots, 40px spacing

### Path
- 52px wide rounded stroke, color `#d6d3d1`, round cap/join
- Center dashed line: `#9ca3af`, width 4, dash [10,10]
- Labels: "入口" at start, "基地" at end, `rgba(255,255,255,0.18)` bold 16px

### Tower Slots
- Empty: r=26 circle, fill `rgba(255,255,255,0.08)`
- Build mode: pulsing border `rgba(251,191,36, sin(t*3))`, width 2
- Occupied: fill `rgba(34,197,94,0.14)`
- Starter hints: blue pulse `rgba(96,165,250,pulse)` + "+" `#fde68a` + "荐" `#bfdbfe`

### Grid
- Remove per-cell rectangle fill (except spawn/base markers)

---

## 3. Entity Rendering

### 3.1 Towers
- Sprite size: 64 (TOWER_VISUAL.baseSize)
- Selection ring: `rgba(253,224,71,0.72)`, width 2, r=24
- Range: fill `rgba(245,158,11,0.08)`, stroke `rgba(245,158,11,0.35)`, width 1.5
- Fallback: r=18, selected `#f59e0b`, default `#38bdf8`, barrel `#082f49` 8x14
- Name label: 10px `rgba(255,255,255,0.85)`, 36px below center
- Buff dots: 26px above, spacing 8, r=3 (damage=#ffd700, atkSpd=#60a5fa, range=#4ade80, crit=#c084fc, str=#22d3ee)
- Aura pulse: type-based colors with sin(t) animation
- Charge visual: red energy ring `rgba(239,68,68,...)`

### 3.2 Enemies
- Sprite scale: 3.15
- HP bar: normal 36x5 (y-26), boss 48x7 (y-30)
  - Border `#0f172a`, bg `#1e293b`, trail `#fb923c` (lerp)
  - Fill: >60% `#ef4444`, >30% `#dc2626`, <=30% `#991b1b`
  - Boss segments: 20% lines `rgba(0,0,0,0.4)`
- Shield bar: above HP, h=3, white
- Boss aura: dual pulsing rings (inner `rgba(244,63,94,...)`, outer `rgba(251,113,133,...)`)
- Runner pulse: `rgba(251,146,60,0.45)`
- Swarm: yellow jittering triangles `rgba(254,240,138,0.72)`
- Tank: white 14x14 square overlay
- Flying: ground shadow ellipse `#0f172a` alpha 0.2
- Status effects: slow=blue ring, stun=pink circle, poison/bleed/silence dots

### 3.3 Projectiles
- Default: r=4 + glow r=8, `#fde68a`
- Sniper: r=5 + glow r=10, `#fb923c`
- Rapid: `#5eead4`
- Freeze crystal: `#a5f3fc` diamond
- Wind blade: `#a3e635` crescent + `#86efac` trail
- Beam: glow width 3.2x, highlight 0.34 ratio

### 3.4 Hero
- Leash: selected `rgba(192,132,252,0.52)` dash [8,8], default `rgba(125,211,252,0.18)`
- Fallback: body `#4c1d95`, stroke `#c4b5fd`
- XP bar: bg `rgba(255,255,255,0.08)`, fill `#a78bfa`, pill shape

### 3.5 Wardens
- Wire up existing PNG sprites (fix embed directive in data.go)
- Core mech: r=76, radial glow `rgba(59,130,246,...)`
- Entity fallbacks: prince=diamond, envoy=hexagram, generic=blue circle
- Exhaust/orbit particles

---

## 4. HUD System

### 4.1 Top Bar
- Centered pill: 760x40, y=6, radius=16, bg `rgba(15,23,42,0.76)`, border `rgba(255,255,255,0.08)`
- Left: hearts `#f87171` + gold `#fbbf24` + waves `#93c5fd`, bold 17px
- Divider: `rgba(255,255,255,0.07)` at x=290
- Right: button group (build/start/speed/menu), h=30, radius=14, gap=6
  - Tones: primary=green, accent=blue, secondary=dark, disabled=dim

### 4.2 Toast
- 500x34, centered, bg `rgba(15,23,42,0.88)`, radius=10, 300ms fade

### 4.3 Pause Overlay
- `rgba(2,6,23,0.42)` full screen
- Menu panel `rgba(15,23,42,0.92)`, radius=18

### 4.4 Victory/Defeat
- Overlay `rgba(0,0,0,0.4)`
- Text bold 52px (victory `#22c55e` / defeat `#ef4444`)

---

## 5. Panel System

### 5.1 Tower Info Panel (bottom-center)
- W=500, x=350, y=540-H-14, radius=14
- Bg `rgba(15,23,42,0.88)`, border `rgba(148,163,184,0.2)`
- Sections: title (bold 14) / attrs 3-col / abilities / buffs (max 6) / skill / buttons
- Sell button: `#ef4444`, h=34, radius=12

### 5.2 Wave Preview (left side)
- X=14, w=220, radius=16, bg `rgba(15,23,42,0.72)`
- Threat colors: danger `#fca5a5`, pressure `#fde68a`, calm `#86efac`

### 5.3 Build Menu
- Panel: `rgba(15,23,42,0.88)`, radius=18
- Cards: 88x56, gap=6, radius=10
  - Normal bg `rgba(255,255,255,0.05)`, selected bg `rgba(34,197,94,0.22)` + border `rgba(74,222,128,0.66)`
  - Name bold 12px white, cost `#fbbf24` 11px
- Faction tabs: h=20, gap=3, radius=6, colored by faction
- Tooltip: 260px, bg `rgba(15,23,42,0.92)`, border `rgba(79,140,255,0.4)`, radius=10

### 5.4 Hero Panel
- Bg `rgba(15,23,42,0.86)`, radius=16, border `rgba(196,181,253,0.16)`
- XP bar: bg `rgba(255,255,255,0.08)`, fill `#a78bfa`, pill
- Skill icon: 34x16, radius=6

### 5.5 Warden Select
- Overlay `rgba(5,10,20,0.92)`
- Title `#e2e8f0` bold 24px, subtitle `#64748b` 13px
- Cards 200x42, selected border `#3b82f6`
- Confirm/Skip buttons 180x38

---

## 6. Scenes

### Select Scene
- Minor tuning: gradient `#0b1220` to `#172554`, cards 180x120 radius=14, btn `#22c55e`

### Result Scene
- Full rewrite: bg `#0f172a`, stats panel 500x220 `rgba(255,255,255,0.04)` radius=16
- Use FontManager for all text

---

## 7. Animation System

Global `animTime float64` accumulator (+=dt each frame).
- Pulse: `sin(animTime * speed) * 0.5 + 0.5`
- HP trail: lerp 0.05/frame
- Toast fade: 300ms linear
- Slot pulse: speed ~3-4 rad/s
- Aura pulse: speed ~2 rad/s
- Boss rings: speed ~2-3 rad/s with phase offsets

---

## File Change Summary

| Action | File | Description |
|--------|------|-------------|
| NEW | `internal/render/theme/colors.go` | Global color constants |
| NEW | `internal/render/theme/layout.go` | Layout constants |
| NEW | `internal/render/theme/ds.go` | Design System tokens |
| NEW | `internal/render/draw/roundrect.go` | Rounded rectangle |
| NEW | `internal/render/draw/gradient.go` | Linear gradient |
| NEW | `internal/render/draw/dashed.go` | Dashed lines |
| NEW | `internal/render/draw/circle.go` | Enhanced circles |
| REWRITE | `internal/render/draw_map.go` | Gradient bg + thick path + slot circles |
| REWRITE | `internal/render/draw_tower.go` | Selection ring + buffs + aura + charge |
| REWRITE | `internal/render/draw_enemy.go` | HP bars + boss aura + status effects |
| REWRITE | `internal/render/draw_projectile.go` | Per-type rendering + glow |
| REWRITE | `internal/render/draw_hero.go` | Dashed leash + XP pill |
| REWRITE | `internal/render/draw_warden.go` | Wire PNG sprites + effects |
| REWRITE | `internal/render/hud/top_bar.go` | Pill bar + icons + buttons |
| REWRITE | `internal/render/hud/build_menu.go` | Card-style + faction tabs |
| REWRITE | `internal/render/hud/info_panel.go` | Bottom-center detail panel |
| NEW | `internal/render/hud/wave_panel.go` | Wave preview panel |
| NEW | `internal/render/hud/toast.go` | Toast messages |
| MODIFY | `internal/render/font.go` | Global singleton |
| MODIFY | `internal/scene/stage.go` | Wire new renderers + animTime |
| MODIFY | `internal/scene/select.go` | Tune to match JS params |
| REWRITE | `internal/scene/result.go` | Dark blue theme + FontManager |
| MODIFY | `data.go` | Add warden PNG embed |

~22 files: 8 new, 11 rewrite, 3 modify.
