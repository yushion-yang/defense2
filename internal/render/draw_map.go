// draw_map.go — 地图渲染。
// 绘制渐变背景、点阵网格、路径连线（粗线+虚线）、塔槽位圆圈、入口/基地标签。
package render

import (
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---------------------------------------------------------------------------
// Cached background (gradient + dot grid) — created once, reused every frame.
// ---------------------------------------------------------------------------

var cachedBg *draw.CachedGradient

// ensureBg lazily initializes the cached background gradient.
func ensureBg() *draw.CachedGradient {
	if cachedBg != nil {
		return cachedBg
	}
	cachedBg = draw.NewCachedGradient(
		theme.CanvasW, theme.CanvasH,
		theme.MapGradientTop, theme.MapGradientBot,
	)

	// Stamp dot grid overlay onto the gradient image.
	img := cachedBg.Image()
	dotClr := theme.MapDotGrid
	for y := 0; y < theme.CanvasH; y += theme.MapDotSpacing {
		for x := 0; x < theme.CanvasW; x += theme.MapDotSpacing {
			vector.DrawFilledRect(img,
				float32(x), float32(y),
				float32(theme.MapDotSize), float32(theme.MapDotSize),
				dotClr, false)
		}
	}

	return cachedBg
}

// ---------------------------------------------------------------------------
// DrawMap renders the full map: background, path, tower slots, labels.
// ---------------------------------------------------------------------------

// DrawMap 渲染地图：渐变背景 + 点阵 → 路径粗线 + 虚线 → 塔槽位 → 入口/基地标签。
//
// Parameters:
//   - screen: 目标画布
//   - gm: 运行时地图
//   - fm: 字体管理器（用于绘制入口/基地标签）；可为 nil（跳过标签）
//   - animTime: 动画时间（秒），当前未使用，预留给后续脉冲动画
//   - towerAt: 返回指定格子 (row, col) 是否有塔；可为 nil（全部视为空槽）
func DrawMap(
	screen *ebiten.Image,
	gm *gamemap.GameMap,
	fm *FontManager,
	animTime float64,
	towerAt func(row, col int) bool,
) {
	// ── 1. Background gradient + dot grid ──
	bg := ensureBg()
	bg.Draw(screen, 0, 0)

	// ── 2. Path: thick rounded line + dashed center ──
	drawPaths(screen, gm)

	// ── 3. Tower slots ──
	drawSlots(screen, gm, towerAt)

	// ── 4. Spawn / base subtle markers ──
	drawSpawnBaseMarkers(screen, gm)

	// ── 5. Labels: "入口" / "基地" ──
	if fm != nil {
		drawPathLabels(screen, gm, fm)
	}
}

// ---------------------------------------------------------------------------
// Path rendering
// ---------------------------------------------------------------------------

// drawPaths draws all path lines (multi-path and single-path).
func drawPaths(screen *ebiten.Image, gm *gamemap.GameMap) {
	if gm.MultiPath && len(gm.Paths) > 0 {
		for _, pe := range gm.Paths {
			drawWaypointPath(screen, pe.Waypoints)
		}
	} else {
		drawWaypointPath(screen, gm.Waypoints)
	}
}

// drawWaypointPath draws thick stroke + dashed center for a single waypoint sequence.
func drawWaypointPath(screen *ebiten.Image, waypoints []gamemap.Point) {
	if len(waypoints) < 2 {
		return
	}

	for i := 0; i < len(waypoints)-1; i++ {
		x1 := float32(waypoints[i].X)
		y1 := float32(waypoints[i].Y)
		x2 := float32(waypoints[i+1].X)
		y2 := float32(waypoints[i+1].Y)

		// Thick rounded base stroke.
		draw.ThickLine(screen, x1, y1, x2, y2,
			theme.MapPathStrokeW, theme.MapPathStroke)

		// Center dashed line.
		draw.DashedLine(screen, x1, y1, x2, y2,
			theme.MapPathDashW,
			theme.MapPathDashOn, theme.MapPathDashOff,
			theme.MapPathDash)
	}
}

// ---------------------------------------------------------------------------
// Tower slots
// ---------------------------------------------------------------------------

// drawSlots draws semi-transparent circles for each buildable cell.
func drawSlots(screen *ebiten.Image, gm *gamemap.GameMap, towerAt func(row, col int) bool) {
	cfg := gm.Config

	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			if cfg.Grid[row][col] != config.CellBuildable {
				continue
			}

			center := gm.CellCenter(row, col)
			cx := float32(center.X)
			cy := float32(center.Y)

			clr := theme.SlotEmpty
			if towerAt != nil && towerAt(row, col) {
				clr = theme.SlotOccupied
			}

			vector.DrawFilledCircle(screen, cx, cy, theme.MapSlotRadius, clr, true)
		}
	}
}

// ---------------------------------------------------------------------------
// Spawn / Base subtle markers
// ---------------------------------------------------------------------------

// drawSpawnBaseMarkers draws very subtle colored circles at spawn and base cells.
func drawSpawnBaseMarkers(screen *ebiten.Image, gm *gamemap.GameMap) {
	cfg := gm.Config

	// Subtle spawn (red) and base (blue) markers with low alpha.
	spawnClr := color.RGBA{R: 180, G: 60, B: 60, A: 30}
	baseClr := color.RGBA{R: 60, G: 60, B: 180, A: 30}

	cs := float64(gm.CellSize)
	halfCS := float32(cs / 2)

	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			ct := cfg.Grid[row][col]
			if ct != config.CellSpawn && ct != config.CellBase {
				continue
			}

			center := gm.CellCenter(row, col)
			cx := float32(center.X)
			cy := float32(center.Y)

			clr := spawnClr
			if ct == config.CellBase {
				clr = baseClr
			}
			vector.DrawFilledCircle(screen, cx, cy, halfCS, clr, true)
		}
	}
}

// ---------------------------------------------------------------------------
// Path labels
// ---------------------------------------------------------------------------

// drawPathLabels draws "入口" at the first waypoint and "基地" at the last.
func drawPathLabels(screen *ebiten.Image, gm *gamemap.GameMap, fm *FontManager) {
	waypoints := gm.Waypoints
	if gm.MultiPath && len(gm.Paths) > 0 {
		// For multi-path, label first entry's start and last entry's end.
		waypoints = gm.Paths[0].Waypoints
	}
	if len(waypoints) < 2 {
		return
	}

	labelSize := float64(theme.FontMapLabel)
	clr := theme.MapPathLabel

	// "入口" at first waypoint (centered above).
	first := waypoints[0]
	fm.DrawCenteredText(screen, "\u5165\u53e3", first.X, first.Y-float64(theme.MapSlotRadius)-labelSize, labelSize, clr)

	// "基地" at last waypoint (centered above).
	last := waypoints[len(waypoints)-1]
	fm.DrawCenteredText(screen, "\u57fa\u5730", last.X, last.Y-float64(theme.MapSlotRadius)-labelSize, labelSize, clr)
}
