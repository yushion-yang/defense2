// map_editor.go — Map Editor scene (tower slot editor).
// Dev tool for visually toggling buildable cells (0↔2) on existing maps.
// Accessible from the TestSelect scene via the "map-editor" scenario entry.
package scene

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Layout constants ────────────────────────────────

const (
	meTopBarH    = 44.0  // top bar height (map selector)
	meCtrlH      = 44.0  // bottom control bar height
	meTabW       = 110.0 // map tab width
	meTabH       = 26.0  // map tab height
	meTabGap     = 6.0   // gap between map tabs
	meGridLineW  = 0.5   // grid line width
	meCellBorder = 1.5   // hover cell border width
)

// ── MapEditorScene ──────────────────────────────────

// MapEditorScene provides a visual editor for toggling tower slots on maps.
type MapEditorScene struct {
	switcher Switcher

	// Map data
	maps        []config.LevelEntry // available map list
	selectedIdx int                 // current map index
	hoverTab    int                 // hovered map tab index
	cfg         *config.MapConfig   // current map config (mutable)
	gm          *gamemap.GameMap    // runtime map for coordinate conversion

	// Grid interaction
	hoverRow int // hovered grid cell (-1 = none)
	hoverCol int

	// Scroll for map tabs
	tabScrollX float64

	// Edit state
	dirty bool // unsaved changes

	// Toast feedback
	toast      string
	toastTimer float64
}

// NewMapEditorScene creates a map editor scene.
func NewMapEditorScene(sw Switcher) *MapEditorScene {
	s := &MapEditorScene{
		switcher: sw,
		hoverTab: -1,
		hoverRow: -1,
		hoverCol: -1,
	}
	s.maps, _ = config.LoadLevelList()
	if len(s.maps) > 0 {
		s.loadMap(0)
	}
	return s
}

// loadMap loads the map at the given index.
func (s *MapEditorScene) loadMap(idx int) {
	if idx < 0 || idx >= len(s.maps) {
		return
	}
	cfg, err := config.LoadMap(s.maps[idx].ID)
	if err != nil {
		return
	}
	s.selectedIdx = idx
	s.cfg = cfg
	s.gm = gamemap.NewGameMap(cfg)
	s.dirty = false
	s.hoverRow = -1
	s.hoverCol = -1
}

// ── Update ──────────────────────────────────────────

func (s *MapEditorScene) Update() error {
	if s.toastTimer > 0 {
		s.toastTimer -= 1.0 / 60.0
	}

	// ── Keyboard shortcuts ──
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		return nil
	}
	// Ctrl+S save
	if inpututil.IsKeyJustPressed(ebiten.KeyS) &&
		(ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyControl)) {
		s.saveMap()
		return nil
	}
	// Left/Right arrow to switch maps
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && s.selectedIdx > 0 {
		s.loadMap(s.selectedIdx - 1)
		playUIClick(s.switcher)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && s.selectedIdx < len(s.maps)-1 {
		s.loadMap(s.selectedIdx + 1)
		playUIClick(s.switcher)
	}

	// ── Hover detection (desktop=mouse, touch=long-press) ──
	if hx, hy, hov := draw.HoverPos(); hov {
		s.hoverTab = s.hitTestMapTabs(hx, hy)
		s.updateHoverCell(hx, hy)
	} else {
		s.hoverTab = -1
		s.hoverRow, s.hoverCol = -1, -1
	}

	// ── Click handling ──
	mx, my := draw.CursorPos()
	if isTapJustPressed() {
		// Back button
		if mx >= 10 && mx <= 70 && my >= 10 && my <= 34 {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
			return nil
		}
		// Map tab click
		if idx := s.hitTestMapTabs(mx, my); idx >= 0 && idx != s.selectedIdx {
			s.loadMap(idx)
			playUIClick(s.switcher)
			return nil
		}
		// Save button
		if s.hitTestSaveBtn(mx, my) {
			s.saveMap()
			return nil
		}
		// Grid cell toggle
		if s.hoverRow >= 0 && s.cfg != nil {
			s.toggleCell(s.hoverRow, s.hoverCol)
		}
	}

	return nil
}

// toggleCell switches empty↔buildable for the given cell.
func (s *MapEditorScene) toggleCell(row, col int) {
	if s.cfg == nil || row < 0 || row >= s.cfg.Rows || col < 0 || col >= s.cfg.Cols {
		return
	}
	cell := s.cfg.Grid[row][col]
	switch cell {
	case config.CellEmpty:
		s.cfg.Grid[row][col] = config.CellBuildable
		s.dirty = true
		playUIClick(s.switcher)
	case config.CellBuildable:
		s.cfg.Grid[row][col] = config.CellEmpty
		s.dirty = true
		playUIClick(s.switcher)
	}
	// Path/Spawn/Base cells are not editable — no action.
}

// saveMap writes the current config back to config/levels/{id}.json.
func (s *MapEditorScene) saveMap() {
	if s.cfg == nil {
		return
	}
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		s.toast = fmt.Sprintf("JSON error: %v", err)
		s.toastTimer = 3.0
		return
	}
	// Append trailing newline to match source file convention.
	data = append(data, '\n')

	path := filepath.Join("config", "levels", s.cfg.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		s.toast = fmt.Sprintf("Save error: %v", err)
		s.toastTimer = 3.0
		return
	}
	s.dirty = false
	s.toast = "Saved " + s.cfg.ID
	s.toastTimer = 2.0
	playUIClick(s.switcher)
}

// ── Hit testing ─────────────────────────────────────

func (s *MapEditorScene) hitTestMapTabs(mx, my float64) int {
	sw := float64(game.ScreenWidth)
	n := len(s.maps)
	totalW := float64(n)*meTabW + float64(n-1)*meTabGap
	startX := (sw - totalW) / 2
	if totalW > sw-120 {
		startX = 60 - s.tabScrollX
	}

	for i := range s.maps {
		x := startX + float64(i)*(meTabW+meTabGap)
		if mx >= x && mx <= x+meTabW && my >= 8 && my <= 8+meTabH {
			return i
		}
	}
	return -1
}

func (s *MapEditorScene) hitTestSaveBtn(mx, my float64) bool {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	bx := sw - 120
	by := sh - meCtrlH + 8
	return mx >= bx && mx <= bx+100 && my >= by && my <= by+28
}

func (s *MapEditorScene) updateHoverCell(mx, my float64) {
	s.hoverRow = -1
	s.hoverCol = -1
	if s.cfg == nil || s.gm == nil {
		return
	}

	ox, oy := s.gridOffset()
	cs := float64(s.cfg.CellSize)
	col := int((mx - ox) / cs)
	row := int((my - oy) / cs)
	if row >= 0 && row < s.cfg.Rows && col >= 0 && col < s.cfg.Cols {
		s.hoverRow = row
		s.hoverCol = col
	}
}

// gridOffset returns the top-left pixel offset for centering the grid.
func (s *MapEditorScene) gridOffset() (float64, float64) {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	gridW := float64(s.cfg.Cols * s.cfg.CellSize)
	gridH := float64(s.cfg.Rows * s.cfg.CellSize)

	// Available area: below top bar, above control bar
	areaW := sw
	areaH := sh - meTopBarH - meCtrlH
	areaY := meTopBarH

	// Scale down if grid doesn't fit
	scale := 1.0
	if gridW > areaW || gridH > areaH {
		sx := areaW / gridW
		sy := areaH / gridH
		if sx < sy {
			scale = sx
		} else {
			scale = sy
		}
	}
	_ = scale // for now we use 1:1; maps are designed to fit

	ox := (areaW - gridW) / 2
	oy := areaY + (areaH-gridH)/2
	return ox, oy
}

// ── Draw ────────────────────────────────────────────

func (s *MapEditorScene) Draw(screen *ebiten.Image) {
	// Background gradient
	draw.LinearGradientV(screen, 0, 0, game.ScreenWidth, game.ScreenHeight,
		theme.SelectGradTop, theme.SelectGradBot)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// ── Top bar: back button + map tabs ──
	s.drawTopBar(screen, fm, sw)

	// ── Grid ──
	if s.cfg != nil {
		s.drawGrid(screen, fm)
	}

	// ── Bottom control bar ──
	s.drawControlBar(screen, fm, sw, sh)

	// ── Toast ──
	if s.toastTimer > 0 && s.toast != "" {
		alpha := 1.0
		if s.toastTimer < 0.5 {
			alpha = s.toastTimer / 0.5
		}
		a := uint8(alpha * 220)
		bg := color.RGBA{R: 15, G: 23, B: 42, A: a}
		tw := float64(len(s.toast)*8 + 40)
		tx := (sw - tw) / 2
		ty := sh - meCtrlH - 50
		draw.RoundRect(screen, float32(tx), float32(ty), float32(tw), 30, 10, bg)
		txtClr := color.RGBA{R: 241, G: 245, B: 249, A: a}
		fm.DrawCenteredText(screen, s.toast, sw/2, ty+7, theme.FontMD, txtClr)
	}
}

func (s *MapEditorScene) drawTopBar(screen *ebiten.Image, fm *render.FontManager, sw float64) {
	// Background strip
	draw.FilledRect(screen, 0, 0, float32(sw), float32(meTopBarH), theme.PanelBg, false)

	// Back button
	draw.RoundRect(screen, 10, 8, 60, 26, 10, theme.BtnSecondary)
	fm.DrawCenteredText(screen, "< Back", 40, 14, theme.FontSM, theme.TextBody)

	// Map tabs
	n := len(s.maps)
	if n == 0 {
		return
	}
	totalW := float64(n)*meTabW + float64(n-1)*meTabGap
	startX := (sw - totalW) / 2
	if totalW > sw-140 {
		startX = 80 - s.tabScrollX
	}

	for i, m := range s.maps {
		x := float32(startX + float64(i)*(meTabW+meTabGap))
		y := float32(8)
		active := i == s.selectedIdx
		hovered := i == s.hoverTab

		bg := theme.BtnMuted
		if active {
			bg = theme.BtnPrimary
		} else if hovered {
			bg = theme.BtnSecondary
		}
		draw.RoundRect(screen, x, y, float32(meTabW), float32(meTabH), 8, bg)

		txtClr := theme.TextMuted
		if active {
			txtClr = theme.TextTitle
		}
		label := m.Name
		if len(label) > 10 {
			label = label[:10]
		}
		fm.DrawCenteredText(screen, label, float64(x)+meTabW/2, float64(y)+6, 11, txtClr)
	}
}

func (s *MapEditorScene) drawGrid(screen *ebiten.Image, fm *render.FontManager) {
	ox, oy := s.gridOffset()
	cs := float64(s.cfg.CellSize)
	rows := s.cfg.Rows
	cols := s.cfg.Cols
	mt := theme.MapThemeFor(s.cfg.Theme)

	// Grid background
	gridW := float64(cols) * cs
	gridH := float64(rows) * cs
	draw.FilledRect(screen, float32(ox), float32(oy), float32(gridW), float32(gridH), mt.GradientTop, false)

	// Draw cells
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := s.cfg.Grid[row][col]
			cx := ox + float64(col)*cs
			cy := oy + float64(row)*cs
			centerX := cx + cs/2
			centerY := cy + cs/2

			switch cell {
			case config.CellPath:
				// Path cells: filled with path color
				draw.FilledRect(screen, float32(cx+1), float32(cy+1), float32(cs-2), float32(cs-2),
					withAlpha(mt.PathColor, 100), false)

			case config.CellBuildable:
				// Tower slot: circle marker
				r := cs/2 - 4
				draw.FilledCircle(screen, float32(centerX), float32(centerY), float32(r),
					color.RGBA{R: 80, G: 180, B: 100, A: 60})
				draw.CircleOutline(screen, float32(centerX), float32(centerY), float32(r), 1.5,
					color.RGBA{R: 80, G: 200, B: 120, A: 160})
				fm.DrawCenteredText(screen, "+", centerX, centerY-6, 14,
					color.RGBA{R: 120, G: 220, B: 140, A: 180})

			case config.CellSpawn:
				// Spawn point: red marker
				r := cs/2 - 3
				draw.FilledCircle(screen, float32(centerX), float32(centerY), float32(r),
					color.RGBA{R: 200, G: 60, B: 60, A: 80})
				draw.CircleOutline(screen, float32(centerX), float32(centerY), float32(r), 2,
					color.RGBA{R: 220, G: 80, B: 80, A: 200})
				fm.DrawCenteredText(screen, "S", centerX, centerY-6, 12,
					color.RGBA{R: 255, G: 120, B: 120, A: 220})

			case config.CellBase:
				// Base: blue marker
				r := cs/2 - 3
				draw.FilledCircle(screen, float32(centerX), float32(centerY), float32(r),
					color.RGBA{R: 60, G: 60, B: 200, A: 80})
				draw.CircleOutline(screen, float32(centerX), float32(centerY), float32(r), 2,
					color.RGBA{R: 80, G: 80, B: 220, A: 200})
				fm.DrawCenteredText(screen, "B", centerX, centerY-6, 12,
					color.RGBA{R: 120, G: 120, B: 255, A: 220})
			}
		}
	}

	// Grid lines
	gridLineClr := color.RGBA{R: 255, G: 255, B: 255, A: 15}
	for row := 0; row <= rows; row++ {
		y := float32(oy + float64(row)*cs)
		draw.Line(screen, float32(ox), y, float32(ox+gridW), y, meGridLineW, gridLineClr, false)
	}
	for col := 0; col <= cols; col++ {
		x := float32(ox + float64(col)*cs)
		draw.Line(screen, x, float32(oy), x, float32(oy+gridH), meGridLineW, gridLineClr, false)
	}

	// Hover highlight
	if s.hoverRow >= 0 && s.hoverCol >= 0 {
		hx := float32(ox + float64(s.hoverCol)*cs)
		hy := float32(oy + float64(s.hoverRow)*cs)
		cell := s.cfg.Grid[s.hoverRow][s.hoverCol]

		// Highlight fill
		var hlClr color.RGBA
		switch cell {
		case config.CellEmpty:
			hlClr = color.RGBA{R: 80, G: 200, B: 120, A: 40} // green hint: will become buildable
		case config.CellBuildable:
			hlClr = color.RGBA{R: 200, G: 80, B: 80, A: 40} // red hint: will become empty
		default:
			hlClr = color.RGBA{R: 150, G: 150, B: 150, A: 20} // locked
		}
		draw.FilledRect(screen, hx, hy, float32(cs), float32(cs), hlClr, false)

		// Hover border
		borderClr := color.RGBA{R: 255, G: 255, B: 255, A: 80}
		if cell != config.CellEmpty && cell != config.CellBuildable {
			borderClr = color.RGBA{R: 255, G: 80, B: 80, A: 60} // locked indicator
		}
		draw.StrokeRect(screen, hx, hy, float32(cs), float32(cs), meCellBorder, borderClr)
	}
}

func (s *MapEditorScene) drawControlBar(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	// Background strip
	barY := sh - meCtrlH
	draw.FilledRect(screen, 0, float32(barY), float32(sw), float32(meCtrlH), theme.PanelBg, false)

	// Map info (left side)
	if s.cfg != nil {
		info := s.cfg.ID + "  " + strconv.Itoa(s.cfg.Cols) + "x" + strconv.Itoa(s.cfg.Rows)
		fm.DrawText(screen, info, 14, barY+14, theme.FontSM, theme.TextMuted)

		// Slot count
		slots := s.countSlots()
		slotText := "Slots: " + strconv.Itoa(slots)
		fm.DrawText(screen, slotText, 180, barY+14, theme.FontSM, theme.TextBody)
	}

	// Dirty indicator (center)
	if s.dirty {
		fm.DrawCenteredText(screen, "* unsaved", sw/2, barY+14, theme.FontSM,
			color.RGBA{R: 251, G: 191, B: 36, A: 220})
	}

	// Save button (right side)
	bx := float32(sw - 120)
	by := float32(barY + 8)
	btnClr := theme.BtnMuted
	if s.dirty {
		btnClr = theme.TonePrimary
	}
	draw.RoundRect(screen, bx, by, 100, 28, 10, btnClr)
	txtClr := theme.TextMuted
	if s.dirty {
		txtClr = theme.TextTitle
	}
	fm.DrawCenteredText(screen, "Save (^S)", float64(bx)+50, float64(by)+7, theme.FontSM, txtClr)
}

// countSlots counts CellBuildable cells in the current map.
func (s *MapEditorScene) countSlots() int {
	if s.cfg == nil {
		return 0
	}
	count := 0
	for _, row := range s.cfg.Grid {
		for _, cell := range row {
			if cell == config.CellBuildable {
				count++
			}
		}
	}
	return count
}

// withAlpha returns a copy of the color with the given alpha.
func withAlpha(c color.RGBA, a uint8) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: a}
}
