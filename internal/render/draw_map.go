// draw_map.go — 地图渲染模块。
//
// 负责绘制完整地图：渐变背景 → 路径连线 → 地形装饰 → 塔槽位 → 出生/基地标记 → 标签。
// 全部内容缓存到离屏图像（mapCache），每帧仅需一次 DrawImage 即可复用。
//
// 缓存失效条件（重新绘制 ~80 次 draw call）：
//  1. mapCacheDirty 标记（地图加载/切换时由 InvalidateMapCache() 设置）
//  2. buildMode 状态变化（建塔模式切换时槽位样式不同）
//  3. 画布尺寸变化（窗口缩放）
//
// 设计取舍：缓存使用固定 animTime=0，放弃了建塔模式空槽位的脉冲动画。
// 此动画仅是微弱的 alpha 波动，肉眼几乎不可见，换取了每帧零重绘的巨大收益。
//
// 另有 DrawParallaxBG() 是每帧渲染的缓慢漂移星空，不受缓存影响。
package render

import (
	"image/color"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"
	"defense2/internal/i18n"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// 渐变背景缓存 — 首次创建后每帧复用，尺寸或颜色变化时重建。
// ---------------------------------------------------------------------------

var (
	cachedBg    *draw.CachedGradient // 渐变离屏图像
	cachedBgW   int                  // 缓存宽度（检测尺寸变化）
	cachedBgH   int                  // 缓存高度
	cachedBgTop color.RGBA           // 缓存顶部颜色（检测主题变化）
	cachedBgBot color.RGBA           // 缓存底部颜色
)

// ensureBg 惰性初始化渐变背景，尺寸覆盖整个地图（可能大于单屏）。
// 仅在尺寸或颜色发生变化时重建，否则直接返回缓存。
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
// 地图离屏缓存 — 完整地图渲染到离屏图像，按需失效重绘。
// ---------------------------------------------------------------------------

var (
	mapCache          *ebiten.Image // 离屏缓存图像（与 screen 同尺寸）
	mapCacheDirty     = true        // 脏标记，true 时下一帧重绘
	mapCacheLastBuild bool          // 上次缓存时的 buildMode，用于检测模式切换
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

// drawMapFull 渲染完整地图到指定的离屏图像。
// 绘制顺序（从下到上）：
//  1. 渐变背景（覆盖整个地图区域，尺寸取 max(地图, 画布)）
//  2. 路径粗线 + 虚线中心线（阴影→底线→虚线三层）
//  3. 地形装饰（空格子上的确定性散点：小点/十字/圆环）
//  4. 塔槽位（凹陷圆 + 建塔模式高亮）
//  5. 出生/基地微弱标记（低 alpha 红/蓝圆）
//  6. "入口"/"基地" 文字标签
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

// drawWaypointPath 绘制单条路径序列的三层视觉效果：
//  1. 阴影层（下偏 2px、加宽 1px、深色）— 提供立体深度感
//  2. 底色粗线（主题路径色）— 路径主体
//  3. 中心虚线（路径色微暗变体）— 增加视觉层次
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
			theme.MapPathStrokeW+1, theme.MapPathShadow)

		// Thick rounded base stroke.
		draw.ThickLine(screen, x1, y1, x2, y2,
			theme.MapPathStrokeW, pathClr)

		// Center dashed line (slightly muted variant of path color).
		dashClr := color.RGBA{
			R: uint8(float64(pathClr.R) * theme.MapPathDashDarken),
			G: uint8(float64(pathClr.G) * theme.MapPathDashDarken),
			B: uint8(float64(pathClr.B) * theme.MapPathDashDarken),
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

// drawSlots 绘制每个可建造格子的塔槽位。
// 三种状态：
//   - 已占用（occupied）：不绘制底色，塔精灵自行覆盖
//   - 建塔模式空槽：凹陷底色 + 内圈 + 金色轮廓 + 脉冲环 + "+" 号
//   - 常规模式空槽：凹陷底色 + 内圈 + 暗色轮廓
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
				draw.FilledCircle(screen, cx, cy, theme.MapSlotRadius, theme.SlotDeprBuild)
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius-2, 1, theme.SlotDeprBuildInner)
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
				draw.FilledCircle(screen, cx, cy, theme.MapSlotRadius, theme.SlotDeprIdle)
				draw.CircleOutline(screen, cx, cy, theme.MapSlotRadius-2, 1, theme.SlotDeprIdleInner)
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
	spawnClr := theme.MapSpawnMarker
	baseClr := theme.MapBaseMarker

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
	fm.DrawCenteredText(screen, i18n.T("game.map.entrance"), first.X, first.Y-float64(theme.MapSlotRadius)-labelSize, labelSize, clr)

	// "基地" at last waypoint (centered above).
	last := waypoints[len(waypoints)-1]
	fm.DrawCenteredText(screen, i18n.T("game.map.base"), last.X, last.Y-float64(theme.MapSlotRadius)-labelSize, labelSize, clr)
}

// ---------------------------------------------------------------------------
// Terrain decorations
// ---------------------------------------------------------------------------

// drawTerrainDecorations 在空格子上散布微弱的装饰元素。
// 使用基于行列位置的确定性哈希（row*7919 + col*104729），无随机数，完全可重现。
// ~30% 的空格子获得装饰，4 种变体：小圆点 / 十字 / 圆环 / 空（自然留白）。
// 装饰 alpha 极低（theme.MapDecoAlpha*），仅在仔细观察时可见，避免喧宾夺主。
func drawTerrainDecorations(screen *ebiten.Image, gm *gamemap.GameMap, dotClr color.RGBA) {
	cfg := gm.Config

	// Derive decoration colors from the theme dot color.
	decoA := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: theme.MapDecoAlphaA}
	decoB := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: theme.MapDecoAlphaB}
	decoC := color.RGBA{R: dotClr.R, G: dotClr.G, B: dotClr.B, A: theme.MapDecoAlphaC}

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
// 视差星空背景 — 每帧绘制的缓慢漂移星点（不缓存，因为位置持续变化）。
// ---------------------------------------------------------------------------

var (
	parallaxStars [][3]float64 // [x, y, radius] 40 颗星的初始位置和大小
	parallaxW     float64      // 缓存的地图宽度（尺寸变化时重新生成星点位置）
	parallaxH     float64      // 缓存的地图高度
)

// initParallaxStars 确定性生成 40 颗星的初始位置（使用黄金比例分布避免聚集）。
// 不使用 math/rand，利用黄金比例 0.618... 的低差异序列产生均匀分布。
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

// DrawParallaxBG 渲染缓慢移动的星空背景，每帧调用（不可缓存因为位置在变化）。
// 每颗星沿 X 轴以不同速度漂移（2~5 px/s），到达右边界后环绕回左侧。
// alpha 极低（15~35），仅营造微弱的空间感。
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
			color.RGBA{R: theme.MapParallaxStar.R, G: theme.MapParallaxStar.G, B: theme.MapParallaxStar.B, A: alpha})
	}
}
