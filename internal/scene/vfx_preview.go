// vfx_preview.go — VFX preview scene.
// Standalone scene for previewing all visual effects: particles, screen effects,
// post-processing, beams, lightning, float text, wave announcements, etc.
// Accessible from the TestSelect scene via the "vfx-preview" scenario entry.
package scene

import (
	"image/color"
	"math"

	"defense2/internal/core/combat"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/particle"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Effect catalog types ────────────────────────────

type vfxCategory struct {
	Name    string
	Effects []vfxEntry
}

type vfxEntry struct {
	Name    string
	Trigger func(s *VFXPreviewScene)
}

// ── Layout constants ────────────────────────────────

const (
	vfxPanelW     = 260.0 // left panel width
	vfxCtrlH      = 50.0  // bottom control bar height
	vfxCatH       = 26.0  // category header height
	vfxItemH      = 22.0  // effect item height
	vfxPadX       = 12.0  // horizontal padding
	vfxPadY       = 8.0   // vertical padding inside panel
	vfxScrollStep = 40.0  // pixels per scroll wheel tick
)

// ── VFXPreviewScene ─────────────────────────────────

// VFXPreviewScene allows previewing all visual effects in isolation.
type VFXPreviewScene struct {
	switcher     Switcher
	particlePool *particle.Pool
	postPipeline *postprocess.Pipeline
	effects      *postprocess.Effects
	beamPool     *combat.BeamPool
	waveAnnounce *hud.WaveAnnounce

	categories  []vfxCategory
	catIdx      int // selected category index
	effectIdx   int // selected effect index within category
	hoverCat    int
	hoverEffect int

	// Preview state
	replayDelay float64 // countdown before auto-replay
	speed       float64 // dt multiplier (0.5, 1.0, 2.0)
	autoReplay  bool

	// Panel scroll
	scrollY    float64 // vertical scroll offset (pixels, positive = scrolled down)
	maxScrollY float64 // computed max scroll

	// Desaturation auto-reset timer
	desatResetTimer float64
}

// NewVFXPreviewScene creates the VFX preview scene.
func NewVFXPreviewScene(sw Switcher) *VFXPreviewScene {
	pp := postprocess.NewPipeline()
	s := &VFXPreviewScene{
		switcher:     sw,
		particlePool: particle.NewPool(),
		postPipeline: pp,
		effects:      pp.Effects,
		beamPool:     combat.NewBeamPool(),
		waveAnnounce: hud.NewWaveAnnounce(),
		speed:        1.0,
		hoverCat:     -1,
		hoverEffect:  -1,
	}
	// Enable shake for this scene.
	render.SetShakeEnabled(true)
	s.buildCatalog()
	return s
}

// ── Catalog construction ────────────────────────────

func (s *VFXPreviewScene) buildCatalog() {
	cx, cy := s.previewCenter()

	s.categories = []vfxCategory{
		{Name: "Particles", Effects: []vfxEntry{
			{"DeathBurst", func(s *VFXPreviewScene) { particle.EmitDeathBurst(s.particlePool, cx, cy) }},
			{"DeathBurstLarge", func(s *VFXPreviewScene) { particle.EmitDeathBurstLarge(s.particlePool, cx, cy) }},
			{"BossDeathBurst", func(s *VFXPreviewScene) { particle.EmitBossDeathBurst(s.particlePool, cx, cy) }},
			{"MuzzleFlash", func(s *VFXPreviewScene) { particle.EmitMuzzleFlash(s.particlePool, cx, cy, 0) }},
			{"SpawnBurst", func(s *VFXPreviewScene) { particle.EmitSpawnBurst(s.particlePool, cx, cy) }},
			{"FireParticles", func(s *VFXPreviewScene) { particle.EmitFireParticles(s.particlePool, cx, cy, 12) }},
			{"IceParticles", func(s *VFXPreviewScene) { particle.EmitIceParticles(s.particlePool, cx, cy, 12) }},
			{"GoldCollect", func(s *VFXPreviewScene) { particle.EmitGoldCollect(s.particlePool, cx, cy) }},
			{"ElectricSparks", func(s *VFXPreviewScene) { particle.EmitElectricSparks(s.particlePool, cx, cy, 8) }},
			{"BleedDrip", func(s *VFXPreviewScene) { particle.EmitBleedDrip(s.particlePool, cx, cy, 10) }},
		}},
		{Name: "Screen Effects", Effects: []vfxEntry{
			{"ScreenShake", func(s *VFXPreviewScene) { render.TriggerShake(4.0, 0.4) }},
			{"HitFlash", func(s *VFXPreviewScene) { s.effects.TriggerHitFlash(0.15) }},
			{"RadialBlur", func(s *VFXPreviewScene) {
				s.effects.TriggerRadialBlur(cx, cy, 0.04, 0.5)
			}},
			{"Ripple", func(s *VFXPreviewScene) { s.effects.TriggerRipple(cx, cy, 15) }},
			{"HitStop (5f)", func(s *VFXPreviewScene) { s.effects.TriggerHitStop(5) }},
			{"Desaturation", func(s *VFXPreviewScene) {
				s.effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
				s.desatResetTimer = 2.0 // auto-reset after 2s
			}},
		}},
		{Name: "Combat VFX", Effects: []vfxEntry{
			{"ImpactRing", func(s *VFXPreviewScene) { render.SpawnHitImpact(cx, cy) }},
			{"ThunderBolt", func(s *VFXPreviewScene) {
				render.SpawnThunderBolt(cx-50, cy-60, cx+50, cy+60, true)
			}},
			{"Beam (purple)", func(s *VFXPreviewScene) {
				s.beamPool.Add(combat.Beam{
					X1: cx - 80, Y1: cy, X2: cx + 80, Y2: cy,
					Width: 4, Color: [3]uint8{180, 100, 255},
					Life: 0.5, MaxLife: 0.5, Wide: true,
				})
			}},
			{"DamageText", func(s *VFXPreviewScene) { render.SpawnDamageText(cx, cy, 1234, false, false) }},
			{"CritText", func(s *VFXPreviewScene) { render.SpawnDamageText(cx, cy, 5678, true, false) }},
			{"GoldText", func(s *VFXPreviewScene) { render.SpawnGoldText(cx, cy, 100) }},
			{"KillText", func(s *VFXPreviewScene) { render.SpawnKillText(cx, cy) }},
		}},
		{Name: "Post-Processing", Effects: []vfxEntry{
			{"Vignette Strong", func(s *VFXPreviewScene) { s.effects.VignetteStrength = 0.8 }},
			{"Vignette Off", func(s *VFXPreviewScene) { s.effects.VignetteStrength = 0 }},
			{"DynamicLight", func(s *VFXPreviewScene) {
				s.postPipeline.Lighting.Clear()
				s.postPipeline.Lighting.AddLight(postprocess.PointLight{
					X: cx, Y: cy,
					Color:     color.RGBA{R: 255, G: 180, B: 80, A: 255},
					Radius:    200,
					Intensity: 1.5,
				})
			}},
			{"Glow Layer", func(s *VFXPreviewScene) {
				// Glow is drawn in Draw(), trigger via particle burst so there's something to glow.
				particle.EmitBossDeathBurst(s.particlePool, cx, cy)
			}},
		}},
		{Name: "Composite", Effects: []vfxEntry{
			{"BossKill", func(s *VFXPreviewScene) {
				render.TriggerShake(6.0, 0.5)
				s.effects.TriggerRadialBlur(cx, cy, 0.05, 0.6)
				s.effects.TriggerRipple(cx, cy, 18)
				s.effects.TriggerHitStop(8)
				s.effects.TriggerHitFlash(0.1)
				particle.EmitBossDeathBurst(s.particlePool, cx, cy)
				render.SpawnHitImpact(cx, cy)
			}},
			{"MultiKill x10", func(s *VFXPreviewScene) {
				render.TriggerShake(3.0, 0.3)
				particle.EmitDeathBurstLarge(s.particlePool, cx, cy)
				render.SpawnKillText(cx, cy)
				render.SpawnDamageText(cx, cy-20, 9999, true, false)
			}},
			{"WaveAnnounce Normal", func(s *VFXPreviewScene) {
				s.waveAnnounce.Trigger(3, 20, false)
			}},
			{"WaveAnnounce Boss", func(s *VFXPreviewScene) {
				s.waveAnnounce.Trigger(5, 20, true)
			}},
		}},
	}
}

// previewCenter returns the center of the preview area in logical coords.
func (s *VFXPreviewScene) previewCenter() (float64, float64) {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	cx := vfxPanelW + (sw-vfxPanelW)/2
	cy := (sh - vfxCtrlH) / 2
	return cx, cy
}

// ── Update ──────────────────────────────────────────

func (s *VFXPreviewScene) Update() error {
	dt := 1.0 / float64(game.TargetTPS)
	effectiveDT := dt * s.speed

	// Keyboard shortcuts.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.triggerCurrent()
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.speed = 0.5
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.speed = 1.0
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.speed = 2.0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		s.navigateEffect(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		s.navigateEffect(1)
	}

	// Mouse scroll for panel.
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.scrollY -= wy * vfxScrollStep
		s.clampScroll()
	}

	// Mouse input.
	mxf, myf := draw.CursorPos()
	s.updateHover(mxf, myf)
	if isTapJustPressed() {
		s.handleClick(mxf, myf)
	}

	// Update subsystems.
	frozen := s.effects.Update(effectiveDT)
	if !frozen {
		s.particlePool.Update(effectiveDT)
		s.beamPool.Update(effectiveDT)
		render.UpdateShake(effectiveDT)
		render.UpdateImpactVFX(effectiveDT)
		render.UpdateThunderBolts(effectiveDT)
		render.UpdateFloatTexts(effectiveDT)
		s.waveAnnounce.Update(effectiveDT)
		hud.UpdateToast(effectiveDT)
	}

	// Desaturation auto-reset.
	if s.desatResetTimer > 0 {
		s.desatResetTimer -= effectiveDT
		if s.desatResetTimer <= 0 {
			s.effects.SetDesaturation(0, 3.0, 0, 0, 0)
		}
	}

	// Auto-replay logic.
	if s.autoReplay && s.isEffectIdle() {
		s.replayDelay -= dt
		if s.replayDelay <= 0 {
			s.triggerCurrent()
		}
	}

	return nil
}

// isEffectIdle returns true when no active effects are playing.
func (s *VFXPreviewScene) isEffectIdle() bool {
	if s.particlePool.ActiveCount() > 0 {
		return false
	}
	if s.beamPool.Count() > 0 {
		return false
	}
	if s.effects.HitFlash.Active || s.effects.RadialBlur.Active || s.effects.HitStopFrames > 0 {
		return false
	}
	return true
}

// triggerCurrent fires the currently selected effect.
func (s *VFXPreviewScene) triggerCurrent() {
	if s.catIdx < 0 || s.catIdx >= len(s.categories) {
		return
	}
	cat := &s.categories[s.catIdx]
	if s.effectIdx < 0 || s.effectIdx >= len(cat.Effects) {
		return
	}
	cat.Effects[s.effectIdx].Trigger(s)
	s.replayDelay = 0.8
}

// navigateEffect moves the effect selection up or down.
func (s *VFXPreviewScene) navigateEffect(delta int) {
	if len(s.categories) == 0 {
		return
	}
	cat := &s.categories[s.catIdx]
	s.effectIdx += delta
	if s.effectIdx < 0 {
		// Wrap to previous category.
		s.catIdx--
		if s.catIdx < 0 {
			s.catIdx = len(s.categories) - 1
		}
		s.effectIdx = len(s.categories[s.catIdx].Effects) - 1
	} else if s.effectIdx >= len(cat.Effects) {
		// Wrap to next category.
		s.catIdx++
		if s.catIdx >= len(s.categories) {
			s.catIdx = 0
		}
		s.effectIdx = 0
	}
}

// ── Mouse handling ──────────────────────────────────

func (s *VFXPreviewScene) updateHover(mx, my float64) {
	s.hoverCat = -1
	s.hoverEffect = -1

	// Only process hover in the left panel area.
	if mx < 0 || mx > vfxPanelW {
		return
	}
	sh := float64(game.ScreenHeight)
	if my > sh-vfxCtrlH {
		return
	}

	y := vfxPadY - s.scrollY
	for ci, cat := range s.categories {
		// Category header.
		if my >= y && my < y+vfxCatH {
			s.hoverCat = ci
			return
		}
		y += vfxCatH
		// Effect items.
		for ei := range cat.Effects {
			if my >= y && my < y+vfxItemH && mx >= vfxPadX && mx <= vfxPanelW-vfxPadX {
				s.hoverCat = ci
				s.hoverEffect = ei
				return
			}
			y += vfxItemH
		}
		y += vfxPadY // gap between categories
	}
}

func (s *VFXPreviewScene) handleClick(mx, my float64) {
	sh := float64(game.ScreenHeight)
	sw := float64(game.ScreenWidth)

	// Bottom control bar buttons.
	if my >= sh-vfxCtrlH {
		s.handleControlClick(mx, my, sw, sh)
		return
	}

	// Left panel.
	if mx <= vfxPanelW {
		s.handlePanelClick(mx, my)
		return
	}
}

func (s *VFXPreviewScene) handlePanelClick(mx, my float64) {
	y := vfxPadY - s.scrollY
	for ci, cat := range s.categories {
		y += vfxCatH // skip header
		for ei := range cat.Effects {
			if my >= y && my < y+vfxItemH && mx >= vfxPadX && mx <= vfxPanelW-vfxPadX {
				s.catIdx = ci
				s.effectIdx = ei
				playUIClick(s.switcher)
				s.triggerCurrent()
				return
			}
			y += vfxItemH
		}
		y += vfxPadY
	}
}

func (s *VFXPreviewScene) handleControlClick(mx, my, sw, sh float64) {
	// Control bar layout: [Back] [Replay] [0.5x] [1x] [2x] [Auto]
	btnW := 64.0
	btnH := 30.0
	gap := 10.0
	baseY := sh - vfxCtrlH + (vfxCtrlH-btnH)/2

	// Center the button group in the full width.
	totalBtns := 6.0
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
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
	case 1: // Replay
		s.triggerCurrent()
	case 2: // 0.5x
		s.speed = 0.5
	case 3: // 1x
		s.speed = 1.0
	case 4: // 2x
		s.speed = 2.0
	case 5: // Auto toggle
		s.autoReplay = !s.autoReplay
	}
}

func (s *VFXPreviewScene) clampScroll() {
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	if s.scrollY > s.maxScrollY {
		s.scrollY = s.maxScrollY
	}
}

// ── Draw ────────────────────────────────────────────

func (s *VFXPreviewScene) Draw(screen *ebiten.Image) {
	fm := render.GlobalFont()
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// Get scene buffer for post-processing.
	physW, physH := screen.Bounds().Dx(), screen.Bounds().Dy()
	buf := s.postPipeline.SceneBuffer(physW, physH)

	// Fill preview area background (very dark).
	draw.FilledRect(buf, float32(vfxPanelW), 0,
		float32(sw-vfxPanelW), float32(sh-vfxCtrlH),
		color.RGBA{R: 15, G: 18, B: 25, A: 255}, false)

	// Apply shake offset for preview area content.
	dx, dy := render.ShakeOffset()

	// Draw crosshair at preview center.
	cx, cy := s.previewCenter()
	crossClr := color.RGBA{R: 255, G: 255, B: 255, A: 25}
	draw.Line(buf, float32(cx+dx)-20, float32(cy+dy), float32(cx+dx)+20, float32(cy+dy), 0.5, crossClr, false)
	draw.Line(buf, float32(cx+dx), float32(cy+dy)-20, float32(cx+dx), float32(cy+dy)+20, 0.5, crossClr, false)

	// Glow pass wraps particle + beam rendering.
	draw.BeginGlowPass(buf)

	// Draw particles (with shake offset applied via translation).
	s.particlePool.Draw(buf)

	// Draw beams.
	render.DrawBeams(buf, s.beamPool)

	draw.EndGlowPass(buf)

	// Draw thunder bolts.
	render.DrawThunderBolts(buf)

	// Draw impact VFX.
	render.DrawImpactVFX(buf)

	// Draw float text.
	render.DrawFloatTexts(buf)

	// Apply post-processing pipeline (vignette, color grade, radial blur, etc.).
	s.postPipeline.Apply(screen)

	// ── Left panel (drawn on top of post-processed output) ──
	// Panel background.
	draw.FilledRect(screen, 0, 0, float32(vfxPanelW), float32(sh-vfxCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Panel right border.
	draw.FilledRect(screen, float32(vfxPanelW)-1, 0, 1, float32(sh-vfxCtrlH),
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	s.drawCatalog(screen, fm, sh)

	// ── Bottom control bar ──
	draw.FilledRect(screen, 0, float32(sh-vfxCtrlH), float32(sw), float32(vfxCtrlH),
		color.RGBA{R: 12, G: 15, B: 22, A: 240}, false)
	// Top border.
	draw.FilledRect(screen, 0, float32(sh-vfxCtrlH), float32(sw), 1,
		color.RGBA{R: 255, G: 255, B: 255, A: 15}, false)

	s.drawControls(screen, fm, sw, sh)

	// ── Wave announce overlay (topmost) ──
	s.waveAnnounce.Draw(screen)
	hud.DrawToast(screen)
}

// drawCatalog renders the left panel category/effect list.
func (s *VFXPreviewScene) drawCatalog(screen *ebiten.Image, fm *render.FontManager, sh float64) {
	if fm == nil {
		return
	}

	panelBottom := sh - vfxCtrlH
	y := vfxPadY - s.scrollY
	totalH := 0.0

	for ci, cat := range s.categories {
		// Category header.
		if y+vfxCatH > 0 && y < panelBottom {
			headerClr := color.RGBA{R: 120, G: 140, B: 180, A: 255}
			if ci == s.hoverCat && s.hoverEffect == -1 {
				headerClr = color.RGBA{R: 160, G: 180, B: 220, A: 255}
			}
			fm.DrawBoldText(screen, cat.Name, vfxPadX, y+4, 12, headerClr)
		}
		y += vfxCatH

		// Effect items.
		for ei, eff := range cat.Effects {
			if y+vfxItemH > 0 && y < panelBottom {
				selected := ci == s.catIdx && ei == s.effectIdx
				hovered := ci == s.hoverCat && ei == s.hoverEffect

				// Item background.
				if selected {
					draw.FilledRect(screen, float32(vfxPadX-4), float32(y),
						float32(vfxPanelW-2*vfxPadX+8), float32(vfxItemH),
						color.RGBA{R: 60, G: 80, B: 140, A: 160}, false)
				} else if hovered {
					draw.FilledRect(screen, float32(vfxPadX-4), float32(y),
						float32(vfxPanelW-2*vfxPadX+8), float32(vfxItemH),
						color.RGBA{R: 40, G: 50, B: 80, A: 120}, false)
				}

				// Item text.
				textClr := theme.TextBody
				if selected {
					textClr = color.RGBA{R: 200, G: 220, B: 255, A: 255}
				}
				fm.DrawText(screen, eff.Name, vfxPadX+8, y+4, 11, textClr)

				// Selected indicator dot.
				if selected {
					draw.FilledCircle(screen, float32(vfxPadX), float32(y+vfxItemH/2),
						2, color.RGBA{R: 100, G: 160, B: 255, A: 255})
				}
			}
			y += vfxItemH
		}
		y += vfxPadY
	}

	// Compute max scroll.
	totalH = y + s.scrollY
	s.maxScrollY = totalH - panelBottom
	if s.maxScrollY < 0 {
		s.maxScrollY = 0
	}
}

// drawControls renders the bottom control bar buttons.
func (s *VFXPreviewScene) drawControls(screen *ebiten.Image, fm *render.FontManager, sw, sh float64) {
	if fm == nil {
		return
	}

	btnW := 64.0
	btnH := 30.0
	gap := 10.0
	baseY := sh - vfxCtrlH + (vfxCtrlH-btnH)/2

	type ctrlBtn struct {
		Label  string
		Active bool // toggle/highlight state
	}
	buttons := []ctrlBtn{
		{"Back", false},
		{"Replay", false},
		{"0.5x", s.speed == 0.5},
		{"1x", s.speed == 1.0},
		{"2x", s.speed == 2.0},
		{vfxAutoLabel(s.autoReplay), s.autoReplay},
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

	// Current speed indicator.
	speedTxt := vfxSpeedLabel(s.speed)
	fm.DrawText(screen, speedTxt, startX+totalW+20, baseY+7, 11, theme.TextMuted)

	// Effect name display.
	if s.catIdx >= 0 && s.catIdx < len(s.categories) {
		cat := &s.categories[s.catIdx]
		if s.effectIdx >= 0 && s.effectIdx < len(cat.Effects) {
			label := cat.Name + " > " + cat.Effects[s.effectIdx].Name
			fm.DrawText(screen, label, vfxPanelW+12, sh-vfxCtrlH+vfxCtrlH/2-6, 11, theme.TextMuted)
		}
	}
}

// ── Helpers ─────────────────────────────────────────

func vfxAutoLabel(on bool) string {
	if on {
		return "Auto ON"
	}
	return "Auto"
}

func vfxSpeedLabel(spd float64) string {
	switch {
	case math.Abs(spd-0.5) < 0.01:
		return "Speed: 0.5x"
	case math.Abs(spd-2.0) < 0.01:
		return "Speed: 2.0x"
	default:
		return "Speed: 1.0x"
	}
}
