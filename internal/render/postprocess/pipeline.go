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
	p.uColorGrade = map[string]any{"TintR": float32(0), "TintG": float32(0), "TintB": float32(0), "TintA": float32(0)}
	p.uRadialBlur = map[string]any{"CenterX": float32(0), "CenterY": float32(0), "Strength": float32(0)}
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
	needColorGrade := fx != nil && fx.HitFlash.Active
	needRadialBlur := fx != nil && fx.RadialBlur.Active && qs.PostProcessing

	// If no bloom and no effects, fast blit.
	if !p.BloomEnabled && !needLighting && !needVignette && !needColorGrade && !needRadialBlur {
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
		p.bloomExtracted.DrawRectShader(qw, qh, shaderBloomExtract, &ebiten.DrawRectShaderOptions{
			Uniforms: p.uBloomExt,
			Images:   [4]*ebiten.Image{p.sceneBuffer},
		})

		// Ping-pong gaussian blur.
		src := p.bloomExtracted
		for i := 0; i < p.BloomPasses; i++ {
			p.bloomBlurA.Clear()
			p.uBlurH["TexelSize"] = float32(1.0 / float64(qw))
			p.bloomBlurA.DrawRectShader(qw, qh, shaderBlurH, &ebiten.DrawRectShaderOptions{
				Uniforms: p.uBlurH,
				Images:   [4]*ebiten.Image{src},
			})
			p.bloomBlurB.Clear()
			p.uBlurV["TexelSize"] = float32(1.0 / float64(qh))
			p.bloomBlurB.DrawRectShader(qw, qh, shaderBlurV, &ebiten.DrawRectShaderOptions{
				Uniforms: p.uBlurV,
				Images:   [4]*ebiten.Image{p.bloomBlurA},
			})
			src = p.bloomBlurB
		}

		// Upscale bloom to full resolution.
		p.bloomUpscaled.Clear()
		var upOpts ebiten.DrawImageOptions
		upOpts.GeoM.Scale(float64(p.sceneW)/float64(qw), float64(p.sceneH)/float64(qh))
		upOpts.Filter = ebiten.FilterLinear
		p.bloomUpscaled.DrawImage(src, &upOpts)

		p.uBloomComb["Intensity"] = float32(p.BloomIntensity)
		if needLighting || needVignette || needColorGrade || needRadialBlur {
			// Bloom combine into fxPingPong for further chaining.
			p.fxPingPong.Clear()
			p.fxPingPong.DrawRectShader(p.sceneW, p.sceneH, shaderBloomCombine, &ebiten.DrawRectShaderOptions{
				Uniforms: p.uBloomComb,
				Images:   [4]*ebiten.Image{p.sceneBuffer, p.bloomUpscaled},
			})
			fxSrc = p.fxPingPong
		} else {
			// No effects after bloom: combine directly to dst.
			dst.DrawRectShader(p.sceneW, p.sceneH, shaderBloomCombine, &ebiten.DrawRectShaderOptions{
				Uniforms: p.uBloomComb,
				Images:   [4]*ebiten.Image{p.sceneBuffer, p.bloomUpscaled},
			})
			return
		}
	} else {
		// No bloom: start effect chain from sceneBuffer.
		p.fxPingPong.Clear()
		p.fxPingPong.DrawImage(p.sceneBuffer, nil)
		fxSrc = p.fxPingPong
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

	// fxTarget picks the correct output for each pass.
	// The last pass writes to dst; intermediate passes ping-pong between
	// bloomUpscaled and fxPingPong to avoid source==destination.
	// fxSrc starts on fxPingPong, so first intermediate target must be bloomUpscaled.
	useBloomUp := true // first intermediate goes to bloomUpscaled (opposite of fxSrc)
	fxTarget := func() *ebiten.Image {
		remaining--
		if remaining == 0 {
			return dst
		}
		if useBloomUp {
			useBloomUp = false
			p.bloomUpscaled.Clear()
			return p.bloomUpscaled
		}
		useBloomUp = true
		p.fxPingPong.Clear()
		return p.fxPingPong
	}

	// Lighting (dynamic point lights).
	if needLighting {
		target := fxTarget()
		target.DrawRectShader(p.sceneW, p.sceneH, shaderLighting, &ebiten.DrawRectShaderOptions{
			Uniforms: p.buildLightingUniforms(),
			Images:   [4]*ebiten.Image{fxSrc},
		})
		fxSrc = target
	}

	// Vignette (always-on edge darkening).
	if needVignette {
		target := fxTarget()
		p.uVignette["Strength"] = float32(fx.VignetteStrength)
		target.DrawRectShader(p.sceneW, p.sceneH, shaderVignette, &ebiten.DrawRectShaderOptions{
			Uniforms: p.uVignette,
			Images:   [4]*ebiten.Image{fxSrc},
		})
		fxSrc = target
	}

	// Color grade (hit flash — tint strength decays over duration).
	if needColorGrade {
		tintA := fx.HitFlash.Timer / fx.HitFlash.Duration
		if tintA < 0 {
			tintA = 0
		}
		target := fxTarget()
		p.uColorGrade["TintR"] = float32(fx.HitTintR)
		p.uColorGrade["TintG"] = float32(fx.HitTintG)
		p.uColorGrade["TintB"] = float32(fx.HitTintB)
		p.uColorGrade["TintA"] = float32(tintA * 0.4) // cap peak flash at 40% blend
		target.DrawRectShader(p.sceneW, p.sceneH, shaderColorGrade, &ebiten.DrawRectShaderOptions{
			Uniforms: p.uColorGrade,
			Images:   [4]*ebiten.Image{fxSrc},
		})
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
		target.DrawRectShader(p.sceneW, p.sceneH, shaderRadialBlur, &ebiten.DrawRectShaderOptions{
			Uniforms: p.uRadialBlur,
			Images:   [4]*ebiten.Image{fxSrc},
		})
	}
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
