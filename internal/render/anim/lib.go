// lib.go — 动画帧库（共享）+ 轻量播放状态（per-instance）。
// AnimLib 按原型缓存帧图像，AnimPlayState 是无依赖的播放状态。
package anim

import "github.com/hajimehoshi/ebiten/v2"

// AnimLib 帧库：只存帧图像，不存播放状态。按原型共享。
type AnimLib struct {
	Anims map[string]*Animation // "walk", "hit", etc.
}

// NewAnimLib 创建空帧库。
func NewAnimLib() *AnimLib {
	return &AnimLib{Anims: make(map[string]*Animation)}
}

// HasAnim 检查是否有指定动画。
func (lib *AnimLib) HasAnim(name string) bool {
	_, ok := lib.Anims[name]
	return ok
}

// Frame 返回指定动画的第 idx 帧，越界或不存在返回 nil。
func (lib *AnimLib) Frame(name string, idx int) *ebiten.Image {
	a, ok := lib.Anims[name]
	if !ok || len(a.Frames) == 0 {
		return nil
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(a.Frames) {
		idx = len(a.Frames) - 1
	}
	return a.Frames[idx]
}

// FrameCount 返回指定动画的帧数。
func (lib *AnimLib) FrameCount(name string) int {
	a, ok := lib.Anims[name]
	if !ok {
		return 0
	}
	return len(a.Frames)
}

// FPS 返回指定动画的播放速率。
func (lib *AnimLib) FPS(name string) float64 {
	a, ok := lib.Anims[name]
	if !ok {
		return 1
	}
	return a.FPS
}

// ---------------------------------------------------------------------------
// AnimPlayState — 轻量播放状态，挂在每个实体上。
// 不引用 anim 包以外的类型，可嵌入 enemy.Enemy 等纯逻辑 struct。
// ---------------------------------------------------------------------------

// AnimPlayState 独立的动画播放状态。
type AnimPlayState struct {
	Current  string  // 当前动画名
	FrameIdx int     // 当前帧索引
	Timer    float64 // 帧计时器（秒）
	Finished bool    // 非循环动画是否播完
}

// Play 切换动画（已在播放则不重置）。
func (s *AnimPlayState) Play(name string) {
	if s.Current == name && !s.Finished {
		return
	}
	s.Current = name
	s.FrameIdx = 0
	s.Timer = 0
	s.Finished = false
}

// Update 推进播放状态。fps 和 loop 由外部传入（来自 AnimLib）。
func (s *AnimPlayState) Update(dt, fps float64, loop bool, frameCount int) {
	if s.Finished || frameCount == 0 {
		return
	}
	s.Timer += dt
	frameDur := 1.0 / fps
	if s.Timer >= frameDur {
		s.Timer -= frameDur
		s.FrameIdx++
		if s.FrameIdx >= frameCount {
			if loop {
				s.FrameIdx = 0
			} else {
				s.FrameIdx = frameCount - 1
				s.Finished = true
			}
		}
	}
}
