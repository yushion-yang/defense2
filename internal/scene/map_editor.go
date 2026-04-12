// map_editor.go — Map Editor scene (tower slot editor).
// Dev tool for visually toggling buildable cells (0↔2) on existing maps.
// Accessible from the TestSelect scene via the "map-editor" scenario entry.
package scene

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
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
	meCtrlH      = 44.0 // bottom control bar height
	meGridLineW  = 0.5  // grid line width
	meCellBorder = 1.5  // hover cell border width
	meCamStep    = 30.0 // camera step per arrow key press
	meCamScroll  = 3.0  // scroll wheel multiplier
	meArrowBtnW  = 28.0 // map prev/next arrow button width
	meArrowBtnH  = 28.0 // map prev/next arrow button height
)

// ── MapEditorScene ──────────────────────────────────

// MapEditorScene provides a visual editor for toggling tower slots on maps.
type MapEditorScene struct {
	switcher Switcher

	// Map data
	maps        []config.LevelEntry // available map list
	selectedIdx int                 // current map index
	cfg         *config.MapConfig   // current map config (mutable)
	gm          *gamemap.GameMap    // runtime map for coordinate conversion

	// Grid interaction
	hoverRow int // hovered grid cell (-1 = none)
	hoverCol int

	// Camera (2D pan)
	camX          float64 // camera X offset
	camY          float64 // camera Y offset (0 = grid top-left at screen top-left)
	camDragging   bool    // right-click drag active
	camDragStartX float64 // screen X at drag start
	camDragStartY float64 // screen Y at drag start
	camDragCamX   float64 // camX at drag start
	camDragCamY   float64 // camY at drag start

	// Edit state
	dirty bool // unsaved changes

	// Toast feedback
	toast      string
	toastTimer float64

	// HUD hover state
	hoverBack bool // hovering back button
	hoverPrev bool // hovering prev-map arrow
	hoverNext bool // hovering next-map arrow
	hoverSave bool // hovering save button

	// D-pad hover/press state (floating overlay)
	dpadHover int // 0=none, 1=up, 2=down, 3=left, 4=right
}

// NewMapEditorScene creates a map editor scene.
func NewMapEditorScene(sw Switcher) *MapEditorScene {
	s := &MapEditorScene{
		switcher: sw,
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
	s.camX = 0
	s.camY = 0
}

// ── Update ──────────────────────────────────────────

func (s *MapEditorScene) Update() error {
	if s.toastTimer > 0 {
		s.toastTimer -= 1.0 / 60.0
	}

	mx, my := draw.CursorPos()
	sh := float64(game.ScreenHeight)

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
	// Arrow keys for camera pan (held = continuous)
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		s.camY -= meCamStep * 0.5
		s.clampCamera()
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		s.camY += meCamStep * 0.5
		s.clampCamera()
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		s.camX -= meCamStep * 0.5
		s.clampCamera()
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		s.camX += meCamStep * 0.5
		s.clampCamera()
	}

	// ── Scroll wheel for camera ──
	wx, wy := ebiten.Wheel()
	if wy != 0 {
		s.camY -= wy * meCamScroll
		s.clampCamera()
	}
	if wx != 0 {
		s.camX -= wx * meCamScroll
		s.clampCamera()
	}

	// ── Right-click / middle-click drag for camera ──
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		s.camDragging = true
		s.camDragStartX = mx
		s.camDragStartY = my
		s.camDragCamX = s.camX
		s.camDragCamY = s.camY
	}
	if s.camDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) ||
			ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			dx := mx - s.camDragStartX
			dy := my - s.camDragStartY
			s.camX = s.camDragCamX - dx
			s.camY = s.camDragCamY - dy
			s.clampCamera()
		} else {
			s.camDragging = false
		}
	}

	// ── Hover detection (desktop=mouse, touch=long-press) ──
	hx, hy, hov := draw.HoverPos()
	if hov {
		s.dpadHover = s.hitTestDpad(hx, hy)
	} else {
		s.dpadHover = 0
	}

	inHUD := my >= sh-meCtrlH
	onDpad := s.dpadHover > 0
	s.hoverBack = false
	s.hoverPrev = false
	s.hoverNext = false
	s.hoverSave = false
	if hov && inHUD {
		s.hoverBack = s.hitTestBackBtn(hx, hy)
		s.hoverPrev = s.hitTestPrevBtn(hx, hy)
		s.hoverNext = s.hitTestNextBtn(hx, hy)
		s.hoverSave = s.hitTestSaveBtn(hx, hy)
	}

	// ── Grid hover (only above HUD and not on D-pad) ──
	if !hov || inHUD || onDpad {
		s.hoverRow = -1
		s.hoverCol = -1
	} else {
		s.updateHoverCell(hx, hy)
	}

	// ── D-pad click (held = continuous pan) ──
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && onDpad {
		switch s.dpadHover {
		case 1: // up
			s.camY -= meCamStep * 0.4
		case 2: // down
			s.camY += meCamStep * 0.4
		case 3: // left
			s.camX -= meCamStep * 0.4
		case 4: // right
			s.camX += meCamStep * 0.4
		}
		s.clampCamera()
	}

	// ── Click handling ──
	if isTapJustPressed() {
		if onDpad {
			// D-pad consumed the tap, don't propagate
		} else if inHUD {
			// Back button
			if s.hitTestBackBtn(mx, my) {
				playUIClick(s.switcher)
				s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
				return nil
			}
			// Prev map
			if s.hitTestPrevBtn(mx, my) && s.selectedIdx > 0 {
				s.loadMap(s.selectedIdx - 1)
				playUIClick(s.switcher)
				return nil
			}
			// Next map
			if s.hitTestNextBtn(mx, my) && s.selectedIdx < len(s.maps)-1 {
				s.loadMap(s.selectedIdx + 1)
				playUIClick(s.switcher)
				return nil
			}
			// Save button
			if s.hitTestSaveBtn(mx, my) {
				s.saveMap()
				return nil
			}
		} else {
			// Grid cell toggle
			if s.hoverRow >= 0 && s.cfg != nil {
				s.toggleCell(s.hoverRow, s.hoverCol)
			}
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

// ── Camera ──────────────────────────────────────────

// clampCamera constrains camX/camY within valid range.
func (s *MapEditorScene) clampCamera() {
	mxX := s.maxCamX()
	mxY := s.maxCamY()
	if s.camX < 0 {
		s.camX = 0
	}
	if s.camX > mxX {
		s.camX = mxX
	}
	if s.camY < 0 {
		s.camY = 0
	}
	if s.camY > mxY {
		s.camY = mxY
	}
}

// maxCamX returns the maximum camera X offset.
func (s *MapEditorScene) maxCamX() float64 {
	if s.cfg == nil {
		return 0
	}
	gridW := float64(s.cfg.Cols * s.cfg.CellSize)
	visibleW := float64(game.ScreenWidth)
	return math.Max(0, gridW-visibleW)
}

// maxCamY returns the maximum camera Y offset.
func (s *MapEditorScene) maxCamY() float64 {
	if s.cfg == nil {
		return 0
	}
	gridH := float64(s.cfg.Rows * s.cfg.CellSize)
	visibleH := float64(game.ScreenHeight) - meCtrlH
	return math.Max(0, gridH-visibleH)
}

// needsCamera returns true if the grid is larger than the visible area.
func (s *MapEditorScene) needsCamera() bool {
	return s.maxCamX() > 0 || s.maxCamY() > 0
}

// ── Hit testing ─────────────────────────────────────

// Bottom HUD layout (left to right):
// [10] [Back 60x28] [20gap] [< 28x28] [8gap] [mapName ~120] [8gap] [> 28x28] [20gap] [Slots text] ... [dirty center] ... [Save 100x28] [10]

const (
	meBackBtnX = 10.0
	meBackBtnW = 60.0
	meBackBtnH = 28.0
	mePrevBtnX = 90.0  // 10 + 60 + 20
	meMapNameX = 126.0 // 90 + 28 + 8
	meMapNameW = 120.0
	meNextBtnX = 254.0 // 126 + 120 + 8
	meSlotsX   = 300.0 // 254 + 28 + 18
)

func (s *MapEditorScene) hitTestBackBtn(mx, my float64) bool {
	sh := float64(game.ScreenHeight)
	by := sh - meCtrlH + (meCtrlH-meBackBtnH)/2
	return mx >= meBackBtnX && mx <= meBackBtnX+meBackBtnW && my >= by && my <= by+meBackBtnH
}

func (s *MapEditorScene) hitTestPrevBtn(mx, my float64) bool {
	sh := float64(game.ScreenHeight)
	by := sh - meCtrlH + (meCtrlH-meArrowBtnH)/2
	return mx >= mePrevBtnX && mx <= mePrevBtnX+meArrowBtnW && my >= by && my <= by+meArrowBtnH
}

func (s *MapEditorScene) hitTestNextBtn(mx, my float64) bool {
	sh := float64(game.ScreenHeight)
	by := sh - meCtrlH + (meCtrlH-meArrowBtnH)/2
	return mx >= meNextBtnX && mx <= meNextBtnX+meArrowBtnW && my >= by && my <= by+meArrowBtnH
}

func (s *MapEditorScene) hitTestSaveBtn(mx, my float64) bool {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	bx := sw - 110
	by := sh - meCtrlH + (meCtrlH-28)/2
	return mx >= bx && mx <= bx+100 && my >= by && my <= by+28
}

// D-pad layout: floating in bottom-right corner, above HUD.
// 3x3 grid of 24x24 buttons with 2px gaps:
//
//	[  Up  ]
//
// [Left] [    ] [Right]
//
//	[ Down ]
const (
	meDpadBtnSize = 24.0
	meDpadGap     = 2.0
	meDpadMarginR = 120.0 // right margin (save btn is at sw-110)
	meDpadMarginB = 8.0   // margin above HUD bar
)

// dpadOrigin returns the top-left of the 3x3 D-pad grid.
func (s *MapEditorScene) dpadOrigin() (float64, float64) {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	totalW := meDpadBtnSize*3 + meDpadGap*2
	totalH := meDpadBtnSize*3 + meDpadGap*2
	x := sw - meDpadMarginR - totalW
	y := sh - meCtrlH - meDpadMarginB - totalH
	return x, y
}

// hitTestDpad returns which D-pad button the cursor is on (0=none, 1=up, 2=down, 3=left, 4=right).
func (s *MapEditorScene) hitTestDpad(mx, my float64) int {
	if !s.needsCamera() {
		return 0
	}
	ox, oy := s.dpadOrigin()
	step := meDpadBtnSize + meDpadGap

	// Up: row 0, col 1
	if s.inBtn(mx, my, ox+step, oy, meDpadBtnSize, meDpadBtnSize) {
		return 1
	}
	// Down: row 2, col 1
	if s.inBtn(mx, my, ox+step, oy+step*2, meDpadBtnSize, meDpadBtnSize) {
		return 2
	}
	// Left: row 1, col 0
	if s.inBtn(mx, my, ox, oy+step, meDpadBtnSize, meDpadBtnSize) {
		return 3
	}
	// Right: row 1, col 2
	if s.inBtn(mx, my, ox+step*2, oy+step, meDpadBtnSize, meDpadBtnSize) {
		return 4
	}
	return 0
}

func (s *MapEditorScene) inBtn(mx, my, bx, by, bw, bh float64) bool {
	return mx >= bx && mx <= bx+bw && my >= by && my <= by+bh
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

// gridOffset returns the top-left pixel offset for the grid.
// Grid starts at screen top-left, shifted by camera offset.
// When grid fits horizontally, it is centered; otherwise shifted by camX.
func (s *MapEditorScene) gridOffset() (float64, float64) {
	sw := float64(game.ScreenWidth)
	gridW := float64(s.cfg.Cols * s.cfg.CellSize)
	ox := -s.camX
	if gridW <= sw {
		// Center the grid horizontally when it fits
		ox = (sw - gridW) / 2
	}
	oy := -s.camY
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

	// ── Grid (clipped to area above HUD) ──
	if s.cfg != nil {
		clipH := int(sh - meCtrlH)
		clipRect := image.Rect(0, 0, int(sw*draw.Scale), int(float64(clipH)*draw.Scale))
		clipped := screen.SubImage(clipRect).(*ebiten.Image)
		s.drawGrid(clipped, fm)
	}

	// ── D-pad overlay (above HUD, only if camera is needed) ──
	if s.needsCamera() {
		s.drawDpad(screen, fm)
	}

	// ── Bottom control bar (drawn on full screen, above grid) ──
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

func (s *MapEditorScene) drawDpad(screen *ebiten.Image, fm *render.FontManager) {
	ox, oy := s.dpadOrigin()
	step := meDpadBtnSize + meDpadGap
	sz := float32(meDpadBtnSize)

	bgNorm := color.RGBA{R: 30, G: 40, B: 60, A: 160}
	bgHover := color.RGBA{R: 60, G: 80, B: 120, A: 200}
	txtNorm := color.RGBA{R: 180, G: 190, B: 210, A: 200}
	txtHover := color.RGBA{R: 240, G: 245, B: 255, A: 255}

	type dpadBtn struct {
		col, row int
		label    string
		id       int // matches dpadHover values
	}
	btns := []dpadBtn{
		{1, 0, "^", 1}, // up
		{1, 2, "v", 2}, // down
		{0, 1, "<", 3}, // left
		{2, 1, ">", 4}, // right
	}

	for _, b := range btns {
		bx := float32(ox + float64(b.col)*step)
		by := float32(oy + float64(b.row)*step)
		bg := bgNorm
		tc := txtNorm
		if s.dpadHover == b.id {
			bg = bgHover
			tc = txtHover
		}
		draw.RoundRect(screen, bx, by, sz, sz, 6, bg)
		fm.DrawCenteredText(screen, b.label, float64(bx)+float64(sz)/2, float64(by)+6, 11, tc)
	}
}

func (s *MapEditorScene) drawControlBar(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	barY := sh - meCtrlH
	btnCenterY := barY + meCtrlH/2

	// Background strip
	draw.FilledRect(screen, 0, float32(barY), float32(sw), float32(meCtrlH), theme.PanelBg, false)

	// ── Back button (left) ──
	{
		bx := float32(meBackBtnX)
		by := float32(btnCenterY - meBackBtnH/2)
		bg := theme.BtnSecondary
		if s.hoverBack {
			bg = theme.BtnPrimary
		}
		draw.RoundRect(screen, bx, by, float32(meBackBtnW), float32(meBackBtnH), 10, bg)
		fm.DrawCenteredText(screen, "← 返回", float64(bx)+meBackBtnW/2, float64(by)+7, theme.FontSM, theme.TextBody)
	}

	// ── Map prev arrow ──
	{
		bx := float32(mePrevBtnX)
		by := float32(btnCenterY - meArrowBtnH/2)
		canPrev := s.selectedIdx > 0
		bg := theme.BtnMuted
		txtClr := theme.TextMuted
		if canPrev {
			bg = theme.BtnSecondary
			txtClr = theme.TextBody
			if s.hoverPrev {
				bg = theme.BtnPrimary
				txtClr = theme.TextTitle
			}
		}
		draw.RoundRect(screen, bx, by, float32(meArrowBtnW), float32(meArrowBtnH), 8, bg)
		fm.DrawCenteredText(screen, "<", float64(bx)+meArrowBtnW/2, float64(by)+7, theme.FontSM, txtClr)
	}

	// ── Map name (center label) ──
	{
		name := ""
		if s.cfg != nil {
			name = s.cfg.ID
		}
		fm.DrawCenteredText(screen, name, meMapNameX+meMapNameW/2, barY+14, theme.FontMD, theme.TextTitle)
	}

	// ── Map next arrow ──
	{
		bx := float32(meNextBtnX)
		by := float32(btnCenterY - meArrowBtnH/2)
		canNext := s.selectedIdx < len(s.maps)-1
		bg := theme.BtnMuted
		txtClr := theme.TextMuted
		if canNext {
			bg = theme.BtnSecondary
			txtClr = theme.TextBody
			if s.hoverNext {
				bg = theme.BtnPrimary
				txtClr = theme.TextTitle
			}
		}
		draw.RoundRect(screen, bx, by, float32(meArrowBtnW), float32(meArrowBtnH), 8, bg)
		fm.DrawCenteredText(screen, ">", float64(bx)+meArrowBtnW/2, float64(by)+7, theme.FontSM, txtClr)
	}

	// ── Slots count ──
	if s.cfg != nil {
		slotText := "Slots: " + strconv.Itoa(s.countSlots())
		fm.DrawText(screen, slotText, meSlotsX, barY+14, theme.FontSM, theme.TextBody)
	}

	// ── Dirty indicator (center) ──
	if s.dirty {
		fm.DrawCenteredText(screen, "* 未保存", sw/2, barY+14, theme.FontSM,
			color.RGBA{R: 251, G: 191, B: 36, A: 220})
	}

	// ── Save button (right side) ──
	{
		bx := float32(sw - 110)
		by := float32(btnCenterY - 14)
		btnClr := theme.BtnMuted
		txtClr := theme.TextMuted
		if s.dirty {
			btnClr = theme.TonePrimary
			txtClr = theme.TextTitle
			if s.hoverSave {
				btnClr = theme.BtnPrimary
			}
		} else if s.hoverSave {
			btnClr = theme.BtnSecondary
			txtClr = theme.TextBody
		}
		draw.RoundRect(screen, bx, by, 100, 28, 10, btnClr)
		fm.DrawCenteredText(screen, "保存 ^S", float64(bx)+50, float64(by)+7, theme.FontSM, txtClr)
	}

	// ── Scroll indicator (right of slots, if scrollable) ──
	if s.needsCamera() {
		parts := ""
		if s.maxCamX() > 0 {
			parts += fmt.Sprintf("X:%.0f%%", s.camX/s.maxCamX()*100)
		}
		if s.maxCamY() > 0 {
			if parts != "" {
				parts += " "
			}
			parts += fmt.Sprintf("Y:%.0f%%", s.camY/s.maxCamY()*100)
		}
		fm.DrawText(screen, parts, meSlotsX+80, barY+14, theme.FontSM, theme.TextMuted)
	}
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
