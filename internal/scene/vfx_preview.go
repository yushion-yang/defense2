// vfx_preview.go — VFX preview scene.
// Standalone scene for previewing all visual effects: particles, screen effects,
// post-processing, beams, lightning, float text, wave announcements, etc.
// Accessible from the TestSelect scene via the "vfx-preview" scenario entry.
package scene

import (
	"image/color"
	"log"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/particle"
	"defense2/internal/render/postprocess"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

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

	// Active VFX state — all effects use continuous per-frame drawing
	activeVFX string  // active vfx id (set by trigger, drawn each frame)
	vfxTime   float64 // accumulated time since trigger (for animation)
	vfxParam  int     // optional mode parameter (e.g. skystrike mode)
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

// vfxTriggerRegistry maps effect ID → trigger function.
func (s *VFXPreviewScene) vfxTriggerRegistry() map[string]func(s *VFXPreviewScene) {
	cx, cy := s.previewCenter()
	sw, sh := float64(game.ScreenWidth), float64(game.ScreenHeight)
	_ = sw
	_ = sh

	return map[string]func(s *VFXPreviewScene){
		// Particles
		"deathBurst":      func(s *VFXPreviewScene) { particle.EmitDeathBurst(s.particlePool, cx, cy) },
		"deathBurstLarge": func(s *VFXPreviewScene) { particle.EmitDeathBurstLarge(s.particlePool, cx, cy) },
		"bossDeathBurst":  func(s *VFXPreviewScene) { particle.EmitBossDeathBurst(s.particlePool, cx, cy) },
		"muzzleFlash":     func(s *VFXPreviewScene) { particle.EmitMuzzleFlash(s.particlePool, cx, cy, 0) },
		"spawnBurst":      func(s *VFXPreviewScene) { particle.EmitSpawnBurst(s.particlePool, cx, cy) },
		"fireParticles":   func(s *VFXPreviewScene) { particle.EmitFireParticles(s.particlePool, cx, cy, 12) },
		"iceParticles":    func(s *VFXPreviewScene) { particle.EmitIceParticles(s.particlePool, cx, cy, 12) },
		"goldCollect":     func(s *VFXPreviewScene) { particle.EmitGoldCollect(s.particlePool, cx, cy) },
		"electricSparks":  func(s *VFXPreviewScene) { particle.EmitElectricSparks(s.particlePool, cx, cy, 8) },
		"bleedDrip":       func(s *VFXPreviewScene) { particle.EmitBleedDrip(s.particlePool, cx, cy, 10) },
		"ambient":         func(s *VFXPreviewScene) { particle.EmitAmbient(s.particlePool, sw, sh) },

		// Screen Effects
		"screenShakeLight":  func(s *VFXPreviewScene) { render.TriggerShake(1.5, 0.1) },
		"screenShakeMedium": func(s *VFXPreviewScene) { render.TriggerShake(2.5, 0.15) },
		"screenShakeHeavy":  func(s *VFXPreviewScene) { render.TriggerShake(5.0, 0.4) },
		"hitFlash":          func(s *VFXPreviewScene) { s.effects.TriggerHitFlash(0.15) },
		"radialBlur":        func(s *VFXPreviewScene) { s.effects.TriggerRadialBlur(cx, cy, 0.04, 0.5) },
		"ripple":            func(s *VFXPreviewScene) { s.effects.TriggerRipple(cx, cy, 15) },
		"hitStop":           func(s *VFXPreviewScene) { s.effects.TriggerHitStop(5) },
		"desaturation": func(s *VFXPreviewScene) {
			s.effects.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)
			s.desatResetTimer = 2.0
		},

		// Combat VFX
		"typedImpactScatter":  func(s *VFXPreviewScene) { render.SpawnTypedImpact(cx, cy, "scatter") },
		"typedImpactPhysical": func(s *VFXPreviewScene) { render.SpawnTypedImpact(cx, cy, "projectile") },
		"typedImpactBeam":     func(s *VFXPreviewScene) { render.SpawnTypedImpact(cx, cy, "wideBeam") },
		"typedImpactBounce":   func(s *VFXPreviewScene) { render.SpawnTypedImpact(cx, cy, "bounce") },
		"typedImpactRadial":   func(s *VFXPreviewScene) { render.SpawnTypedImpact(cx, cy, "radial") },
		"beam": func(s *VFXPreviewScene) {
			s.beamPool.Add(combat.Beam{
				X1: cx - 80, Y1: cy, X2: cx + 80, Y2: cy,
				Width: 4, Color: [3]uint8{180, 100, 255},
				Life: 0.5, MaxLife: 0.5, Wide: true,
			})
		},
		"damageText": func(s *VFXPreviewScene) { render.SpawnDamageText(cx, cy, 1234, false, false) },
		"critText":   func(s *VFXPreviewScene) { render.SpawnDamageText(cx, cy, 5678, true, false) },
		"goldText":   func(s *VFXPreviewScene) { render.SpawnGoldText(cx, cy, 100) },
		"customText": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "Hello VFX!", color.RGBA{R: 100, G: 255, B: 200, A: 255}, 14, 1.5)
		},

		// Post-Processing
		"vignetteStrong": func(s *VFXPreviewScene) { s.effects.VignetteStrength = 0.8 },
		"vignetteOff":    func(s *VFXPreviewScene) { s.effects.VignetteStrength = 0 },
		"dynamicLightWarden": func(s *VFXPreviewScene) {
			s.postPipeline.Lighting.Clear()
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{X: cx, Y: cy, Color: color.RGBA{R: 255, G: 180, B: 80, A: 255}, Radius: 120, Intensity: 0.6})
		},
		"dynamicLightFireball": func(s *VFXPreviewScene) {
			s.postPipeline.Lighting.Clear()
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{X: cx, Y: cy, Color: color.RGBA{R: 255, G: 77, B: 26, A: 255}, Radius: 80, Intensity: 0.8})
		},
		"dynamicLightSkystrike": func(s *VFXPreviewScene) {
			s.postPipeline.Lighting.Clear()
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{X: cx, Y: cy, Color: color.RGBA{R: 230, G: 242, B: 255, A: 255}, Radius: 100, Intensity: 0.9})
		},
		"dynamicLightBuff": func(s *VFXPreviewScene) {
			s.postPipeline.Lighting.Clear()
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{X: cx, Y: cy, Color: color.RGBA{R: 255, G: 217, B: 77, A: 255}, Radius: 100, Intensity: 0.5})
		},
		"dynamicLightSelected": func(s *VFXPreviewScene) {
			s.postPipeline.Lighting.Clear()
			s.postPipeline.Lighting.AddLight(postprocess.PointLight{X: cx, Y: cy, Color: color.RGBA{R: 153, G: 191, B: 255, A: 255}, Radius: 90, Intensity: 0.4})
		},
		"glowLayer": func(s *VFXPreviewScene) {
			particle.EmitBossDeathBurst(s.particlePool, cx, cy)
		},
		"dayNightTint": func(s *VFXPreviewScene) {
			s.effects.DayNightR = 0.2
			s.effects.DayNightG = 0.1
			s.effects.DayNightB = 0.4
			s.effects.DayNightA = 0.5
			s.desatResetTimer = 2.0 // reuse timer to auto-reset
		},

		// Composite
		"bossKill": func(s *VFXPreviewScene) {
			render.TriggerShake(6.0, 0.5)
			s.effects.TriggerRadialBlur(cx, cy, 0.05, 0.6)
			s.effects.TriggerRipple(cx, cy, 18)
			s.effects.TriggerHitStop(8)
			s.effects.TriggerHitFlash(0.1)
			particle.EmitBossDeathBurst(s.particlePool, cx, cy)
		},
		"multiKill": func(s *VFXPreviewScene) {
			render.TriggerShake(3.0, 0.3)
			particle.EmitDeathBurstLarge(s.particlePool, cx, cy)
			render.SpawnDamageText(cx, cy-20, 9999, true, false)
		},
		"waveAnnounceNormal": func(s *VFXPreviewScene) { s.waveAnnounce.Trigger(3, 20, false) },
		"waveAnnounceBoss":   func(s *VFXPreviewScene) { s.waveAnnounce.Trigger(5, 20, true) },
		"toast":              func(s *VFXPreviewScene) { hud.ShowToast("VFX Preview Toast!") },

		// Tower / Projectile / Warden / Enemy VFX — all use continuous drawing
		"buildRipple":      func(s *VFXPreviewScene) { s.activateVFX("buildRipple", 0) },
		"spinBlades":       func(s *VFXPreviewScene) { s.activateVFX("spinBlades", 0) },
		"strengthGlow":     func(s *VFXPreviewScene) { s.activateVFX("strengthGlow", 0) },
		"auraPulse":        func(s *VFXPreviewScene) { s.activateVFX("auraPulse", 0) },
		"damageUpAura":     func(s *VFXPreviewScene) { s.activateVFX("damageUpAura", 0) },
		"attackSpeedAura":  func(s *VFXPreviewScene) { s.activateVFX("attackSpeedAura", 0) },
		"rangeAura":        func(s *VFXPreviewScene) { s.activateVFX("rangeAura", 0) },
		"critAura":         func(s *VFXPreviewScene) { s.activateVFX("critAura", 0) },
		"buffDots":         func(s *VFXPreviewScene) { s.activateVFX("buffDots", 0) },
		"pentagram":        func(s *VFXPreviewScene) { s.activateVFX("pentagram", 0) },
		"projPenetrate":    func(s *VFXPreviewScene) { s.activateVFX("projPenetrate", 0) },
		"projScatter":      func(s *VFXPreviewScene) { s.activateVFX("projScatter", 0) },
		"projSniper":       func(s *VFXPreviewScene) { s.activateVFX("projSniper", 0) },
		"projFreeze":       func(s *VFXPreviewScene) { s.activateVFX("projFreeze", 0) },
		"projRapid":        func(s *VFXPreviewScene) { s.activateVFX("projRapid", 0) },
		"projWind":         func(s *VFXPreviewScene) { s.activateVFX("projWind", 0) },
		"projDefault":      func(s *VFXPreviewScene) { s.activateVFX("projDefault", 0) },
		"projTrailSniper":  func(s *VFXPreviewScene) { s.activateVFX("projTrailSniper", 0) },
		"projTrailRapid":   func(s *VFXPreviewScene) { s.activateVFX("projTrailRapid", 0) },
		"projTrailFreeze":  func(s *VFXPreviewScene) { s.activateVFX("projTrailFreeze", 0) },
		"projTrailWind":    func(s *VFXPreviewScene) { s.activateVFX("projTrailWind", 0) },
		"projTrailDefault": func(s *VFXPreviewScene) { s.activateVFX("projTrailDefault", 0) },
		"fireTrails":       func(s *VFXPreviewScene) { s.activateVFX("fireTrails", 0) },
		"fireballs":        func(s *VFXPreviewScene) { s.activateVFX("fireballs", 0) },
		"shootFlash":       func(s *VFXPreviewScene) { s.activateVFX("shootFlash", 0) },
		"chainLinks":       func(s *VFXPreviewScene) { s.activateVFX("chainLinks", 0) },
		"skystrikeIce":     func(s *VFXPreviewScene) { s.activateVFX("skystrikeIce", 1) },
		"skystrikeWater":   func(s *VFXPreviewScene) { s.activateVFX("skystrikeWater", 2) },
		"skystrikeGeyser":  func(s *VFXPreviewScene) { s.activateVFX("skystrikeGeyser", 3) },
		"goldBeam":         func(s *VFXPreviewScene) { s.activateVFX("goldBeam", 0) },
		"movementTrail":    func(s *VFXPreviewScene) { s.activateVFX("movementTrail", 0) },
		"bossPulse":        func(s *VFXPreviewScene) { s.activateVFX("bossPulse", 0) },
		"runnerRing":       func(s *VFXPreviewScene) { s.activateVFX("runnerRing", 0) },
		"stunStars":        func(s *VFXPreviewScene) { s.activateVFX("stunStars", 0) },
		"enemyHitFlash":    func(s *VFXPreviewScene) { s.activateVFX("enemyHitFlash", 0) },
		"statusDotSlow":    func(s *VFXPreviewScene) { s.activateVFX("statusDotSlow", 0) },
		"statusDotStun":    func(s *VFXPreviewScene) { s.activateVFX("statusDotStun", 0) },
		"statusDotBleed":   func(s *VFXPreviewScene) { s.activateVFX("statusDotBleed", 0) },
		"statusDotBurn":    func(s *VFXPreviewScene) { s.activateVFX("statusDotBurn", 0) },
		"statusDotPoison":  func(s *VFXPreviewScene) { s.activateVFX("statusDotPoison", 0) },
		"statusDotRoot":    func(s *VFXPreviewScene) { s.activateVFX("statusDotRoot", 0) },
		"bufferAura":       func(s *VFXPreviewScene) { s.activateVFX("bufferAura", 0) },
		"purgeGlow":        func(s *VFXPreviewScene) { s.activateVFX("purgeGlow", 0) },
		"immunityRing":     func(s *VFXPreviewScene) { s.activateVFX("immunityRing", 0) },

		// Enemy ability trigger VFX
		"blockFlash":     func(s *VFXPreviewScene) { s.activateVFX("blockFlash", 0) },
		"dodgeFlash":     func(s *VFXPreviewScene) { s.activateVFX("dodgeFlash", 0) },
		"armorSpark":     func(s *VFXPreviewScene) { s.activateVFX("armorSpark", 0) },
		"damageCapPulse": func(s *VFXPreviewScene) { s.activateVFX("damageCapPulse", 0) },
		"purgeWave":      func(s *VFXPreviewScene) { s.activateVFX("purgeWave", 0) },
		"phaseAura":      func(s *VFXPreviewScene) { s.activateVFX("phaseAura", 0) },
		"dashTrails":     func(s *VFXPreviewScene) { s.activateVFX("dashTrails", 0) },
		"healerAura":     func(s *VFXPreviewScene) { s.activateVFX("healerAura", 0) },
		"speedAura":      func(s *VFXPreviewScene) { s.activateVFX("speedAura", 0) },
		"strengthDrain":  func(s *VFXPreviewScene) { s.activateVFX("strengthDrain", 0) },
		"slowOverlay":    func(s *VFXPreviewScene) { s.activateVFX("slowOverlay", 0) },
		"burnOverlay":    func(s *VFXPreviewScene) { s.activateVFX("burnOverlay", 0) },
		"poisonOverlay":  func(s *VFXPreviewScene) { s.activateVFX("poisonOverlay", 0) },

		// Tower UI VFX
		"upgradeDiamond": func(s *VFXPreviewScene) { s.activateVFX("upgradeDiamond", 0) },
		"selectionRing":  func(s *VFXPreviewScene) { s.activateVFX("selectionRing", 0) },

		// Combo text
		"comboX3": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "×3 连杀!", color.RGBA{R: 255, G: 255, B: 255, A: 220}, 14, 1.2)
		},
		"comboX5": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "×5 连杀!", color.RGBA{R: 255, G: 220, B: 60, A: 255}, 16, 1.5)
			render.TriggerShake(1.5, 0.1)
		},
		"comboX10": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "×10 超级连杀!", color.RGBA{R: 255, G: 140, B: 40, A: 255}, 18, 2.0)
			render.TriggerShake(2.0, 0.15)
		},
		"comboX20": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "×20 无双!", color.RGBA{R: 255, G: 60, B: 40, A: 255}, 22, 2.0)
			render.TriggerShake(3.0, 0.2)
		},
		"comboX50": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "×50 传说!", color.RGBA{R: 255, G: 215, B: 0, A: 255}, 24, 2.5)
		},
		"overkill": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "OVERKILL", color.RGBA{R: 255, G: 215, B: 0, A: 255}, 16, 1.5)
		},
		"perfectWave": func(s *VFXPreviewScene) {
			render.SpawnText(cx, cy, "完美!", color.RGBA{R: 255, G: 215, B: 0, A: 255}, 20, 2.0)
		},
	}
}

func (s *VFXPreviewScene) buildCatalog() {
	catalog, err := config.LoadVFXCatalog()
	if err != nil {
		log.Printf("vfx_preview: failed to load catalog, using fallback: %v", err)
		s.buildFallbackCatalog()
		return
	}

	registry := s.vfxTriggerRegistry()
	s.categories = make([]vfxCategory, 0, len(catalog.Categories))

	for _, cfgCat := range catalog.Categories {
		cat := vfxCategory{Name: cfgCat.Label}
		for _, cfgEff := range cfgCat.Effects {
			trigger, ok := registry[cfgEff.ID]
			if !ok {
				log.Printf("vfx_preview: no trigger for effect %q, skipping", cfgEff.ID)
				continue
			}
			cat.Effects = append(cat.Effects, vfxEntry{
				Name:    cfgEff.Label + "  " + cfgEff.Name,
				Trigger: trigger,
			})
		}
		if len(cat.Effects) > 0 {
			s.categories = append(s.categories, cat)
		}
	}
}

// buildFallbackCatalog provides a hardcoded fallback when JSON config is unavailable.
func (s *VFXPreviewScene) buildFallbackCatalog() {
	cx, cy := s.previewCenter()
	s.categories = []vfxCategory{
		{Name: "粒子效果", Effects: []vfxEntry{
			{"死亡爆裂  DeathBurst", func(s *VFXPreviewScene) { particle.EmitDeathBurst(s.particlePool, cx, cy) }},
			{"枪口闪光  MuzzleFlash", func(s *VFXPreviewScene) { particle.EmitMuzzleFlash(s.particlePool, cx, cy, 0) }},
		}},
		{Name: "屏幕特效", Effects: []vfxEntry{
			{"屏幕震动  ScreenShake", func(s *VFXPreviewScene) { render.TriggerShake(4.0, 0.4) }},
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
		s.beamPool.Tick(effectiveDT)
		render.UpdateShake(effectiveDT)
		render.UpdateImpactVFX(effectiveDT)
		render.UpdateFloatTexts(effectiveDT)
		s.waveAnnounce.Update(effectiveDT)
		hud.UpdateToast(effectiveDT)
	}

	// Desaturation / daynight auto-reset.
	if s.desatResetTimer > 0 {
		s.desatResetTimer -= effectiveDT
		if s.desatResetTimer <= 0 {
			s.effects.SetDesaturation(0, 3.0, 0, 0, 0)
			s.effects.DayNightA = 0
		}
	}

	// Active VFX animation time.
	s.vfxTime += effectiveDT

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

	// Clear previous effect state before triggering new one.
	s.clearActiveEffects()

	cat.Effects[s.effectIdx].Trigger(s)
	s.replayDelay = 0.8
}

// clearActiveEffects resets all active visual effect state so that
// switching to a new effect doesn't leave the old one rendering.
func (s *VFXPreviewScene) clearActiveEffects() {
	// Clear continuous VFX.
	s.activeVFX = ""
	s.vfxTime = 0
	s.vfxParam = 0

	// Clear one-shot subsystems.
	s.particlePool.Clear()
	s.beamPool.Clear()
	render.ClearImpactVFX()
	render.ClearFloatTexts()

	// Reset post-processing state.
	s.effects.VignetteStrength = 0
	s.effects.DayNightA = 0
	s.effects.SetDesaturation(0, 3.0, 0, 0, 0)
	s.desatResetTimer = 0
	s.postPipeline.Lighting.Clear()
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
	render.DrawBeams(buf, s.beamPool, 1.0/60.0)

	draw.EndGlowPass(buf)

	// Draw impact VFX.
	render.DrawImpactVFX(buf)

	// Draw float text.
	render.DrawFloatTexts(buf)

	// Draw active vfx effect.
	s.drawActiveVFX(buf)

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

// activateVFX sets the active VFX and resets its timer.
func (s *VFXPreviewScene) activateVFX(id string, param int) {
	s.activeVFX = id
	s.vfxTime = 0
	s.vfxParam = param
}

// drawActiveVFX renders the currently active VFX effect every frame.
// All effects are animated via s.vfxTime for realistic preview.
func (s *VFXPreviewScene) drawActiveVFX(screen *ebiten.Image) {
	if s.activeVFX == "" {
		return
	}

	cx, cy := s.previewCenter()
	fcx, fcy := float32(cx), float32(cy)
	t := s.vfxTime

	switch s.activeVFX {
	// ── Tower VFX ──
	case "buildRipple":
		// Animate progress 0→1 over 0.3s, loop
		p := math.Mod(t, 0.6) / 0.3
		if p <= 1 {
			vfx.DrawBuildRipple(screen, fcx, fcy, p)
		}
	case "spinBlades":
		vfx.DrawSpinBlades(screen, fcx, fcy, 60, t*3, 1.0)
	case "strengthGlow":
		vfx.DrawStrengthGlow(screen, fcx, fcy, 120, t)
	case "auraPulse":
		vfx.DrawAuraPulse(screen, fcx, fcy, 80, color.RGBA{R: 255, G: 160, B: 60, A: 255}, t)
	case "damageUpAura":
		vfx.DrawDamageAura(screen, fcx, fcy, 100, t)
	case "attackSpeedAura":
		vfx.DrawSpeedAuraRing(screen, fcx, fcy, 100, t)
	case "rangeAura":
		vfx.DrawRangeAura(screen, fcx, fcy, 100, t)
	case "critAura":
		vfx.DrawCritAura(screen, fcx, fcy, 100, t)
	case "buffDots":
		vfx.DrawBuffDots(screen, fcx, fcy, 4)
	case "pentagram":
		vfx.DrawPentagram(screen, fcx, fcy, 1.0, t)

	// ── Projectile VFX ──
	// All projectile previews: fly from origin to right with trail + muzzle flash.
	case "projPenetrate":
		s.drawProjFlight(screen, cx, cy, t, "sniper", true, false,
			color.RGBA{R: 200, G: 140, B: 255, A: 255}, color.RGBA{R: 180, G: 100, B: 255, A: 200})
	case "projScatter":
		s.drawProjFlight(screen, cx, cy, t, "freeze", false, true,
			color.RGBA{R: 140, G: 200, B: 255, A: 255}, color.RGBA{R: 100, G: 180, B: 255, A: 200})
	case "projSniper":
		s.drawProjFlight(screen, cx, cy, t, "sniper", false, false,
			color.RGBA{R: 255, G: 180, B: 80, A: 255}, color.RGBA{R: 255, G: 160, B: 60, A: 200})
	case "projFreeze":
		s.drawProjFlight(screen, cx, cy, t, "freeze", false, false,
			color.RGBA{R: 140, G: 220, B: 255, A: 255}, color.RGBA{R: 100, G: 200, B: 255, A: 200})
	case "projRapid":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 180, G: 230, B: 60, A: 255}, color.RGBA{R: 160, G: 220, B: 40, A: 200})
	case "projWind":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 140, G: 230, B: 160, A: 255}, color.RGBA{R: 120, G: 220, B: 140, A: 200})
	case "projDefault":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 253, G: 230, B: 138, A: 255}, color.RGBA{R: 255, G: 220, B: 100, A: 200})
	case "projTrailSniper":
		s.drawProjFlight(screen, cx, cy, t, "sniper", false, false,
			color.RGBA{R: 255, G: 180, B: 80, A: 255}, color.RGBA{R: 255, G: 160, B: 60, A: 200})
	case "projTrailRapid":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 180, G: 230, B: 60, A: 255}, color.RGBA{R: 160, G: 220, B: 40, A: 200})
	case "projTrailFreeze":
		s.drawProjFlight(screen, cx, cy, t, "freeze", false, false,
			color.RGBA{R: 140, G: 220, B: 255, A: 255}, color.RGBA{R: 100, G: 200, B: 255, A: 200})
	case "projTrailWind":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 140, G: 230, B: 160, A: 255}, color.RGBA{R: 120, G: 220, B: 140, A: 200})
	case "projTrailDefault":
		s.drawProjFlight(screen, cx, cy, t, "default", false, false,
			color.RGBA{R: 253, G: 230, B: 138, A: 255}, color.RGBA{R: 255, G: 220, B: 100, A: 200})

	// ── Warden VFX ──
	case "fireTrails":
		// Simulate decaying fire trails
		life := 1.0 - math.Mod(t, 1.5)/1.5
		vfx.DrawFireTrails(screen, []vfx.FireTrailVFX{
			{X: cx - 30, Y: cy, Life: life, MaxLife: 1.0, Radius: 12},
			{X: cx, Y: cy + 15, Life: life * 0.7, MaxLife: 1.0, Radius: 10},
			{X: cx + 30, Y: cy, Life: life * 0.5, MaxLife: 1.0, Radius: 8},
		})
	case "fireballs":
		// Simulate fireball flying across
		p := math.Mod(t, 1.2) / 1.2
		fbX := cx - 60 + 120*p
		vfx.DrawFireballs(screen, []vfx.FireballVFX{
			{X: fbX, Y: cy, Radius: 6, Progress: p, StartX: cx - 60, StartY: cy, EndX: cx + 60, EndY: cy},
		})
	case "shootFlash":
		// Repeating flash every 0.5s
		flashT := math.Mod(t, 0.5)
		if flashT < 0.15 {
			vfx.DrawShootFlash(screen, fcx, fcy, 0.15-flashT, color.RGBA{R: 100, G: 200, B: 255, A: 200})
		}
	case "chainLinks":
		// Pulsing chain links
		vfx.DrawChainLinks(screen, [][4]float64{
			{cx - 60, cy - 30, cx + 60, cy + 30},
			{cx - 40, cy + 20, cx + 40, cy - 20},
		}, t)
	case "skystrikeIce", "skystrikeWater", "skystrikeGeyser":
		// Repeat strike every 1.2s
		cycleT := math.Mod(t, 1.2)
		if cycleT < 0.8 {
			vfx.DrawSkyStrike(screen, fcx, fcy, 0.8-cycleT, s.vfxParam)
		}
	case "goldBeam":
		// Decaying beam
		p := 1.0 - math.Mod(t, 1.0)/0.6
		if p > 0 {
			vfx.DrawGoldBeam(screen, fcx-40, fcy, fcx+40, fcy, p, t)
		}
	case "movementTrail":
		// Simulate circular flight path trail
		trail := make([][2]float64, 16)
		for i := range trail {
			age := float64(i) * 0.06
			a := t - age
			trail[i] = [2]float64{cx + 50*math.Cos(a*2), cy + 30*math.Sin(a*2)}
		}
		vfx.DrawMovementTrail(screen, trail, len(trail)-1, color.RGBA{R: 255, G: 180, B: 80, A: 200})

	// ── Enemy VFX ──
	case "bossPulse":
		vfx.DrawBossPulse(screen, fcx, fcy, 10, t)
	case "runnerRing":
		vfx.DrawRunnerRing(screen, fcx, fcy, 10, t)
	case "stunStars":
		vfx.DrawStunStars(screen, fcx, fcy, 10, t)
	case "enemyHitFlash":
		// Repeating flash
		flashT := math.Mod(t, 0.4)
		if flashT < 0.1 {
			vfx.DrawHitFlash(screen, fcx, fcy, 12, 0.1-flashT)
		}
	case "statusDotSlow":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 125, G: 211, B: 252, A: 235}}}, t)
	case "statusDotStun":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 255, G: 255, B: 100, A: 235}}}, t)
	case "statusDotBleed":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 239, G: 68, B: 68, A: 255}}}, t)
	case "statusDotBurn":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 255, G: 140, B: 40, A: 255}}}, t)
	case "statusDotPoison":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 80, G: 200, B: 40, A: 235}}}, t)
	case "statusDotRoot":
		vfx.DrawStatusDots(screen, fcx, fcy, []vfx.StatusDot{{Color: color.RGBA{R: 139, G: 90, B: 43, A: 235}}}, t)
	case "bufferAura":
		vfx.DrawBufferAura(screen, fcx, fcy, 60, t)
	case "purgeGlow":
		vfx.DrawPurgeGlow(screen, fcx, fcy, 12, t)
	// Enemy ability trigger VFX (repeating animations)
	case "blockFlash":
		cycleT := math.Mod(t, 0.5)
		if cycleT < 0.25 {
			vfx.DrawBlockFlash(screen, fcx, fcy, 10, 0.25-cycleT)
		}
	case "dodgeFlash":
		cycleT := math.Mod(t, 0.6)
		if cycleT < 0.3 {
			vfx.DrawDodgeFlash(screen, fcx, fcy, 10, 0.3-cycleT)
		}
	case "armorSpark":
		cycleT := math.Mod(t, 0.4)
		if cycleT < 0.15 {
			vfx.DrawArmorSpark(screen, fcx, fcy, 10, 0.15-cycleT)
		}
	case "damageCapPulse":
		cycleT := math.Mod(t, 0.6)
		if cycleT < 0.3 {
			vfx.DrawDamageCapPulse(screen, fcx, fcy, 10, 0.3-cycleT)
		}
	case "purgeWave":
		cycleT := math.Mod(t, 0.8)
		if cycleT < 0.4 {
			vfx.DrawPurgeWave(screen, fcx, fcy, 10, 0.4-cycleT)
		}
	case "phaseAura":
		vfx.DrawPhaseAura(screen, fcx, fcy, 10, t)
	case "dashTrails":
		angle := t * 2
		vfx.DrawDashTrails(screen, fcx, fcy, math.Cos(angle), math.Sin(angle))
	case "healerAura":
		vfx.DrawHealerAura(screen, fcx, fcy, 60, t)
		cycleT := math.Mod(t, 1.5)
		if cycleT < 0.4 {
			vfx.DrawHealPulse(screen, fcx, fcy, 10, 60, cycleT/0.4)
		}
	case "speedAura":
		vfx.DrawSpeedAura(screen, fcx, fcy, 60, t)
	case "strengthDrain":
		vfx.DrawStrengthDrainLink(screen, fcx-50, fcy, fcx+50, fcy, t)
	case "slowOverlay":
		vfx.DrawSlowOverlay(screen, fcx, fcy, 12, t)
	case "burnOverlay":
		vfx.DrawBurnOverlay(screen, fcx, fcy, 12, t)
	case "poisonOverlay":
		vfx.DrawPoisonOverlay(screen, fcx, fcy, 12, t)
	case "immunityRing":
		// Show both CC immune (red) and slow immune (cyan) side by side
		vfx.DrawImmunityRing(screen, fcx-25, fcy, 12, color.RGBA{R: 220, G: 60, B: 60, A: 80}, t)
		vfx.DrawImmunityRing(screen, fcx+25, fcy, 12, color.RGBA{R: 60, G: 180, B: 200, A: 80}, t)
	case "upgradeDiamond":
		vfx.DrawUpgradeDiamond(screen, fcx, fcy, t)
	case "selectionRing":
		// Animate range ring pulsing
		pulseR := float32(80 + 3*math.Sin(t*3))
		vfx.DrawSelectionRing(screen, fcx, fcy,
			16, 2, color.RGBA{R: 100, G: 160, B: 255, A: 180},
			pulseR, 1, color.RGBA{R: 100, G: 160, B: 255, A: 60})
	}
}

// drawProjFlight draws a projectile flying from left to right with trail and muzzle flash.
// Loops every 1.0s: muzzle flash at origin → projectile flies across → reset.
func (s *VFXPreviewScene) drawProjFlight(
	screen *ebiten.Image, cx, cy, t float64,
	style string, penetrate, scatter bool,
	trailClr, flashClr color.RGBA,
) {
	const (
		flightDur = 0.8  // seconds for one flight
		cycleDur  = 1.0  // total cycle including pause
		halfSpan  = 90.0 // half of the flight distance
	)

	cycle := math.Mod(t, cycleDur)
	progress := cycle / flightDur // 0→1 during flight, >1 = pause
	if progress > 1 {
		progress = 1
	}

	// Origin (tower position) and current projectile position.
	originX := cx - halfSpan
	originY := cy
	curX := originX + 2*halfSpan*progress
	curY := cy
	angle := 0.0 // flying rightward

	// Muzzle flash at origin (visible briefly at start).
	if cycle < 0.15 {
		vfx.DrawShootFlash(screen, float32(originX), float32(originY), 0.15-cycle, flashClr)
	}

	// Trail behind the projectile.
	const trailLen = 8
	trail := make([]vfx.TrailPt, trailLen)
	for i := range trail {
		frac := float64(i) / float64(trailLen)
		tx := curX - frac*40*progress // trail stretches back as projectile moves
		trail[i] = vfx.TrailPt{X: tx, Y: curY, Active: true}
	}
	vfx.DrawProjectileTrail(screen, trail, 0, trailClr)

	// Projectile body at current position.
	vfx.DrawProjectileBody(screen, float32(curX), float32(curY), angle, style, penetrate, scatter)
}
