// font.go — 字体管理与文本渲染工具。
// 基于 Ebitengine text/v2 提供双字体渲染：JetBrains Mono（英文/数字）+ Noto Sans SC（中文回退）。
package render

import (
	"bytes"
	"image/color"
	"log"
	"math"
	"sync"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// FontManager 管理双字体源并缓存不同尺寸的 MultiFace。
// 单线程使用（Ebitengine Update/Draw 同一 goroutine），不需要 mutex。
type FontManager struct {
	primary  *text.GoTextFaceSource // JetBrains Mono（英文/数字优先）
	fallback *text.GoTextFaceSource // Noto Sans SC（中文回退）
	faces    map[float64]text.Face
}

// NewFontManager 从 TTF 字节数据创建字体管理器（单字体）。
func NewFontManager(ttfData []byte) (*FontManager, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttfData))
	if err != nil {
		return nil, err
	}
	return &FontManager{
		fallback: src,
		faces:    make(map[float64]text.Face),
	}, nil
}

// NewDualFontManager 从两个 TTF 创建双字体管理器。
func NewDualFontManager(primaryTTF, fallbackTTF []byte) (*FontManager, error) {
	pSrc, err := text.NewGoTextFaceSource(bytes.NewReader(primaryTTF))
	if err != nil {
		return nil, err
	}
	fSrc, err := text.NewGoTextFaceSource(bytes.NewReader(fallbackTTF))
	if err != nil {
		return nil, err
	}
	return &FontManager{
		primary:  pSrc,
		fallback: fSrc,
		faces:    make(map[float64]text.Face),
	}, nil
}

// Face 返回指定尺寸的字体（缓存复用）。
// 有双字体时返回 MultiFace，否则返回单 GoTextFace。
func (fm *FontManager) Face(size float64) text.Face {
	if f, ok := fm.faces[size]; ok {
		return f
	}
	var face text.Face
	if fm.primary != nil {
		pFace := &text.GoTextFace{Source: fm.primary, Size: size}
		fFace := &text.GoTextFace{Source: fm.fallback, Size: size}
		mf, err := text.NewMultiFace(pFace, fFace)
		if err != nil {
			log.Printf("MultiFace creation failed, using fallback only: %v", err)
			face = fFace
		} else {
			face = mf
		}
	} else {
		face = &text.GoTextFace{Source: fm.fallback, Size: size}
	}
	fm.faces[size] = face
	return face
}

// DrawText 在指定位置绘制文本（左上角对齐）。
// x, y, size 均为逻辑坐标，内部按 DeviceScale 缩放。
func (fm *FontManager) DrawText(screen *ebiten.Image, s string, x, y, size float64, clr color.Color) {
	sc := draw.Scale
	face := fm.Face(size * sc)
	op := &text.DrawOptions{}
	// 像素对齐：四舍五入到整数像素，避免子像素模糊
	px := math.Round(x * sc)
	py := math.Round(y * sc)
	op.GeoM.Translate(px, py)
	r, g, b, a := clr.RGBA()
	op.ColorScale.SetR(float32(r) / 0xffff)
	op.ColorScale.SetG(float32(g) / 0xffff)
	op.ColorScale.SetB(float32(b) / 0xffff)
	op.ColorScale.SetA(float32(a) / 0xffff)
	text.Draw(screen, s, face, op)
}

// DrawCenteredText 在指定 X 中心位置绘制居中文本。
func (fm *FontManager) DrawCenteredText(screen *ebiten.Image, s string, centerX, y, size float64, clr color.Color) {
	w := fm.MeasureText(s, size)
	fm.DrawText(screen, s, centerX-w/2, y, size, clr)
}

// DrawRightText 在指定位置右对齐绘制文本。
func (fm *FontManager) DrawRightText(screen *ebiten.Image, s string, rightX, y, size float64, clr color.Color) {
	w := fm.MeasureText(s, size)
	fm.DrawText(screen, s, rightX-w, y, size, clr)
}

// ── Bold variants (simulated via multi-pass offset rendering) ──

// DrawBoldText 绘制粗体文本（通过单像素偏移渲染模拟）。
// 在原生像素级别偏移 1px，确保清晰不模糊。
func (fm *FontManager) DrawBoldText(screen *ebiten.Image, s string, x, y, size float64, clr color.Color) {
	sc := draw.Scale
	face := fm.Face(size * sc)
	px := math.Round(x * sc)
	py := math.Round(y * sc)

	r, g, b, a := clr.RGBA()
	var cs ebiten.ColorScale
	cs.SetR(float32(r) / 0xffff)
	cs.SetG(float32(g) / 0xffff)
	cs.SetB(float32(b) / 0xffff)
	cs.SetA(float32(a) / 0xffff)

	// 原生像素偏移 1px 模拟粗体（不用 0.5 逻辑像素避免子像素模糊）
	for _, dx := range []float64{0, 1} {
		op := &text.DrawOptions{}
		op.GeoM.Translate(px+dx, py)
		op.ColorScale = cs
		text.Draw(screen, s, face, op)
	}
}

// DrawCenteredBoldText 绘制居中粗体文本。
func (fm *FontManager) DrawCenteredBoldText(screen *ebiten.Image, s string, centerX, y, size float64, clr color.Color) {
	w := fm.MeasureText(s, size)
	fm.DrawBoldText(screen, s, centerX-w/2, y, size, clr)
}

// DrawRightBoldText 绘制右对齐粗体文本。
func (fm *FontManager) DrawRightBoldText(screen *ebiten.Image, s string, rightX, y, size float64, clr color.Color) {
	w := fm.MeasureText(s, size)
	fm.DrawBoldText(screen, s, rightX-w, y, size, clr)
}

// MeasureText 测量文本渲染宽度（返回逻辑宽度）。
func (fm *FontManager) MeasureText(s string, size float64) float64 {
	sc := draw.Scale
	face := fm.Face(size * sc)
	w, _ := text.Measure(s, face, 0)
	return w / sc
}

// MeasureTextSize 测量文本渲染宽度和高度（返回逻辑尺寸）。
func (fm *FontManager) MeasureTextSize(s string, size float64) (float64, float64) {
	sc := draw.Scale
	face := fm.Face(size * sc)
	w, h := text.Measure(s, face, 0)
	return w / sc, h / sc
}

// DrawCenteredVText 在矩形区域内水平+垂直居中绘制文本。
func (fm *FontManager) DrawCenteredVText(screen *ebiten.Image, s string, cx, cy, size float64, clr color.Color) {
	w, h := fm.MeasureTextSize(s, size)
	fm.DrawText(screen, s, cx-w/2, cy-h/2, size, clr)
}

// DrawCenteredVBoldText 在矩形区域内水平+垂直居中绘制粗体文本。
func (fm *FontManager) DrawCenteredVBoldText(screen *ebiten.Image, s string, cx, cy, size float64, clr color.Color) {
	w, h := fm.MeasureTextSize(s, size)
	fm.DrawBoldText(screen, s, cx-w/2, cy-h/2, size, clr)
}

// ── Global singleton ──

var (
	globalFM   *FontManager
	globalOnce sync.Once
)

// InitGlobalFont 初始化全局字体管理器（单字体模式，兼容旧调用）。
func InitGlobalFont(ttfData []byte) error {
	var err error
	globalOnce.Do(func() {
		globalFM, err = NewFontManager(ttfData)
	})
	return err
}

// InitGlobalDualFont 初始化全局双字体管理器。
// primaryTTF: 英文/数字优先字体（如 JetBrains Mono）
// fallbackTTF: 中文回退字体（如 Noto Sans SC）
func InitGlobalDualFont(primaryTTF, fallbackTTF []byte) error {
	var err error
	globalOnce.Do(func() {
		globalFM, err = NewDualFontManager(primaryTTF, fallbackTTF)
	})
	return err
}

// GlobalFont 返回全局字体管理器。未初始化时返回 nil。
func GlobalFont() *FontManager {
	return globalFM
}
