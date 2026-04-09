// pipeline.go — post-processing pipeline with bloom effect.
// Manages offscreen buffers and applies GPU shader passes.
package postprocess

import (
	"defense2/internal/core/game"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pipeline manages all post-processing effects.
type Pipeline struct {
	sceneBuffer    *ebiten.Image // full resolution scene render target
	bloomExtracted *ebiten.Image // 1/4 resolution bright pixels
	bloomBlurA     *ebiten.Image // 1/4 resolution ping buffer
	bloomBlurB     *ebiten.Image // 1/4 resolution pong buffer
	bloomUpscaled  *ebiten.Image // full resolution upscaled bloom (for combine)
	fxPingPong     *ebiten.Image // full resolution temp buffer for effect pass chaining

	// Current scene buffer dimensions (physical pixels).
	sceneW, sceneH int

	// Bloom configuration.
	BloomEnabled   bool
	BloomThreshold float64
	BloomIntensity float64
	BloomPasses    int

	// Screen-level effects (vignette, hit flash, radial blur, hit-stop).
	Effects *Effects

	// Dynamic point lighting.
	Lighting *LightingState

	// Pre-allocated uniform maps (reused each frame, values updated in-place).
	uBloomExt   map[string]any
	uBlurH      map[string]any
	uBlurV      map[string]any
	uBloomComb  map[string]any
	uVignette   map[string]any
	uColorGrade map[string]any
	uRadialBlur map[string]any
	uLighting   map[string]any
	uRipple     map[string]any
	uDesat      map[string]any

	// Pre-allocated shader options (avoids per-frame heap allocation).
	opBloomExt  ebiten.DrawRectShaderOptions
	opBlurH     ebiten.DrawRectShaderOptions
	opBlurV     ebiten.DrawRectShaderOptions
	opBloomComb ebiten.DrawRectShaderOptions
	opVignette  ebiten.DrawRectShaderOptions
	opColorGr   ebiten.DrawRectShaderOptions
	opRadialBl  ebiten.DrawRectShaderOptions
	opLighting  ebiten.DrawRectShaderOptions
	opRipple    ebiten.DrawRectShaderOptions
	opDesat     ebiten.DrawRectShaderOptions
}

// NewPipeline creates a pipeline with default bloom settings.
func NewPipeline() *Pipeline {
	p := &Pipeline{
		BloomEnabled:   false, // 暂禁: DrawRectShader v2.9.9 runtime crash
		BloomThreshold: BloomDefault.Threshold,
		BloomIntensity: BloomDefault.Intensity,
		BloomPasses:    BloomDefault.Passes,
		Effects:        NewEffects(),
		Lighting:       NewLightingState(),
	}

	// Pre-allocate uniform maps so Apply() never allocates per frame.
	p.uBloomExt = map[string]any{"Threshold": float32(0)}
	p.uBlurH = map[string]any{"TexelSize": float32(0)}
	p.uBlurV = map[string]any{"TexelSize": float32(0)}
	p.uBloomComb = map[string]any{"Intensity": float32(0)}
	p.uVignette = map[string]any{"Strength": float32(0)}
	p.uColorGrade = map[string]any{
		"TintR": float32(0), "TintG": float32(0), "TintB": float32(0), "TintA": float32(0),
		"DayNightR": float32(0), "DayNightG": float32(0), "DayNightB": float32(0), "DayNightA": float32(0),
	}
	p.uRadialBlur = map[string]any{"CenterX": float32(0), "CenterY": float32(0), "Strength": float32(0)}
	p.uRipple = map[string]any{
		"Ripple0X": float32(0), "Ripple0Y": float32(0), "Ripple0T": float32(0), "Ripple0A": float32(0),
		"Ripple1X": float32(0), "Ripple1Y": float32(0), "Ripple1T": float32(0), "Ripple1A": float32(0),
		"Ripple2X": float32(0), "Ripple2Y": float32(0), "Ripple2T": float32(0), "Ripple2A": float32(0),
		"Ripple3X": float32(0), "Ripple3Y": float32(0), "Ripple3T": float32(0), "Ripple3A": float32(0),
		"ScreenW": float32(0), "ScreenH": float32(0),
	}
	p.uDesat = map[string]any{
		"Strength": float32(0), "TintR": float32(0), "TintG": float32(0), "TintB": float32(0),
	}
	p.uLighting = map[string]any{
		"Ambient": float32(0), "LightCount": float32(0),
		"LightX0": float32(0), "LightY0": float32(0), "LightR0": float32(0), "LightG0": float32(0), "LightB0": float32(0), "LightRadius0": float32(1), "LightIntensity0": float32(0),
		"LightX1": float32(0), "LightY1": float32(0), "LightR1": float32(0), "LightG1": float32(0), "LightB1": float32(0), "LightRadius1": float32(1), "LightIntensity1": float32(0),
		"LightX2": float32(0), "LightY2": float32(0), "LightR2": float32(0), "LightG2": float32(0), "LightB2": float32(0), "LightRadius2": float32(1), "LightIntensity2": float32(0),
		"LightX3": float32(0), "LightY3": float32(0), "LightR3": float32(0), "LightG3": float32(0), "LightB3": float32(0), "LightRadius3": float32(1), "LightIntensity3": float32(0),
	}

	return p
}

// SceneBuffer returns the offscreen image for world rendering.
// It auto-resizes if the physical dimensions change.
func (p *Pipeline) SceneBuffer(physW, physH int) *ebiten.Image {
	if physW <= 0 || physH <= 0 {
		physW, physH = 1, 1
	}
	if p.sceneBuffer == nil || p.sceneW != physW || p.sceneH != physH {
		// Deallocate old GPU textures before replacing.
		if p.sceneBuffer != nil {
			p.sceneBuffer.Deallocate()
			p.bloomExtracted.Deallocate()
			p.bloomBlurA.Deallocate()
			p.bloomBlurB.Deallocate()
			p.bloomUpscaled.Deallocate()
			p.fxPingPong.Deallocate()
		}
		p.sceneW = physW
		p.sceneH = physH
		p.sceneBuffer = ebiten.NewImage(physW, physH)

		// Bloom buffers at 1/4 resolution (half width, half height).
		qw, qh := physW/2, physH/2
		if qw < 1 {
			qw = 1
		}
		if qh < 1 {
			qh = 1
		}
		p.bloomExtracted = ebiten.NewImage(qw, qh)
		p.bloomBlurA = ebiten.NewImage(qw, qh)
		p.bloomBlurB = ebiten.NewImage(qw, qh)
		p.bloomUpscaled = ebiten.NewImage(physW, physH)
		p.fxPingPong = ebiten.NewImage(physW, physH)
	}
	p.sceneBuffer.Clear()
	return p.sceneBuffer
}

// Apply runs the post-processing chain and draws the result to dst.
// Chain order: Bloom -> Vignette -> Color Grade -> Radial Blur.
// If shaders are not compiled, it blits the scene directly.
func (p *Pipeline) Apply(dst *ebiten.Image) {
	if p.sceneBuffer == nil {
		return
	}

	if !shadersReady {
		dst.DrawImage(p.sceneBuffer, nil)
		return
	}

	// Determine which effect passes are needed.
	// Respect quality settings: vignette/lighting/radialBlur are skipped on Low,
	// but hit flash (color grade) is always kept as important gameplay feedback.
	qs := game.Settings()
	fx := p.Effects
	ls := p.Lighting
	needLighting := ls != nil && ls.Enabled && ls.Count > 0 && qs.PostProcessing
	needVignette := fx != nil && fx.VignetteStrength > 0 && qs.PostProcessing
	needColorGrade := fx != nil && (fx.HitFlash.Active || fx.DayNightA > 0)
	needRadialBlur := fx != nil && fx.RadialBlur.Active && qs.PostProcessing
	needRipple := fx != nil && qs.PostProcessing && hasActiveRipples(fx)
	needDesat := fx != nil && fx.DesatStrength > 0.01

	// If no bloom and no effects, fast blit.
	if !p.BloomEnabled && !needLighting && !needVignette && !needColorGrade && !needRadialBlur && !needRipple && !needDesat {
		dst.DrawImage(p.sceneBuffer, nil)
		return
	}

	// fxSrc tracks which buffer holds the current result.
	var fxSrc *ebiten.Image

	// --- Bloom passes ---
	if p.BloomEnabled {
		qw := p.bloomExtracted.Bounds().Dx()
		qh := p.bloomExtracted.Bounds().Dy()

		// Extract bright pixels (downscale to 1/4 res).
		p.bloomExtracted.Clear()
		p.uBloomExt["Threshold"] = float32(p.BloomThreshold)
		p.opBloomExt.Uniforms = p.uBloomExt
		p.opBloomExt.Images = [4]*ebiten.Image{p.sceneBuffer}
		p.bloomExtracted.DrawRectShader(qw, qh, shaderBloomExtract, &p.opBloomExt)

		// Ping-pong gaussian blur.
		src := p.bloomExtracted
		for i := 0; i < p.BloomPasses; i++ {
			p.bloomBlurA.Clear()
			p.uBlurH["TexelSize"] = float32(1.0 / float64(qw))
			p.opBlurH.Uniforms = p.uBlurH
			p.opBlurH.Images = [4]*ebiten.Image{src}
			p.bloomBlurA.DrawRectShader(qw, qh, shaderBlurH, &p.opBlurH)
			p.bloomBlurB.Clear()
			p.uBlurV["TexelSize"] = float32(1.0 / float64(qh))
			p.opBlurV.Uniforms = p.uBlurV
			p.opBlurV.Images = [4]*ebiten.Image{p.bloomBlurA}
			p.bloomBlurB.DrawRectShader(qw, qh, shaderBlurV, &p.opBlurV)
			src = p.bloomBlurB
		}

		// Upscale bloom to full resolution.
		p.bloomUpscaled.Clear()
		var upOpts ebiten.DrawImageOptions
		upOpts.GeoM.Scale(float64(p.sceneW)/float64(qw), float64(p.sceneH)/float64(qh))
		upOpts.Filter = ebiten.FilterLinear
		p.bloomUpscaled.DrawImage(src, &upOpts)

		p.uBloomComb["Intensity"] = float32(p.BloomIntensity)
		p.opBloomComb.Uniforms = p.uBloomComb
		p.opBloomComb.Images = [4]*ebiten.Image{p.sceneBuffer, p.bloomUpscaled}
		if needLighting || needVignette || needColorGrade || needRadialBlur || needRipple || needDesat {
			// Bloom combine into fxPingPong for further chaining.
			p.fxPingPong.Clear()
			p.fxPingPong.DrawRectShader(p.sceneW, p.sceneH, shaderBloomCombine, &p.opBloomComb)
			fxSrc = p.fxPingPong
		} else {
			// No effects after bloom: combine directly to dst.
			dst.DrawRectShader(p.sceneW, p.sceneH, shaderBloomCombine, &p.opBloomComb)
			return
		}
	} else {
		// No bloom: start effect chain directly from sceneBuffer (no full-screen copy).
		fxSrc = p.sceneBuffer
	}

	// --- Effect pass chaining ---
	// Count remaining passes to know when to write directly to dst.
	remaining := 0
	if needLighting {
		remaining++
	}
	if needVignette {
		remaining++
	}
	if needColorGrade {
		remaining++
	}
	if needRadialBlur {
		remaining++
	}
	if needRipple {
		remaining++
	}
	if needDesat {
		remaining++
	}

	// fxTarget picks the correct output for each pass.
	// The last pass writes to dst; intermediate passes ping-pong between
	// fxPingPong and bloomUpscaled to avoid source==destination.
	// When fxSrc=sceneBuffer (no bloom), first intermediate goes to fxPingPong.
	// When fxSrc=fxPingPong (bloom), first intermediate goes to bloomUpscaled.
	usePingPong := fxSrc != p.fxPingPong // first target is fxPingPong if src is sceneBuffer
	fxTarget := func() *ebiten.Image {
		remaining--
		if remaining == 0 {
			return dst
		}
		if usePingPong {
			usePingPong = false
			p.fxPingPong.Clear()
			return p.fxPingPong
		}
		usePingPong = true
		p.bloomUpscaled.Clear()
		return p.bloomUpscaled
	}

	// Lighting (dynamic point lights).
	if needLighting {
		target := fxTarget()
		p.opLighting.Uniforms = p.buildLightingUniforms()
		p.opLighting.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderLighting, &p.opLighting)
		fxSrc = target
	}

	// Vignette (always-on edge darkening).
	if needVignette {
		target := fxTarget()
		p.uVignette["Strength"] = float32(fx.VignetteStrength)
		p.opVignette.Uniforms = p.uVignette
		p.opVignette.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderVignette, &p.opVignette)
		fxSrc = target
	}

	// Ripple distortion (ice tower hits).
	if needRipple {
		scale := draw.Scale
		for i := 0; i < MaxRipples; i++ {
			r := fx.Ripples[i]
			prefix := [4]string{"Ripple0", "Ripple1", "Ripple2", "Ripple3"}[i]
			p.uRipple[prefix+"X"] = float32(r.X * scale)
			p.uRipple[prefix+"Y"] = float32(r.Y * scale)
			p.uRipple[prefix+"T"] = float32(r.Time)
			p.uRipple[prefix+"A"] = float32(r.Amplitude)
		}
		p.uRipple["ScreenW"] = float32(p.sceneW)
		p.uRipple["ScreenH"] = float32(p.sceneH)
		target := fxTarget()
		p.opRipple.Uniforms = p.uRipple
		p.opRipple.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderRipple, &p.opRipple)
		fxSrc = target
	}

	// Color grade (hit flash + day/night ambient tint).
	if needColorGrade {
		tintA := 0.0
		if fx.HitFlash.Active && fx.HitFlash.Duration > 0 {
			tintA = fx.HitFlash.Timer / fx.HitFlash.Duration
			if tintA < 0 {
				tintA = 0
			}
		}
		target := fxTarget()
		p.uColorGrade["TintR"] = float32(fx.HitTintR)
		p.uColorGrade["TintG"] = float32(fx.HitTintG)
		p.uColorGrade["TintB"] = float32(fx.HitTintB)
		p.uColorGrade["TintA"] = float32(tintA * 0.4) // cap peak flash at 40% blend
		p.uColorGrade["DayNightR"] = float32(fx.DayNightR)
		p.uColorGrade["DayNightG"] = float32(fx.DayNightG)
		p.uColorGrade["DayNightB"] = float32(fx.DayNightB)
		p.uColorGrade["DayNightA"] = float32(fx.DayNightA)
		p.opColorGr.Uniforms = p.uColorGrade
		p.opColorGr.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderColorGrade, &p.opColorGr)
		fxSrc = target
	}

	// Desaturation (pause/defeat grayscale + tint).
	if needDesat {
		p.uDesat["Strength"] = float32(fx.DesatStrength)
		p.uDesat["TintR"] = float32(fx.DesatTintR)
		p.uDesat["TintG"] = float32(fx.DesatTintG)
		p.uDesat["TintB"] = float32(fx.DesatTintB)
		target := fxTarget()
		p.opDesat.Uniforms = p.uDesat
		p.opDesat.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderDesaturate, &p.opDesat)
		fxSrc = target
	}

	// Radial blur (strength decays over duration).
	if needRadialBlur {
		t := fx.RadialBlur.Timer / fx.RadialBlur.Duration
		if t < 0 {
			t = 0
		}
		target := fxTarget()
		p.uRadialBlur["CenterX"] = float32(fx.BlurCenterX)
		p.uRadialBlur["CenterY"] = float32(fx.BlurCenterY)
		p.uRadialBlur["Strength"] = float32(fx.BlurStrength * t)
		p.opRadialBl.Uniforms = p.uRadialBlur
		p.opRadialBl.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderRadialBlur, &p.opRadialBl)
	}
}

func hasActiveRipples(fx *Effects) bool {
	for i := range fx.Ripples {
		if fx.Ripples[i].Amplitude > 0 {
			return true
		}
	}
	return false
}

// buildLightingUniforms updates p.uLighting in-place and returns it.
// Positions and radii are converted to physical pixels via draw.S().
func (p *Pipeline) buildLightingUniforms() map[string]any {
	ls := p.Lighting
	// Cap active light count at the quality-level maximum.
	lightCap := game.Settings().MaxLights
	count := ls.Count
	if lightCap >= 0 && count > lightCap {
		count = lightCap
	}
	p.uLighting["Ambient"] = float32(ls.Ambient)
	p.uLighting["LightCount"] = float32(count)
	for i := 0; i < MaxLights; i++ {
		suffix := [4]string{"0", "1", "2", "3"}[i]
		if i < count {
			l := &ls.Lights[i]
			p.uLighting["LightX"+suffix] = float32(draw.S(l.X))
			p.uLighting["LightY"+suffix] = float32(draw.S(l.Y))
			p.uLighting["LightR"+suffix] = float32(l.Color.R) / 255
			p.uLighting["LightG"+suffix] = float32(l.Color.G) / 255
			p.uLighting["LightB"+suffix] = float32(l.Color.B) / 255
			p.uLighting["LightRadius"+suffix] = float32(draw.S(l.Radius))
			p.uLighting["LightIntensity"+suffix] = float32(l.Intensity)
		} else {
			p.uLighting["LightX"+suffix] = float32(0)
			p.uLighting["LightY"+suffix] = float32(0)
			p.uLighting["LightR"+suffix] = float32(0)
			p.uLighting["LightG"+suffix] = float32(0)
			p.uLighting["LightB"+suffix] = float32(0)
			p.uLighting["LightRadius"+suffix] = float32(1)
			p.uLighting["LightIntensity"+suffix] = float32(0)
		}
	}
	return p.uLighting
}
