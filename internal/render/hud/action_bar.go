// action_bar.go — Bottom-center pill-shaped action toolbar.
// Displays [造塔] [道具] buttons for primary player actions.
//
// Draws directly to the screen each frame (no offscreen cache) so that
// draw.* auto-scaling works correctly on HiDPI displays.
// Button hit detection uses rects computed during the last render.
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ActionBarData holds the runtime data the action bar needs to render.
type ActionBarData struct {
	BuildActive   bool            // whether build mode is active
	ItemActive    bool            // whether item mode is active
	ItemTotal     int             // total items in inventory
	BuildBtnState *ui.ButtonState // optional micro-interaction state for build button
	ItemBtnState  *ui.ButtonState // optional micro-interaction state for items button
}

// DrawActionBar renders the centered pill-shaped action bar at the bottom of the screen.
func DrawActionBar(screen *ebiten.Image, d ActionBarData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		pillH  = float32(theme.ActionBarH)
		btnH   = float32(theme.ActionBarBtnH)
		btnGap = float32(theme.ActionBarBtnGap)
		btnR   = float32(theme.ActionBarBtnR)
		pillR  = float32(16)
	)
	pillY := float32(theme.CanvasH) - float32(theme.BottomMargin) - pillH

	// ── Build button list ──
	type btnDef struct {
		name  string
		label string
		clr   color.RGBA
	}
	var btnsBuf [4]btnDef
	btns := btnsBuf[:0]

	// Build button: active=green, default=secondary
	buildClr := theme.ToneSecondary
	if d.BuildActive {
		buildClr = theme.TonePrimary
	}
	btns = append(btns, btnDef{"build", "造塔", buildClr})

	// Items button
	itemClr := theme.ToneSecondary
	if d.ItemActive {
		itemClr = theme.ToneAccent
	} else if d.ItemTotal == 0 {
		itemClr = theme.ToneDisabled
	}
	itemLabel := "道具"
	if d.ItemTotal > 0 {
		itemLabel = "道具(" + strconv.Itoa(d.ItemTotal) + ")"
	}
	btns = append(btns, btnDef{"items", itemLabel, itemClr})

	// ── Measure total button width ──
	// Hover detection using last frame's rects
	mx, my := draw.CursorPos()
	hoverIdx := ui.HitTestButtonRow(lastActionBarBtnRects, float64(mx), float64(my))

	const btnPadX float32 = 16
	var itemBuf [4]ui.ButtonRowItem
	var nameBuf [4]string
	items := itemBuf[:len(btns)]
	names := nameBuf[:len(btns)]
	totalBtnW := float32(0)
	for i, b := range btns {
		// 直接查找按钮状态，避免 per-frame map 分配
		var st *ui.ButtonState
		switch b.name {
		case "build":
			st = d.BuildBtnState
		case "items":
			st = d.ItemBtnState
		}
		items[i] = ui.ButtonRowItem{Label: b.label, Color: b.clr, State: st, Hovered: i == hoverIdx}
		names[i] = b.name
		tw := float32(fm.MeasureText(b.label, theme.FontH2))
		w := tw + btnPadX*2
		if w < 40 {
			w = 40
		}
		totalBtnW += w
	}
	totalBtnW += float32(len(items)-1) * btnGap

	// Pill width = buttons + horizontal padding
	const pillPadX float32 = 12
	pillW := totalBtnW + pillPadX*2
	pillX := float32(theme.CanvasW)/2 - pillW/2

	// ── Pill background + border ──
	draw.RoundRect(screen, pillX, pillY, pillW, pillH, pillR, theme.ActionBarBg)
	draw.StrokeRoundRect(screen, pillX, pillY, pillW, pillH, pillR, 1, theme.ActionBarBorder)

	// ── Buttons (centered inside pill) ──
	btnArea := ui.Rect{
		X: pillX + pillPadX,
		Y: pillY + (pillH-btnH)/2,
		W: totalBtnW,
		H: btnH,
	}

	result := ui.DrawButtonRowAutoWidth(screen, btnArea, items, ui.ButtonRowStyle{
		Height:   btnH,
		Gap:      btnGap,
		Radius:   btnR,
		FontSize: theme.FontH2,
	})
	lastActionBarBtnRects = result.Rects
	lastActionBarBtnNames = names
	lastActionBarPillRect = ui.Rect{X: pillX, Y: pillY, W: pillW, H: pillH}
}

// ── Button hit detection (reuses Rects from last DrawActionBar call) ──

var lastActionBarBtnRects []ui.Rect
var lastActionBarBtnNames []string
var lastActionBarPillRect ui.Rect

// ActionBarHitTest returns the button name hit by (px, py), or "" if none.
// Uses the Rects computed by the last DrawActionBar call.
func ActionBarHitTest(px, py float32) string {
	idx := ui.HitTestButtonRow(lastActionBarBtnRects, float64(px), float64(py))
	if idx >= 0 && idx < len(lastActionBarBtnNames) {
		return lastActionBarBtnNames[idx]
	}
	return ""
}

// ActionBarItemBtnCenter returns the screen-space center of the "items" button.
func ActionBarItemBtnCenter() (float32, float32) {
	for i, name := range lastActionBarBtnNames {
		if name == "items" && i < len(lastActionBarBtnRects) {
			r := lastActionBarBtnRects[i]
			return r.X + r.W/2, r.Y + r.H/2
		}
	}
	return float32(theme.CanvasW) / 2, float32(theme.CanvasH) - 20
}

// ActionBarRect returns the bounding rectangle of the action bar pill.
func ActionBarRect() (x, y, w, h float32) {
	r := lastActionBarPillRect
	return r.X, r.Y, r.W, r.H
}
