// shaders.go — embeds and compiles Kage shaders for the post-processing pipeline.
// WASM/WebGL 下跳过编译：部分 shader（lighting 30 个 uniform）超出
// WebGL MAX_FRAGMENT_UNIFORM_VECTORS 限制，触发 Uniform1fv nil panic。
package postprocess

import (
	_ "embed"
	"fmt"
	"runtime"

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

//go:embed desaturate.kage
var shaderDesaturateSrc []byte

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
	shaderDesaturate   *ebiten.Shader
	shadersReady       bool
)

// InitShaders compiles all Kage shaders. Call once at startup.
// Returns an error if any shader fails to compile.
// WASM 环境直接跳过，避免 WebGL uniform 限制导致 crash。
func InitShaders() error {
	if runtime.GOOS == "js" {
		// WebGL uniform 数量限制，跳过所有后处理 shader
		shadersReady = false
		return nil
	}

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

	shaderDesaturate, err = ebiten.NewShader(shaderDesaturateSrc)
	if err != nil {
		return fmt.Errorf("desaturate shader: %w", err)
	}

	shadersReady = true
	return nil
}

// ShadersReady reports whether shaders have been compiled successfully.
func ShadersReady() bool {
	return shadersReady
}
