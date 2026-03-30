// draw_map.go — 地图渲染。
// 绘制渐变背景、点阵网格、路径连线（粗线+虚线）、塔槽位圆圈、入口/基地标签。
//
// The full map is cached to an offscreen image and only re-rendered when
// buildMode changes or InvalidateMapCache() is called (e.g. on map load).
// Animated elements (slot pulse) use a fixed animTime=0 in the cache;
// the visual impact is negligible and avoids per-frame re-draws (~80 calls).
package render

import (
	"image/color"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
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

	return cachedBg
}

// ---------------------------------------------------------------------------
// Map cache — full map rendered to offscreen image, invalidated on demand.
// ---------------------------------------------------------------------------

var (
	mapCache         *ebiten.Image
	mapCacheDirty    = true
	mapCacheLastBuild bool // tracks the buildMode used to render the cache
)

// InvalidateMapCache forces the map to be re-rendered on the next DrawMap call.
// Call this when the map layout changes (new map loaded, path modified, etc.).
func InvalidateMapCache() {
	mapCacheDirty = true
}

// ---------------------------------------------------------------------------
// DrawMap renders the full map: background, path, tower slots, labels.
// ---------------------------------------------------------------------------

// DrawMap 渲染地图：渐变背景 + 点阵 → 路径粗线 + 虚线 → 塔槽位 → 入口/基地标签。
// The result is cached offscreen; only re-rendered when buildMode changes or
// the cache is explicitly invalidated via InvalidateMapCache().
//
// Parameters:
//   - screen: 目标画布
//   - gm: 运行时地图
//   - fm: 字体管理器（用于绘制入口/基地标签）；可为 nil（跳过标签）
//   - animTime: 动画时间（秒），用于建塔模式空槽位脉冲动画（缓存时使用固定值 0）
//   - towerAt: 返回指定格子 (row, col) 是否有塔；可为 nil（全部视为空槽）
//   - buildMode: 是否处于建塔模式（显示高亮空槽+加号）
func DrawMap(
	screen *ebiten.Image,
	gm *gamemap.GameMap,
	fm *FontManager,
	animTime float64,
	towerAt func(row, col int) bool,
	buildMode bool,
) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	needRedraw := mapCacheDirty || mapCache == nil || buildMode != mapCacheLastBuild
	if mapCache != nil {
		cw, ch := mapCache.Bounds().Dx(), mapCache.Bounds().Dy()
		if cw != w || ch != h {
			needRedraw = true
		}
	}

	if needRedraw {
		if mapCache != nil && (mapCache.Bounds().Dx() != w || mapCache.Bounds().Dy() != h) {
			mapCache.Deallocate()
			mapCache = nil
		}
		if mapCache == nil {
			mapCache = ebiten.NewImage(w, h)
		} else {
			mapCache.Clear()
		}
		// Render with animTime=0 so the cache is static (no pulse animation).
		drawMapFull(mapCache, gm, fm, 0, towerAt, buildMode)
		mapCacheDirty = false
		mapCacheLastBuild = buildMode
	}

	screen.DrawImage(mapCache, nil)
}

// drawMapFull renders the entire map to the given target image.
func drawMapFull(
	screen *ebiten.Image,
	gm *gamemap.GameMap,
	fm *FontManager,
	animTime float64,
	towerAt func(row, col int) bool,
	buildMode bool,
) {
	// ── 1. Background gradient + dot grid ──
	bg := ensureBg()
	bg.Draw(screen, 0, 0)

	// ── 2. Path: thick rounded line + dashed center ──
	drawPaths(screen, gm)

	// ── 2.5. Terrain decorations on empty cells (deterministic, no rand) ──
	drawTerrainDecorations(screen, gm)

	// ── 3. Tower slots ──
	drawSlots(screen, gm, fm, towerAt, buildMode, animTime)

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

		// Shadow (2px below, darker, slightly thicker) for depth.
		draw.ThickLine(screen, x1, y1+2, x2, y2+2,
			theme.MapPathStrokeW+1, color.RGBA{0, 0, 0, 30})

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
// In build mode, empty slots are highlighted with an outline ring and "+" sign.
func drawSlots(screen *ebiten.Image, gm *gamemap.GameMap, fm *FontManager, towerAt func(row, col int) bool, buildMode bool, animTime float64) {
	cfg := gm.Config

	// Build mode pulse: ring alpha oscillates between 0.5 and 1.0
	var pulseAlpha uint8
	if buildMode {
		pulse := 0.5 + 0.5*math.Sin(animTime*3.0)
		pulseAlpha = uint8(128 + 127*pulse) // 128..255
	}

	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			if cfg.Grid[row][col] != config.CellBuildable {
				continue
			}

			center := gm.CellCenter(row, col)
			cx := float32(center.X)
			cy := float32(center.Y)

			occupied := towerAt != nil && towerAt(row, col)

			if occupied {
				// Occupied slot: subtle green tint
				draw.FilledCircle(screen, cx, cy, theme.MapSlotRadius, theme.SlotOccupied)
			} else if buildMode {
				// Build mode empty slot: 金黄色轮廓 + "+" 号
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius, 1.5, theme.SlotBuildRing)

				ringClr := theme.SlotBuildPulse
				ringClr.A = pulseAlpha
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius+2, 1, ringClr)

				// "+" sign
				if fm != nil {
					fm.DrawCenteredVText(screen, "+",
						float64(cx), float64(cy),
						14, theme.SlotPlusSign)
				}
			} else {
				// Normal mode empty slot: 仅轮廓线，中心完全透明
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius, 1.5, theme.SlotIdleRing)
			}
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
			draw.FilledCircle(screen, cx, cy, halfCS, clr)
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

// ---------------------------------------------------------------------------
// Terrain decorations
// ---------------------------------------------------------------------------

// drawTerrainDecorations scatters subtle decorative elements on empty cells.
// Uses a deterministic hash based on cell position — no rand, fully reproducible.
func drawTerrainDecorations(screen *ebiten.Image, gm *gamemap.GameMap) {
	cfg := gm.Config
	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			if cfg.Grid[row][col] != config.CellEmpty {
				continue
			}

			// Deterministic "random" based on position
			hash := uint32(row*7919 + col*104729)

			// Only ~30% of empty cells get decoration
			if hash%100 > 30 {
				continue
			}

			center := gm.CellCenter(row, col)
			cx := float32(center.X)
			cy := float32(center.Y)

			switch (hash / 100) % 4 {
			case 0: // small dot
				draw.FilledCircle(screen, cx, cy, 1.5, color.RGBA{60, 80, 100, 25})
			case 1: // tiny cross
				draw.Line(screen, cx-2, cy, cx+2, cy, 0.5, color.RGBA{50, 70, 90, 20}, false)
				draw.Line(screen, cx, cy-2, cx, cy+2, 0.5, color.RGBA{50, 70, 90, 20}, false)
			case 2: // faint ring
				draw.CircleOutline(screen, cx, cy, 3, 0.5, color.RGBA{40, 60, 80, 15})
			case 3: // nothing (variation)
			}
		}
	}
}
