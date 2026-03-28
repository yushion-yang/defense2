// font.go — 字体管理与文本渲染工具。
// 基于 Ebitengine text/v2 提供中文文本渲染能力。
package render

import (
	"bytes"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// FontManager 管理 TTF 字体源并缓存不同尺寸的字体。
type FontManager struct {
	source *text.GoTextFaceSource
	mu     sync.Mutex
	faces  map[float64]*text.GoTextFace
}

// NewFontManager 从 TTF 字节数据创建字体管理器。
func NewFontManager(ttfData []byte) (*FontManager, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttfData))
	if err != nil {
		return nil, err
	}
	return &FontManager{
		source: src,
		faces:  make(map[float64]*text.GoTextFace),
	}, nil
}

// Face 返回指定尺寸的字体（缓存复用）。
func (fm *FontManager) Face(size float64) *text.GoTextFace {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if f, ok := fm.faces[size]; ok {
		return f
	}
	f := &text.GoTextFace{Source: fm.source, Size: size}
	fm.faces[size] = f
	return f
}

// DrawText 在指定位置绘制文本（左上角对齐）。
func (fm *FontManager) DrawText(screen *ebiten.Image, s string, x, y, size float64, clr color.Color) {
	face := fm.Face(size)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
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

// MeasureText 测量文本渲染宽度。
func (fm *FontManager) MeasureText(s string, size float64) float64 {
	face := fm.Face(size)
	w, _ := text.Measure(s, face, 0)
	return w
}

// ── Global singleton ──

var (
	globalFM   *FontManager
	globalOnce sync.Once
)

// InitGlobalFont 初始化全局字体管理器（只执行一次）。
func InitGlobalFont(ttfData []byte) error {
	var err error
	globalOnce.Do(func() {
		globalFM, err = NewFontManager(ttfData)
	})
	return err
}

// GlobalFont 返回全局字体管理器。未初始化时返回 nil。
func GlobalFont() *FontManager {
	return globalFM
}
