// shaders.go — embeds and compiles Kage shaders for the post-processing pipeline.
package postprocess

import (
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed bloom_extract.kage
var bloomExtractSrc []byte

//go:embed blur_h.kage
var blurHSrc []byte

//go:embed blur_v.kage
var blurVSrc []byte

//go:embed bloom_combine.kage
var bloomCombineSrc []byte

//go:embed vignette.kage
var vignetteSrc []byte

//go:embed color_grade.kage
var colorGradeSrc []byte

//go:embed radial_blur.kage
var radialBlurSrc []byte

//go:embed lighting.kage
var lightingSrc []byte

//go:embed ripple.kage
var rippleSrc []byte

// Compiled shaders (set by InitShaders).
var (
	shaderBloomExtract *ebiten.Shader
	shaderBlurH        *ebiten.Shader
	shaderBlurV        *ebiten.Shader
	shaderBloomCombine *ebiten.Shader
	shaderVignette     *ebiten.Shader
	shaderColorGrade   *ebiten.Shader
	shaderRadialBlur   *ebiten.Shader
	shaderLighting     *ebiten.Shader
	shaderRipple       *ebiten.Shader
	shadersReady       bool
)

// InitShaders compiles all Kage shaders. Call once at startup.
// Returns an error if any shader fails to compile.
func InitShaders() error {
	var err error

	shaderBloomExtract, err = ebiten.NewShader(bloomExtractSrc)
	if err != nil {
		return fmt.Errorf("bloom_extract shader: %w", err)
	}

	shaderBlurH, err = ebiten.NewShader(blurHSrc)
	if err != nil {
		return fmt.Errorf("blur_h shader: %w", err)
	}

	shaderBlurV, err = ebiten.NewShader(blurVSrc)
	if err != nil {
		return fmt.Errorf("blur_v shader: %w", err)
	}

	shaderBloomCombine, err = ebiten.NewShader(bloomCombineSrc)
	if err != nil {
		return fmt.Errorf("bloom_combine shader: %w", err)
	}

	shaderVignette, err = ebiten.NewShader(vignetteSrc)
	if err != nil {
		return fmt.Errorf("vignette shader: %w", err)
	}

	shaderColorGrade, err = ebiten.NewShader(colorGradeSrc)
	if err != nil {
		return fmt.Errorf("color_grade shader: %w", err)
	}

	shaderRadialBlur, err = ebiten.NewShader(radialBlurSrc)
	if err != nil {
		return fmt.Errorf("radial_blur shader: %w", err)
	}

	shaderLighting, err = ebiten.NewShader(lightingSrc)
	if err != nil {
		return fmt.Errorf("lighting shader: %w", err)
	}

	shaderRipple, err = ebiten.NewShader(rippleSrc)
	if err != nil {
		return fmt.Errorf("ripple shader: %w", err)
	}

	shadersReady = true
	return nil
}

// ShadersReady reports whether shaders have been compiled successfully.
func ShadersReady() bool {
	return shadersReady
}
