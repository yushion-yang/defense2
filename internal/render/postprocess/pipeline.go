// pipeline.go — GPU 后处理管线。
//
// 管理离屏缓冲区和 Kage shader pass 链，将世界空间渲染的场景图像
// 经过一系列全屏 GPU 特效后输出到最终画面。
//
// 管线架构：
//
//	SceneBuffer (世界渲染目标)
//	  │
//	  ├─ Bloom [当前禁用: Ebitengine v2.9.9 DrawRectShader crash]
//	  │   Extract(亮度提取, 1/4 分辨率) → Blur(乒乓高斯模糊 N 轮) → Combine(合成)
//	  │
//	  ├─ Lighting (动态点光源, 最多 4 盏, 画质可配上限)
//	  ├─ Vignette (边缘暗角, 常驻)
//	  ├─ Ripple (冰塔命中波纹扭曲, 最多 4 个并发)
//	  ├─ Color Grade (受击红闪 + 昼夜色调)
//	  ├─ Desaturate (暂停/失败灰度化 + 色调覆盖)
//	  └─ Radial Blur (径向模糊, 爆炸冲击反馈)
//	      │
//	      ▼
//	    dst (最终屏幕)
//
// 性能关键设计：
//   - 所有 uniform map 和 DrawRectShaderOptions 在 NewPipeline 时预分配，
//     Apply() 中仅更新值，零每帧堆分配
//   - Bloom 缓冲区为场景的 1/4 分辨率（半宽×半高），降低 GPU 开销
//   - 效果 pass 之间通过 fxPingPong/bloomUpscaled 交替作为 src/dst，
//     避免 source==destination 问题，最后一个 pass 直接写入 dst
//   - 画质设置（PostProcessing=false 或 Low）跳过 vignette/lighting/radialBlur，
//     但保留 hitFlash（重要的游戏性反馈）
package postprocess

import (
	"defense2/internal/core/game"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pipeline 管理所有后处理效果的离屏缓冲区和 shader pass。
type Pipeline struct {
	// ── 离屏缓冲区 ──
	sceneBuffer    *ebiten.Image // 全分辨率场景渲染目标（世界空间内容画到这里）
	bloomExtracted *ebiten.Image // 1/4 分辨率亮度提取缓冲
	bloomBlurA     *ebiten.Image // 1/4 分辨率乒乓模糊 A 缓冲
	bloomBlurB     *ebiten.Image // 1/4 分辨率乒乓模糊 B 缓冲
	bloomUpscaled  *ebiten.Image // 全分辨率上采样 bloom（也复用为效果链中间缓冲）
	fxPingPong     *ebiten.Image // 全分辨率临时缓冲（效果 pass 间交替使用）

	// 当前场景缓冲区尺寸（物理像素）。窗口缩放时自动重建所有缓冲区。
	sceneW, sceneH int

	// ── Bloom 配置 ──
	BloomEnabled   bool    // 当前禁用：Ebitengine v2.9.9 DrawRectShader 运行时 crash
	BloomThreshold float64 // 亮度提取阈值
	BloomIntensity float64 // 合成强度
	BloomPasses    int     // 高斯模糊轮数

	// ── 屏幕级效果状态 ──
	Effects  *Effects       // 暗角/受击闪/径向模糊/波纹/灰度化
	Lighting *LightingState // 动态点光源（最多 4 盏）

	// ── 预分配的 uniform map（每帧原地更新值，零分配） ──
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

	// ── 预分配的 shader 选项（避免每帧堆分配 DrawRectShaderOptions） ──
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

// NewPipeline 创建后处理管线（含默认 Bloom 配置）。
// 所有 uniform map 和 shader options 在此一次性预分配，
// 后续 Apply() 调用只更新值不分配内存。
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

// SceneBuffer 返回用于世界渲染的离屏图像。
// 物理尺寸变化时自动重建所有缓冲区（场景+bloom+乒乓），
// 旧 GPU 纹理通过 Deallocate() 显式释放以避免 GPU 内存泄漏。
// 每帧调用开头会 Clear() 场景缓冲区。
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

// Apply 执行后处理链并将结果绘制到 dst。
//
// 流程概览：
//  1. 检查 shader 是否就绪（首帧可能未编译完成），否则直接 blit 场景
//  2. 判断各效果是否需要激活（尊重画质设置 + 效果状态）
//  3. 若无任何效果需要，快速 blit 跳过所有 GPU pass
//  4. Bloom pass（当前禁用）：Extract → Blur × N → Upscale → Combine
//  5. 效果 pass 链：Lighting → Vignette → Ripple → ColorGrade → Desaturate → RadialBlur
//
// pass 链的缓冲区管理：
//   - fxSrc: 指向当前持有结果的缓冲区
//   - fxTarget(): 返回下一个 pass 的输出缓冲区
//   - 最后一个 pass 直接写入 dst（避免多余的全屏 copy）
//   - 中间 pass 在 fxPingPong 和 bloomUpscaled 之间交替，确保 src != dst
func (p *Pipeline) Apply(dst *ebiten.Image) {
	if p.sceneBuffer == nil {
		return
	}

	if !shadersReady {
		dst.DrawImage(p.sceneBuffer, nil)
		return
	}

	// 判断各效果 pass 是否需要激活。
	// 画质策略：PostProcessing=false（Low 画质）时跳过 vignette/lighting/radialBlur/ripple，
	// 但 hitFlash（受击红闪）和 desaturate（暂停灰度）始终保留——它们是重要的游戏反馈。
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

	// --- 效果 pass 链 ---
	// 计算剩余 pass 数量，最后一个 pass 直接写入 dst，中间 pass 写入乒乓缓冲。
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

	// fxTarget 为每个 pass 选择正确的输出缓冲区。
	// 规则：最后一个 pass 写入 dst；中间 pass 在 fxPingPong 和 bloomUpscaled 之间交替。
	// 当 fxSrc=sceneBuffer（无 bloom）时，第一个中间缓冲用 fxPingPong；
	// 当 fxSrc=fxPingPong（有 bloom）时，第一个中间缓冲用 bloomUpscaled。
	// 这确保了 source 和 destination 永远不是同一个缓冲区。
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

	// 动态点光源（最多 4 盏，画质可配上限）。
	// 光源位置和半径需从逻辑坐标转为物理像素（draw.S()）。
	if needLighting {
		target := fxTarget()
		p.opLighting.Uniforms = p.buildLightingUniforms()
		p.opLighting.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderLighting, &p.opLighting)
		fxSrc = target
	}

	// 暗角效果（常驻的边缘暗化，Strength 由 Effects 控制）。
	if needVignette {
		target := fxTarget()
		p.uVignette["Strength"] = float32(fx.VignetteStrength)
		p.opVignette.Uniforms = p.uVignette
		p.opVignette.Images = [4]*ebiten.Image{fxSrc}
		target.DrawRectShader(p.sceneW, p.sceneH, shaderVignette, &p.opVignette)
		fxSrc = target
	}

	// 波纹扭曲效果（冰塔命中时触发，最多 4 个并发波纹）。
	// 每个波纹有独立的中心点(X,Y)、时间(T)和振幅(A)。
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

	// 色彩调整（受击红闪 + 昼夜环境色调）。
	// hitFlash 峰值 alpha 上限为 40%（tintA * 0.4），避免全屏被色调覆盖。
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

	// 灰度化（暂停/战败时画面去色 + 色调叠加）。
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

	// 径向模糊（爆炸/boss 死亡等冲击反馈，强度随时间衰减）。
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

// lightKeys 预计算 4 盏灯的 uniform 键名字符串。
// 避免 buildLightingUniforms 中每帧拼接 "LightX0"/"LightY0" 等字符串
// （原先每帧 28 次 fmt.Sprintf 分配，现在 0 分配）。
var lightKeys = [MaxLights]struct{ X, Y, R, G, B, Radius, Intensity string }{
	{"LightX0", "LightY0", "LightR0", "LightG0", "LightB0", "LightRadius0", "LightIntensity0"},
	{"LightX1", "LightY1", "LightR1", "LightG1", "LightB1", "LightRadius1", "LightIntensity1"},
	{"LightX2", "LightY2", "LightR2", "LightG2", "LightB2", "LightRadius2", "LightIntensity2"},
	{"LightX3", "LightY3", "LightR3", "LightG3", "LightB3", "LightRadius3", "LightIntensity3"},
}

// buildLightingUniforms 原地更新 p.uLighting 并返回。
// 光源位置和半径通过 draw.S() 从逻辑坐标转为物理像素（shader 在物理空间工作）。
// 活跃光源数量受画质设置的 MaxLights 上限约束。
// 非活跃槽位清零（Intensity=0）以避免 shader 残留。
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
		k := &lightKeys[i]
		if i < count {
			l := &ls.Lights[i]
			p.uLighting[k.X] = float32(draw.S(l.X))
			p.uLighting[k.Y] = float32(draw.S(l.Y))
			p.uLighting[k.R] = float32(l.Color.R) / 255
			p.uLighting[k.G] = float32(l.Color.G) / 255
			p.uLighting[k.B] = float32(l.Color.B) / 255
			p.uLighting[k.Radius] = float32(draw.S(l.Radius))
			p.uLighting[k.Intensity] = float32(l.Intensity)
		} else {
			p.uLighting[k.X] = float32(0)
			p.uLighting[k.Y] = float32(0)
			p.uLighting[k.R] = float32(0)
			p.uLighting[k.G] = float32(0)
			p.uLighting[k.B] = float32(0)
			p.uLighting[k.Radius] = float32(1)
			p.uLighting[k.Intensity] = float32(0)
		}
	}
	return p.uLighting
}
