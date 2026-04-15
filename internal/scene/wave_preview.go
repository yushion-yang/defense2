// wave_preview.go — Wave Preview scene.
// Standalone scene for previewing enemy compositions of all waves per map.
// Accessible from the TestSelect scene via the "wave-preview" scenario entry.
package scene

import (
	"cmp"
	"image/color"
	"slices"
	"strconv"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Layout constants ────────────────────────────────

const (
	wpMapPanelW  = 200.0 // left map list panel width
	wpCtrlH      = 50.0  // bottom control bar height
	wpMapItemH   = 28.0  // map item height
	wpPadX       = 12.0  // horizontal padding
	wpPadY       = 8.0   // vertical padding
	wpScrollStep = 40.0  // pixels per scroll wheel tick
	wpWaveRowH   = 22.0  // wave row height in right panel
	wpHeaderH    = 40.0  // right panel header height
)

// ── WavePreviewScene ────────────────────────────────

// WavePreviewScene previews enemy compositions for all waves in each map.
type WavePreviewScene struct {
	switcher Switcher

	// Map data
	maps       []config.LevelEntry
	selectedID int // index into maps
	hoverMap   int // hover index in map list

	// Archetypes for label lookup
	archetypes map[string]*enemy.SpawnConfig

	// Cached wave data for the selected map
	waves []wavePreviewRow

	// Scroll
	mapScrollY    float64
	mapMaxScroll  float64
	waveScrollY   float64
	waveMaxScroll float64
}

// wavePreviewRow cached preview data for one wave.
type wavePreviewRow struct {
	Wave    int
	Entries []enemy.WavePreviewEntry
	Total   int
	IsBoss  bool
}

// NewWavePreviewScene creates the wave preview scene.
func NewWavePreviewScene(sw Switcher) *WavePreviewScene {
	s := &WavePreviewScene{
		switcher:   sw,
		selectedID: 0,
		hoverMap:   -1,
	}
	s.loadMaps()
	s.loadArchetypes()
	s.computeWaves()
	return s
}

func (s *WavePreviewScene) loadMaps() {
	levels, err := config.LoadLevelList("")
	if err != nil || len(levels) == 0 {
		return
	}
	s.maps = levels
}

func (s *WavePreviewScene) loadArchetypes() {
	archs, err := config.LoadEnemyArchetypes()
	if err != nil || len(archs) == 0 {
		return
	}
	// Convert config.EnemyArchetype to enemy.SpawnConfig for label lookup.
	result := make(map[string]*enemy.SpawnConfig, len(archs))
	for id, a := range archs {
		result[id] = &enemy.SpawnConfig{
			Label: a.Label,
		}
	}
	s.archetypes = result
}

func (s *WavePreviewScene) computeWaves() {
	s.waves = nil
	if len(s.maps) == 0 || s.selectedID < 0 || s.selectedID >= len(s.maps) {
		return
	}
	m := s.maps[s.selectedID]
	sc := config.GlobalSpawnerConfig()

	for w := 1; w <= m.Waves; w++ {
		entries, total, isBoss := enemy.PreviewWave(w, m.Waves, sc.Scaling.EnemiesPerWave, s.archetypes)
		// Sort entries by weight (count) descending for readability.
		slices.SortFunc(entries, func(a, b enemy.WavePreviewEntry) int {
			return cmp.Compare(b.Count, a.Count)
		})
		s.waves = append(s.waves, wavePreviewRow{
			Wave:    w,
			Entries: entries,
			Total:   total,
			IsBoss:  isBoss,
		})
	}
	s.waveScrollY = 0
}

// ── Update ──────────────────────────────────────────

func (s *WavePreviewScene) Update() error {
	// Keyboard shortcuts.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		s.navigateMap(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		s.navigateMap(1)
	}

	// Mouse scroll.
	mx, _ := draw.CursorPos()
	_, wy := ebiten.Wheel()
	if wy != 0 {
		if mx < wpMapPanelW {
			s.mapScrollY -= wy * wpScrollStep
			s.clampMapScroll()
		} else {
			s.waveScrollY -= wy * wpScrollStep
			s.clampWaveScroll()
		}
	}

	// Hover (desktop=mouse, touch=long-press).
	if hx, hy, hov := draw.HoverPos(); hov {
		s.hoverMap = s.hitTestMapList(hx, hy)
	} else {
		s.hoverMap = -1
	}

	if isTapJustPressed() {
		mxf, myf := draw.CursorPos()
		s.handleClick(mxf, myf)
	}

	return nil
}

func (s *WavePreviewScene) navigateMap(delta int) {
	if len(s.maps) == 0 {
		return
	}
	s.selectedID += delta
	if s.selectedID < 0 {
		s.selectedID = len(s.maps) - 1
	} else if s.selectedID >= len(s.maps) {
		s.selectedID = 0
	}
	s.computeWaves()
}

func (s *WavePreviewScene) handleClick(mx, my float64) {
	sh := float64(game.ScreenHeight)

	// Bottom control bar.
	if my >= sh-wpCtrlH {
		s.handleControlClick(mx)
		return
	}

	// Left panel — map selection.
	if mx <= wpMapPanelW {
		idx := s.hitTestMapList(mx, my)
		if idx >= 0 && idx != s.selectedID {
			s.selectedID = idx
			s.computeWaves()
			playUIClick(s.switcher)
		}
		return
	}
}

func (s *WavePreviewScene) handleControlClick(mx float64) {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	btnW := 72.0
	btnH := 30.0
	baseY := sh - wpCtrlH + (wpCtrlH-btnH)/2

	bx := (sw - btnW) / 2
	if mx >= bx && mx <= bx+btnW {
		_, cmy := draw.CursorPos()
		if cmy >= baseY && cmy <= baseY+btnH {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		}
	}
}

func (s *WavePreviewScene) hitTestMapList(mx, my float64) int {
	if mx < 0 || mx > wpMapPanelW {
		return -1
	}
	sh := float64(game.ScreenHeight)
	if my > sh-wpCtrlH || my < wpPadY {
		return -1
	}

	y := wpPadY - s.mapScrollY
	for i := range s.maps {
		if my >= y && my < y+wpMapItemH {
			return i
		}
		y += wpMapItemH
	}
	return -1
}

func (s *WavePreviewScene) clampMapScroll() {
	if s.mapScrollY < 0 {
		s.mapScrollY = 0
	}
	if s.mapScrollY > s.mapMaxScroll {
		s.mapScrollY = s.mapMaxScroll
	}
}

func (s *WavePreviewScene) clampWaveScroll() {
	if s.waveScrollY < 0 {
		s.waveScrollY = 0
	}
	if s.waveScrollY > s.waveMaxScroll {
		s.waveScrollY = s.waveMaxScroll
	}
}

// ── Draw ────────────────────────────────────────────

func (s *WavePreviewScene) Draw(screen *ebiten.Image) {
	fm := render.GlobalFont()
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// Full background.
	draw.FilledRect(screen, 0, 0, float32(sw), float32(sh),
		color.RGBA{R: 12, G: 15, B: 22, A: 255}, false)

	// Right panel background (slightly lighter).
	draw.FilledRect(screen, float32(wpMapPanelW), 0,
		float32(sw-wpMapPanelW), float32(sh-wpCtrlH),
		color.RGBA{R: 15, G: 18, B: 25, A: 255}, false)

	// Right panel — wave table.
	s.drawWaveTable(screen, fm, sw, sh)

	// Left panel background.
	draw.FilledRect(screen, 0, 0, float32(wpMapPanelW), float32(sh-wpCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Right border of map panel.
	draw.FilledRect(screen, float32(wpMapPanelW)-1, 0, 1, float32(sh-wpCtrlH),
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	// Left panel — map list.
	s.drawMapList(screen, fm, sh)

	// Bottom control bar.
	draw.FilledRect(screen, 0, float32(sh-wpCtrlH), float32(sw), float32(wpCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Top border.
	draw.FilledRect(screen, 0, float32(sh-wpCtrlH), float32(sw), 1,
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	s.drawControls(screen, fm, sw, sh)
}

func (s *WavePreviewScene) drawMapList(screen *ebiten.Image, fm *render.FontManager, sh float64) {
	if fm == nil {
		return
	}

	panelBottom := sh - wpCtrlH
	y := wpPadY - s.mapScrollY

	for i, m := range s.maps {
		if y+wpMapItemH > 0 && y < panelBottom {
			selected := i == s.selectedID
			hovered := i == s.hoverMap

			// Background highlight.
			if selected {
				draw.FilledRect(screen, float32(wpPadX-4), float32(y),
					float32(wpMapPanelW-2*wpPadX+8), float32(wpMapItemH),
					color.RGBA{R: 60, G: 80, B: 140, A: 160}, false)
			} else if hovered {
				draw.FilledRect(screen, float32(wpPadX-4), float32(y),
					float32(wpMapPanelW-2*wpPadX+8), float32(wpMapItemH),
					color.RGBA{R: 40, G: 50, B: 80, A: 120}, false)
			}

			// Map name.
			textClr := theme.TextBody
			if selected {
				textClr = color.RGBA{R: 200, G: 220, B: 255, A: 255}
			}

			label := m.Name
			if label == "" {
				label = m.ID
			}
			wavesStr := " (" + strconv.Itoa(m.Waves) + "波)"
			fm.DrawText(screen, label+wavesStr, wpPadX+4, y+6, 11, textClr)

			// Selected indicator dot.
			if selected {
				draw.FilledCircle(screen, float32(wpPadX), float32(y+wpMapItemH/2),
					2, color.RGBA{R: 100, G: 160, B: 255, A: 255})
			}
		}
		y += wpMapItemH
	}

	// Compute max scroll.
	totalH := y + s.mapScrollY
	s.mapMaxScroll = totalH - panelBottom
	if s.mapMaxScroll < 0 {
		s.mapMaxScroll = 0
	}
}

func (s *WavePreviewScene) drawWaveTable(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	if fm == nil {
		return
	}

	panelLeft := wpMapPanelW + wpPadX
	panelTop := wpPadY
	panelBottom := sh - wpCtrlH

	if len(s.maps) == 0 {
		fm.DrawCenteredText(screen, "无地图数据", (wpMapPanelW+sw)/2, sh/2, 14, theme.TextMuted)
		return
	}

	m := s.maps[s.selectedID]

	// Header: map name + stats.
	headerLabel := m.Name
	if headerLabel == "" {
		headerLabel = m.ID
	}
	headerLabel += "  —  " + strconv.Itoa(m.Waves) + " 波"
	if m.Difficulty != "" {
		headerLabel += "  [" + m.Difficulty + "]"
	}
	fm.DrawBoldText(screen, headerLabel, panelLeft, panelTop+4, 16, theme.TextTitle)

	// Subheader.
	fm.DrawText(screen, "Up/Down: 切换地图  Scroll: 滚动波次  Esc: 返回",
		panelLeft, panelTop+26, 10, theme.TextLocked)

	if len(s.waves) == 0 {
		fm.DrawCenteredText(screen, "该地图无波次数据", (wpMapPanelW+sw)/2, sh/2, 14, theme.TextMuted)
		return
	}

	// Wave rows.
	y := panelTop + wpHeaderH - s.waveScrollY
	maxTextW := sw - wpMapPanelW - 2*wpPadX

	for _, row := range s.waves {
		if y+wpWaveRowH > panelTop+wpHeaderH && y < panelBottom {
			// Wave number.
			waveLabel := "第" + strconv.Itoa(row.Wave) + "波"
			if row.IsBoss {
				waveLabel += " ★Boss"
			}
			waveLabel += ":"

			// Build composition string.
			compStr := ""
			for i, e := range row.Entries {
				if i > 0 {
					compStr += ", "
				}
				compStr += e.Label + " x" + strconv.Itoa(e.Count)
			}
			compStr += "  (总" + strconv.Itoa(row.Total) + "只)"

			// Wave number in accent color.
			waveClr := theme.TextBody
			if row.IsBoss {
				waveClr = color.RGBA{R: 255, G: 180, B: 60, A: 255}
			}

			// Truncate if too long.
			fullText := waveLabel + " " + compStr
			_ = maxTextW // text will naturally clip at screen edge

			fm.DrawText(screen, fullText, panelLeft, y+4, 11, waveClr)
		}
		y += wpWaveRowH
	}

	// Compute max scroll for wave panel.
	totalH := y + s.waveScrollY - (panelTop + wpHeaderH)
	visibleH := panelBottom - (panelTop + wpHeaderH)
	s.waveMaxScroll = totalH - visibleH
	if s.waveMaxScroll < 0 {
		s.waveMaxScroll = 0
	}
}

func (s *WavePreviewScene) drawControls(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	if fm == nil {
		return
	}

	btnW := 72.0
	btnH := 30.0
	baseY := sh - wpCtrlH + (wpCtrlH-btnH)/2

	// Single "返回" button centered.
	bx := (sw - btnW) / 2
	draw.RoundRect(screen, float32(bx), float32(baseY), float32(btnW), float32(btnH), 8, theme.BtnSecondary)
	fm.DrawCenteredText(screen, "返回", bx+btnW/2, baseY+7, 11, theme.TextTitle)

	// Map info at bottom-left.
	if len(s.maps) > 0 && s.selectedID >= 0 && s.selectedID < len(s.maps) {
		m := s.maps[s.selectedID]
		info := m.ID
		if m.Description != "" {
			info += " — " + m.Description
		}
		fm.DrawText(screen, info, wpMapPanelW+12, sh-wpCtrlH+wpCtrlH/2-6, 11, theme.TextMuted)
	}
}
