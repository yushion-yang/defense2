// audio_preview.go — Audio preview scene.
// Standalone scene for previewing all audio (SFX + BGM) in the game.
// Accessible from the TestSelect scene via the "audio-preview" scenario entry.
package scene

import (
	"encoding/json"
	"image/color"
	"log"
	"math"
	"strings"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Audio catalog types ────────────────────────────

type audioCategory struct {
	Name    string
	Entries []audioEntry
}

type audioEntry struct {
	Name string // display name from JSON
	Key  string // camelCase key for PlaySafe (SFX) or raw key for PlayBGM (BGM)
	File string // original filename
	IsBGM bool  // BGM uses PlayBGM instead of PlaySafe
}

// ── Layout constants (match VFX Preview) ───────────

const (
	apPanelW     = 260.0 // left panel width
	apCtrlH      = 50.0  // bottom control bar height
	apCatH       = 26.0  // category header height
	apItemH      = 22.0  // item height
	apPadX       = 12.0  // horizontal padding
	apPadY       = 8.0   // vertical padding inside panel
	apScrollStep = 40.0  // pixels per scroll wheel tick
)

// ── AudioPreviewScene ─────────────────────────────

// AudioPreviewScene allows previewing all audio assets in isolation.
type AudioPreviewScene struct {
	switcher Switcher
	audioMgr *gameAudio.Manager

	categories []audioCategory
	catIdx     int // selected category index
	entryIdx   int // selected entry index within category
	hoverCat   int
	hoverEntry int

	// Playback state
	playingKey  string  // currently playing SFX/BGM key ("" = nothing)
	playingBGM  bool    // true if playing a BGM track
	playTime    float64 // accumulated time since play started
	statusDots  float64 // animated dots timer

	// Volume
	previewVol float64 // preview volume (0.0 ~ 1.0)

	// Panel scroll
	scrollY    float64
	maxScrollY float64
}

// NewAudioPreviewScene creates the audio preview scene.
func NewAudioPreviewScene(sw Switcher) *AudioPreviewScene {
	s := &AudioPreviewScene{
		switcher:   sw,
		audioMgr:   sw.AudioManager(),
		previewVol: 0.8,
		hoverCat:   -1,
		hoverEntry: -1,
	}
	s.buildCatalog()
	return s
}

// ── Catalog construction ────────────────────────────

// rawSFXConfig mirrors the relevant parts of config/audio/sfx.json.
type rawSFXConfig struct {
	Categories []rawSFXCategory `json:"categories"`
}

type rawSFXCategory struct {
	ID   string      `json:"id"`
	Name string      `json:"name"`
	SFX  []rawSFXDef `json:"sfx"`
}

type rawSFXDef struct {
	File string `json:"file"`
	Name string `json:"name"`
}

// rawBGMConfig mirrors the relevant parts of config/audio/bgm.json.
type rawBGMConfig struct {
	Tracks []rawBGMTrack `json:"tracks"`
}

type rawBGMTrack struct {
	File string `json:"file"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// kebabToCamelLocal converts kebab-case to camelCase (same logic as audio.kebabToCamel).
func kebabToCamelLocal(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) <= 1 {
		return s
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

func (s *AudioPreviewScene) buildCatalog() {
	fs := config.GetDataFS()
	if fs == nil {
		log.Println("audio_preview: dataFS not initialized, using empty catalog")
		return
	}

	// Load BGM tracks first.
	s.loadBGMCatalog(fs)

	// Load SFX categories.
	s.loadSFXCatalog(fs)
}

func (s *AudioPreviewScene) loadBGMCatalog(fs interface{ ReadFile(string) ([]byte, error) }) {
	data, err := fs.ReadFile("config/audio/bgm.json")
	if err != nil {
		log.Printf("audio_preview: failed to read bgm.json: %v", err)
		return
	}
	var bgmCfg rawBGMConfig
	if err := json.Unmarshal(data, &bgmCfg); err != nil {
		log.Printf("audio_preview: failed to parse bgm.json: %v", err)
		return
	}

	cat := audioCategory{Name: "BGM - 背景音乐"}
	for _, t := range bgmCfg.Tracks {
		cat.Entries = append(cat.Entries, audioEntry{
			Name:  t.Name,
			Key:   t.Key,
			File:  t.File,
			IsBGM: true,
		})
	}
	if len(cat.Entries) > 0 {
		s.categories = append(s.categories, cat)
	}
}

func (s *AudioPreviewScene) loadSFXCatalog(fs interface{ ReadFile(string) ([]byte, error) }) {
	data, err := fs.ReadFile("config/audio/sfx.json")
	if err != nil {
		log.Printf("audio_preview: failed to read sfx.json: %v", err)
		return
	}
	var sfxCfg rawSFXConfig
	if err := json.Unmarshal(data, &sfxCfg); err != nil {
		log.Printf("audio_preview: failed to parse sfx.json: %v", err)
		return
	}

	for _, cfgCat := range sfxCfg.Categories {
		cat := audioCategory{Name: cfgCat.Name}
		for _, sfx := range cfgCat.SFX {
			if sfx.File == "" || sfx.Name == "" {
				continue
			}
			// Derive camelCase key from filename (same logic as audio loader).
			baseName := strings.TrimSuffix(sfx.File, ".wav")
			key := kebabToCamelLocal(baseName)
			cat.Entries = append(cat.Entries, audioEntry{
				Name: sfx.Name,
				Key:  key,
				File: sfx.File,
			})
		}
		if len(cat.Entries) > 0 {
			s.categories = append(s.categories, cat)
		}
	}
}

// previewCenter returns the center of the preview area in logical coords.
func (s *AudioPreviewScene) previewCenter() (float64, float64) {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	cx := apPanelW + (sw-apPanelW)/2
	cy := (sh - apCtrlH) / 2
	return cx, cy
}

// ── Update ──────────────────────────────────────────

func (s *AudioPreviewScene) Update() error {
	dt := 1.0 / float64(game.TargetTPS)

	// Keyboard shortcuts.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.stopPlaying()
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.togglePlay()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		s.navigateEntry(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		s.navigateEntry(1)
	}

	// Mouse scroll for panel.
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.scrollY -= wy * apScrollStep
		s.clampScroll()
	}

	// Mouse input.
	mxf, myf := draw.CursorPos()
	s.updateHover(mxf, myf)
	if isTapJustPressed() {
		s.handleClick(mxf, myf)
	}

	// Update play timer.
	if s.playingKey != "" {
		s.playTime += dt
		s.statusDots += dt
	}

	return nil
}

// togglePlay plays or stops the currently selected entry.
func (s *AudioPreviewScene) togglePlay() {
	if len(s.categories) == 0 {
		return
	}
	// If the current selection is already playing, stop it.
	entry := s.currentEntry()
	if entry == nil {
		return
	}
	if s.playingKey == entry.Key {
		s.stopPlaying()
		return
	}
	s.playCurrent()
}

// playCurrent starts playing the currently selected audio entry.
func (s *AudioPreviewScene) playCurrent() {
	entry := s.currentEntry()
	if entry == nil {
		return
	}

	// Stop any current playback first.
	s.stopPlaying()

	s.playingKey = entry.Key
	s.playingBGM = entry.IsBGM
	s.playTime = 0
	s.statusDots = 0

	if s.audioMgr == nil {
		return
	}

	if entry.IsBGM {
		s.audioMgr.PlayBGM(entry.Key)
	} else {
		s.audioMgr.PlaySafeAt(entry.Key, s.previewVol)
	}
}

// stopPlaying stops any current playback.
func (s *AudioPreviewScene) stopPlaying() {
	if s.playingBGM && s.audioMgr != nil {
		s.audioMgr.StopBGM()
	}
	s.playingKey = ""
	s.playingBGM = false
	s.playTime = 0
}

// currentEntry returns the currently selected audioEntry, or nil.
func (s *AudioPreviewScene) currentEntry() *audioEntry {
	if s.catIdx < 0 || s.catIdx >= len(s.categories) {
		return nil
	}
	cat := &s.categories[s.catIdx]
	if s.entryIdx < 0 || s.entryIdx >= len(cat.Entries) {
		return nil
	}
	return &cat.Entries[s.entryIdx]
}

// navigateEntry moves the entry selection up or down (wraps across categories).
func (s *AudioPreviewScene) navigateEntry(delta int) {
	if len(s.categories) == 0 {
		return
	}
	cat := &s.categories[s.catIdx]
	s.entryIdx += delta
	if s.entryIdx < 0 {
		// Wrap to previous category.
		s.catIdx--
		if s.catIdx < 0 {
			s.catIdx = len(s.categories) - 1
		}
		s.entryIdx = len(s.categories[s.catIdx].Entries) - 1
	} else if s.entryIdx >= len(cat.Entries) {
		// Wrap to next category.
		s.catIdx++
		if s.catIdx >= len(s.categories) {
			s.catIdx = 0
		}
		s.entryIdx = 0
	}
}

// ── Mouse handling ──────────────────────────────────

func (s *AudioPreviewScene) updateHover(mx, my float64) {
	s.hoverCat = -1
	s.hoverEntry = -1

	if mx < 0 || mx > apPanelW {
		return
	}
	sh := float64(game.ScreenHeight)
	if my > sh-apCtrlH {
		return
	}

	y := apPadY - s.scrollY
	for ci, cat := range s.categories {
		if my >= y && my < y+apCatH {
			s.hoverCat = ci
			return
		}
		y += apCatH
		for ei := range cat.Entries {
			if my >= y && my < y+apItemH && mx >= apPadX && mx <= apPanelW-apPadX {
				s.hoverCat = ci
				s.hoverEntry = ei
				return
			}
			y += apItemH
		}
		y += apPadY
	}
}

func (s *AudioPreviewScene) handleClick(mx, my float64) {
	sh := float64(game.ScreenHeight)
	sw := float64(game.ScreenWidth)

	// Bottom control bar.
	if my >= sh-apCtrlH {
		s.handleControlClick(mx, my, sw, sh)
		return
	}

	// Left panel.
	if mx <= apPanelW {
		s.handlePanelClick(mx, my)
		return
	}
}

func (s *AudioPreviewScene) handlePanelClick(mx, my float64) {
	y := apPadY - s.scrollY
	for ci, cat := range s.categories {
		y += apCatH // skip header
		for ei := range cat.Entries {
			if my >= y && my < y+apItemH && mx >= apPadX && mx <= apPanelW-apPadX {
				s.catIdx = ci
				s.entryIdx = ei
				playUIClick(s.switcher)
				s.playCurrent()
				return
			}
			y += apItemH
		}
		y += apPadY
	}
}

func (s *AudioPreviewScene) handleControlClick(mx, my, sw, sh float64) {
	// Control bar layout: [Back] [Play/Stop] [Vol-] [Vol+]
	btnW := 72.0
	btnH := 30.0
	gap := 10.0
	baseY := sh - apCtrlH + (apCtrlH-btnH)/2

	totalBtns := 4.0
	totalW := totalBtns*btnW + (totalBtns-1)*gap
	startX := (sw - totalW) / 2

	btnIdx := -1
	for i := 0; i < int(totalBtns); i++ {
		bx := startX + float64(i)*(btnW+gap)
		if mx >= bx && mx <= bx+btnW && my >= baseY && my <= baseY+btnH {
			btnIdx = i
			break
		}
	}
	if btnIdx < 0 {
		return
	}

	playUIClick(s.switcher)
	switch btnIdx {
	case 0: // Back
		s.stopPlaying()
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
	case 1: // Play/Stop toggle
		s.togglePlay()
	case 2: // Vol-
		s.previewVol -= 0.1
		if s.previewVol < 0 {
			s.previewVol = 0
		}
	case 3: // Vol+
		s.previewVol += 0.1
		if s.previewVol > 1.0 {
			s.previewVol = 1.0
		}
	}
}

func (s *AudioPreviewScene) clampScroll() {
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	if s.scrollY > s.maxScrollY {
		s.scrollY = s.maxScrollY
	}
}

// ── Draw ────────────────────────────────────────────

func (s *AudioPreviewScene) Draw(screen *ebiten.Image) {
	fm := render.GlobalFont()
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// Fill full background.
	draw.FilledRect(screen, 0, 0, float32(sw), float32(sh),
		color.RGBA{R: 12, G: 15, B: 22, A: 255}, false)

	// Preview area background (right side, slightly lighter).
	draw.FilledRect(screen, float32(apPanelW), 0,
		float32(sw-apPanelW), float32(sh-apCtrlH),
		color.RGBA{R: 15, G: 18, B: 25, A: 255}, false)

	// Draw preview area content.
	s.drawPreviewArea(screen, fm, sw, sh)

	// ── Left panel ──
	draw.FilledRect(screen, 0, 0, float32(apPanelW), float32(sh-apCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Right border.
	draw.FilledRect(screen, float32(apPanelW)-1, 0, 1, float32(sh-apCtrlH),
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	s.drawCatalog(screen, fm, sh)

	// ── Bottom control bar ──
	draw.FilledRect(screen, 0, float32(sh-apCtrlH), float32(sw), float32(apCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Top border.
	draw.FilledRect(screen, 0, float32(sh-apCtrlH), float32(sw), 1,
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	s.drawControls(screen, fm, sw, sh)
}

// drawPreviewArea renders the central info display for the selected sound.
func (s *AudioPreviewScene) drawPreviewArea(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	if fm == nil {
		return
	}

	cx, _ := s.previewCenter()
	areaTop := 60.0
	entry := s.currentEntry()

	if entry == nil {
		// No selection — show prompt.
		fm.DrawCenteredText(screen, "选择左侧音效进行试听", cx, sh/2-20, 14, theme.TextMuted)
		fm.DrawCenteredText(screen, "Space: 播放/停止  Up/Down: 导航  Esc: 返回", cx, sh/2+10, 11, theme.TextLocked)
		return
	}

	// Sound name (large).
	fm.DrawCenteredText(screen, entry.Name, cx, areaTop, 20, theme.TextTitle)

	// File name.
	fm.DrawCenteredText(screen, entry.File, cx, areaTop+32, 12, theme.TextMuted)

	// Key name.
	keyLabel := "Key: " + entry.Key
	if entry.IsBGM {
		keyLabel += "  (BGM)"
	}
	fm.DrawCenteredText(screen, keyLabel, cx, areaTop+52, 11, theme.TextLocked)

	// Status indicator.
	statusY := areaTop + 90
	if s.playingKey == entry.Key {
		// Playing — animated dots.
		dotCount := int(math.Mod(s.statusDots*3, 4))
		dots := strings.Repeat(".", dotCount)
		statusText := "Playing" + dots
		statusClr := color.RGBA{R: 80, G: 200, B: 120, A: 255}
		fm.DrawCenteredText(screen, statusText, cx, statusY, 16, statusClr)

		// Time display.
		secs := int(s.playTime)
		timeText := apFormatTime(secs)
		fm.DrawCenteredText(screen, timeText, cx, statusY+26, 12, theme.TextMuted)

		// Simple progress bar for BGM (or pulse indicator for SFX).
		barY := float32(statusY + 50)
		barW := float32(200)
		barH := float32(4)
		barX := float32(cx) - barW/2

		// Bar background.
		draw.RoundRect(screen, barX, barY, barW, barH, 2,
			color.RGBA{R: 40, G: 45, B: 60, A: 200})

		if entry.IsBGM {
			// Oscillating progress for BGM (no duration info available).
			p := float32(math.Mod(s.playTime/10.0, 1.0))
			fillW := barW * p
			draw.RoundRect(screen, barX, barY, fillW, barH, 2,
				color.RGBA{R: 80, G: 200, B: 120, A: 200})
		} else {
			// Short pulse for SFX — flash and decay.
			decay := float32(math.Max(0, 1.0-s.playTime/0.5))
			if decay > 0 {
				fillW := barW * decay
				draw.RoundRect(screen, barX, barY, fillW, barH, 2,
					color.RGBA{R: 100, G: 180, B: 255, A: uint8(200 * decay)})
			}
		}
	} else {
		fm.DrawCenteredText(screen, "Stopped", cx, statusY, 16, theme.TextLocked)
	}

	// Volume indicator.
	volY := sh - apCtrlH - 40
	volText := apVolumeLabel(s.previewVol)
	fm.DrawCenteredText(screen, volText, cx, volY, 11, theme.TextMuted)

	// Draw volume bar.
	volBarW := float32(160)
	volBarH := float32(4)
	volBarX := float32(cx) - volBarW/2
	volBarY := float32(volY + 18)
	draw.RoundRect(screen, volBarX, volBarY, volBarW, volBarH, 2,
		color.RGBA{R: 40, G: 45, B: 60, A: 200})
	fillW := volBarW * float32(s.previewVol)
	if fillW > 0 {
		draw.RoundRect(screen, volBarX, volBarY, fillW, volBarH, 2,
			color.RGBA{R: 100, G: 160, B: 255, A: 200})
	}
}

// drawCatalog renders the left panel category/entry list.
func (s *AudioPreviewScene) drawCatalog(screen *ebiten.Image, fm *render.FontManager, sh float64) {
	if fm == nil {
		return
	}

	panelBottom := sh - apCtrlH
	y := apPadY - s.scrollY

	for ci, cat := range s.categories {
		// Category header.
		if y+apCatH > 0 && y < panelBottom {
			headerClr := color.RGBA{R: 120, G: 140, B: 180, A: 255}
			if ci == s.hoverCat && s.hoverEntry == -1 {
				headerClr = color.RGBA{R: 160, G: 180, B: 220, A: 255}
			}
			fm.DrawBoldText(screen, cat.Name, apPadX, y+4, 12, headerClr)
		}
		y += apCatH

		// Entry items.
		for ei, ent := range cat.Entries {
			if y+apItemH > 0 && y < panelBottom {
				selected := ci == s.catIdx && ei == s.entryIdx
				hovered := ci == s.hoverCat && ei == s.hoverEntry

				// Item background.
				if selected {
					draw.FilledRect(screen, float32(apPadX-4), float32(y),
						float32(apPanelW-2*apPadX+8), float32(apItemH),
						color.RGBA{R: 60, G: 80, B: 140, A: 160}, false)
				} else if hovered {
					draw.FilledRect(screen, float32(apPadX-4), float32(y),
						float32(apPanelW-2*apPadX+8), float32(apItemH),
						color.RGBA{R: 40, G: 50, B: 80, A: 120}, false)
				}

				// Item text.
				textClr := theme.TextBody
				if selected {
					textClr = color.RGBA{R: 200, G: 220, B: 255, A: 255}
				}

				// Show a playing indicator if this entry is currently playing.
				prefix := ""
				if s.playingKey == ent.Key {
					prefix = ">> "
					textClr = color.RGBA{R: 80, G: 200, B: 120, A: 255}
				}

				fm.DrawText(screen, prefix+ent.Name, apPadX+8, y+4, 11, textClr)

				// Selected indicator dot.
				if selected {
					draw.FilledCircle(screen, float32(apPadX), float32(y+apItemH/2),
						2, color.RGBA{R: 100, G: 160, B: 255, A: 255})
				}
			}
			y += apItemH
		}
		y += apPadY
	}

	// Compute max scroll.
	totalH := y + s.scrollY
	s.maxScrollY = totalH - panelBottom
	if s.maxScrollY < 0 {
		s.maxScrollY = 0
	}
}

// drawControls renders the bottom control bar buttons.
func (s *AudioPreviewScene) drawControls(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	if fm == nil {
		return
	}

	btnW := 72.0
	btnH := 30.0
	gap := 10.0
	baseY := sh - apCtrlH + (apCtrlH-btnH)/2

	type ctrlBtn struct {
		Label  string
		Active bool
	}

	playLabel := "Play"
	if s.playingKey != "" {
		playLabel = "Stop"
	}

	buttons := []ctrlBtn{
		{"Back", false},
		{playLabel, s.playingKey != ""},
		{"Vol -", false},
		{"Vol +", false},
	}

	totalBtns := float64(len(buttons))
	totalW := totalBtns*btnW + (totalBtns-1)*gap
	startX := (sw - totalW) / 2

	for i, btn := range buttons {
		bx := startX + float64(i)*(btnW+gap)
		bg := theme.BtnSecondary
		if btn.Active {
			bg = theme.BtnPrimary
		}
		draw.RoundRect(screen, float32(bx), float32(baseY), float32(btnW), float32(btnH), 8, bg)
		fm.DrawCenteredText(screen, btn.Label, bx+btnW/2, baseY+7, 11, theme.TextTitle)
	}

	// Entry name display at bottom-left of control bar.
	if entry := s.currentEntry(); entry != nil {
		label := entry.Name + "  (" + entry.File + ")"
		fm.DrawText(screen, label, apPanelW+12, sh-apCtrlH+apCtrlH/2-6, 11, theme.TextMuted)
	}
}

// ── Helpers ─────────────────────────────────────────

func apFormatTime(secs int) string {
	m := secs / 60
	s := secs % 60
	// Manual formatting to avoid fmt import for a simple case.
	ms := "0"
	if m >= 10 {
		ms = string(rune('0'+m/10)) + string(rune('0'+m%10))
	} else {
		ms = string(rune('0' + m))
	}
	ss := ""
	if s < 10 {
		ss = "0" + string(rune('0'+s))
	} else {
		ss = string(rune('0'+s/10)) + string(rune('0'+s%10))
	}
	return ms + ":" + ss
}

func apVolumeLabel(vol float64) string {
	pct := int(vol*100 + 0.5)
	// Convert to string without fmt.
	if pct <= 0 {
		return "Volume: 0%"
	}
	if pct >= 100 {
		return "Volume: 100%"
	}
	s := ""
	if pct >= 10 {
		s = string(rune('0'+pct/10)) + string(rune('0'+pct%10))
	} else {
		s = string(rune('0' + pct))
	}
	return "Volume: " + s + "%"
}
