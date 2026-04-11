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

var (
	cachedBg    *draw.CachedGradient
	cachedBgW   int
	cachedBgH   int
	cachedBgTop color.RGBA
	cachedBgBot color.RGBA
)

// ensureBg lazily initializes the cached background gradient sized to cover
// the full map (which may be larger than one screen).
func ensureBg(w, h int, top, bot color.RGBA) *draw.CachedGradient {
	if cachedBg != nil && cachedBgW == w && cachedBgH == h &&
		cachedBgTop == top && cachedBgBot == bot {
		return cachedBg
	}
	cachedBg = draw.NewCachedGradient(w, h, top, bot)
	cachedBgW = w
	cachedBgH = h
	cachedBgTop = top
	cachedBgBot = bot
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
	mt := theme.MapThemeFor(gm.Theme)

	// ── 1. Background gradient covering full map ──
	mapW := gm.Config.Cols * gm.CellSize
	mapH := gm.Config.Rows * gm.CellSize
	if mapW < theme.CanvasW {
		mapW = theme.CanvasW
	}
	if mapH < theme.CanvasH {
		mapH = theme.CanvasH
	}
	bg := ensureBg(mapW, mapH, mt.GradientTop, mt.GradientBot)
	bg.Draw(screen, 0, 0)

	// ── 2. Path: thick rounded line + dashed center ──
	drawPaths(screen, gm, mt.PathColor)

	// ── 2.5. Terrain decorations on empty cells (deterministic, no rand) ──
	drawTerrainDecorations(screen, gm, mt.DotColor)

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
func drawPaths(screen *ebiten.Image, gm *gamemap.GameMap, pathClr color.RGBA) {
	if gm.MultiPath && len(gm.Paths) > 0 {
		for _, pe := range gm.Paths {
			drawWaypointPath(screen, pe.Waypoints, pathClr)
		}
	} else {
		drawWaypointPath(screen, gm.Waypoints, pathClr)
	}
}

// drawWaypointPath draws thick stroke + dashed center for a single waypoint sequence.
func drawWaypointPath(screen *ebiten.Image, waypoints []gamemap.Point, pathClr color.RGBA) {
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
			theme.MapPathStrokeW, pathClr)

		// Center dashed line (slightly muted variant of path color).
		dashClr := color.RGBA{
			R: uint8(float64(pathClr.R) * 0.7),
			G: uint8(float64(pathClr.G) * 0.7),
			B: uint8(float64(pathClr.B) * 0.7),
			A: pathClr.A,
		}
		draw.DashedLine(screen, x1, y1, x2, y2,
			theme.MapPathDashW,
			theme.MapPathDashOn, theme.MapPathDashOff,
			dashClr)
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
				// 已占用格子不绘制底色，塔精灵会覆盖
			} else if buildMode {
				// Build mode empty slot: 凹陷效果 + 金黄色轮廓 + "+" 号
				draw.FilledCircle(screen, cx, cy, theme.MapSlotRadius, color.RGBA{R: 10, G: 15, B: 30, A: 35})
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius-2, 1, color.RGBA{R: 5, G: 10, B: 20, A: 25})
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
				// Normal mode empty slot: 凹陷效果 + 轮廓线
				draw.FilledCircle(screen, cx, cy, theme.MapSlotRadius, color.RGBA{R: 10, G: 15, B: 30, A: 25})
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius-2, 1, color.RGBA{R: 5, G: 10, B: 20, A: 18})
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
func drawTerrainDecorations(screen *ebiten.Image, gm *gamemap.GameMap, dotClr color.RGBA) {
	cfg := gm.Config

	// Derive decoration colors from the theme dot color.
	decoA := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: 25}
	decoB := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: 20}
	decoC := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: 15}

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
				draw.FilledCircle(screen, cx, cy, 1.5, decoA)
			case 1: // tiny cross
				draw.Line(screen, cx-2, cy, cx+2, cy, 0.5, decoB, false)
				draw.Line(screen, cx, cy-2, cx, cy+2, 0.5, decoB, false)
			case 2: // faint ring
				draw.CircleOutline(screen, cx, cy, 3, 0.5, decoC)
			case 3: // nothing (variation)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Parallax background — slow-drifting star field drawn every frame.
// ---------------------------------------------------------------------------

var (
	parallaxStars  [][3]float64
	parallaxW      float64
	parallaxH      float64
)

func initParallaxStars(w, h float64) {
	if len(parallaxStars) > 0 && parallaxW == w && parallaxH == h {
		return
	}
	parallaxW = w
	parallaxH = h
	const count = 40
	parallaxStars = make([][3]float64, count)
	for i := 0; i < count; i++ {
		fi := float64(i)
		parallaxStars[i] = [3]float64{
			math.Mod(fi*0.618033988*w, w),
			math.Mod((fi*0.381966+0.2)*h, h),
			0.5 + math.Mod(fi*0.7, 1.5),
		}
	}
}

// DrawParallaxBG renders slow-moving background stars. Called every frame (not cached).
// worldW/worldH are the logical pixel dimensions of the full map.
func DrawParallaxBG(screen *ebiten.Image, animTime, worldW, worldH float64) {
	if worldW <= 0 {
		worldW = float64(theme.CanvasW)
	}
	if worldH <= 0 {
		worldH = float64(theme.CanvasH)
	}
	initParallaxStars(worldW, worldH)
	for i, s := range parallaxStars {
		speed := 2.0 + float64(i%3)*1.5
		x := math.Mod(s[0]+animTime*speed, worldW)
		y := s[1]
		alpha := uint8(15 + (i%4)*5)
		r := float32(s[2])
		draw.FilledCircle(screen, float32(x), float32(y), r,
			color.RGBA{R: 180, G: 200, B: 255, A: alpha})
	}
}
