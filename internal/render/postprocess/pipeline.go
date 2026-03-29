// pipeline.go — post-processing pipeline with bloom effect.
// Manages offscreen buffers and applies GPU shader passes.
package postprocess

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Pipeline manages all post-processing effects.
type Pipeline struct {
	sceneBuffer    *ebiten.Image // full resolution scene render target
	bloomExtracted *ebiten.Image // 1/4 resolution bright pixels
	bloomBlurA     *ebiten.Image // 1/4 resolution ping buffer
	bloomBlurB     *ebiten.Image // 1/4 resolution pong buffer
	bloomUpscaled  *ebiten.Image // full resolution upscaled bloom (for combine)

	// Current scene buffer dimensions (physical pixels).
	sceneW, sceneH int

	// Bloom configuration.
	BloomEnabled   bool
	BloomThreshold float64
	BloomIntensity float64
	BloomPasses    int
}

// NewPipeline creates a pipeline with default bloom settings.
func NewPipeline() *Pipeline {
	p := &Pipeline{
		BloomEnabled:   true,
		BloomThreshold: BloomDefault.Threshold,
		BloomIntensity: BloomDefault.Intensity,
		BloomPasses:    BloomDefault.Passes,
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
	}
	p.sceneBuffer.Clear()
	return p.sceneBuffer
}

// Apply runs the post-processing chain and draws the result to dst.
// If bloom is disabled or shaders are not compiled, it blits the scene directly.
func (p *Pipeline) Apply(dst *ebiten.Image) {
	if p.sceneBuffer == nil {
		return
	}

	if !p.BloomEnabled || !shadersReady {
		// No bloom: just blit scene to destination.
		dst.DrawImage(p.sceneBuffer, nil)
		return
	}

	qw := p.bloomExtracted.Bounds().Dx()
	qh := p.bloomExtracted.Bounds().Dy()

	// Pass 1: Extract bright pixels (downscale to 1/4 res).
	p.bloomExtracted.Clear()
	p.bloomExtracted.DrawRectShader(qw, qh, shaderBloomExtract, &ebiten.DrawRectShaderOptions{
		Uniforms: map[string]any{
			"Threshold": float32(p.BloomThreshold),
		},
		Images: [4]*ebiten.Image{p.sceneBuffer},
	})

	// Pass 2+: Ping-pong gaussian blur.
	src := p.bloomExtracted
	for i := 0; i < p.BloomPasses; i++ {
		// Horizontal blur: src -> bloomBlurA
		p.bloomBlurA.Clear()
		p.bloomBlurA.DrawRectShader(qw, qh, shaderBlurH, &ebiten.DrawRectShaderOptions{
			Uniforms: map[string]any{
				"TexelSize": float32(1.0 / float64(qw)),
			},
			Images: [4]*ebiten.Image{src},
		})

		// Vertical blur: bloomBlurA -> bloomBlurB
		p.bloomBlurB.Clear()
		p.bloomBlurB.DrawRectShader(qw, qh, shaderBlurV, &ebiten.DrawRectShaderOptions{
			Uniforms: map[string]any{
				"TexelSize": float32(1.0 / float64(qh)),
			},
			Images: [4]*ebiten.Image{p.bloomBlurA},
		})

		src = p.bloomBlurB
	}

	// Pass 3: Upscale bloom to full resolution, then combine with scene.
	// DrawRectShader requires all source images to be the same size,
	// so we upscale the 1/4 res bloom to full res first.
	p.bloomUpscaled.Clear()
	upOpts := &ebiten.DrawImageOptions{}
	upOpts.GeoM.Scale(float64(p.sceneW)/float64(qw), float64(p.sceneH)/float64(qh))
	upOpts.Filter = ebiten.FilterLinear
	p.bloomUpscaled.DrawImage(src, upOpts)

	// Combine: scene (imageSrc0) + upscaled bloom (imageSrc1) -> dst.
	dst.DrawRectShader(p.sceneW, p.sceneH, shaderBloomCombine, &ebiten.DrawRectShaderOptions{
		Uniforms: map[string]any{
			"Intensity": float32(p.BloomIntensity),
		},
		Images: [4]*ebiten.Image{
			p.sceneBuffer,
			p.bloomUpscaled,
		},
	})
}
