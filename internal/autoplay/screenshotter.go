// screenshotter.go — 关键帧截图管理器。
// 管理截图请求队列，实际 PNG 编码在 scene 层完成。
package autoplay

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// Screenshotter 截图管理器。
type Screenshotter struct {
	outputDir string
	pending   []string // 等待捕获的文件名
	captured  []string // 已捕获的文件名
}

// NewScreenshotter 创建截图管理器。
func NewScreenshotter(outputDir string) *Screenshotter {
	return &Screenshotter{
		outputDir: outputDir,
	}
}

// RequestCapture 请求捕获一张截图。
func (s *Screenshotter) RequestCapture(filename string) {
	s.pending = append(s.pending, filename)
}

// HasPending 检查是否有待截图请求（不消费队列）。
func (s *Screenshotter) HasPending() bool {
	return len(s.pending) > 0
}

// NextPending 获取并弹出下一个待捕获的文件名，无则返回空字符串。
func (s *Screenshotter) NextPending() string {
	if len(s.pending) == 0 {
		return ""
	}
	name := s.pending[0]
	s.pending = s.pending[1:]
	s.captured = append(s.captured, name)
	return name
}

// CapturedFiles 返回所有已捕获的文件名列表。
func (s *Screenshotter) CapturedFiles() []string {
	return s.captured
}

// RequestStart 请求游戏开始截图。
func (s *Screenshotter) RequestStart() {
	s.RequestCapture("start.png")
}

// RequestWave 请求波次截图（每 5 波）。
func (s *Screenshotter) RequestWave(wave int) {
	s.RequestCapture(fmt.Sprintf("wave_%d.png", wave))
}

// RequestBossWave 请求 Boss 波截图。
func (s *Screenshotter) RequestBossWave(wave int) {
	s.RequestCapture(fmt.Sprintf("boss_wave_%d.png", wave))
}

// RequestLeak 请求泄漏截图。
func (s *Screenshotter) RequestLeak(wave, tick int) {
	s.RequestCapture(fmt.Sprintf("leak_wave_%d_%d.png", wave, tick))
}

// RequestResult 请求结果截图。
func (s *Screenshotter) RequestResult() {
	s.RequestCapture("result.png")
}

// RequestAnomaly 请求异常截图。
func (s *Screenshotter) RequestAnomaly(anomalyType string, tick int) {
	s.RequestCapture(fmt.Sprintf("anomaly_%s_%d.png", anomalyType, tick))
}

// SaveImage 将 image.Image 保存为 PNG 文件。
func SaveImage(img image.Image, dir, filename string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}
