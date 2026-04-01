// anim.go — 帧动画控制器。
// 管理多组命名动画（idle/attack/walk/hit 等），支持循环播放和单次播放。
package anim

import "github.com/hajimehoshi/ebiten/v2"

// Animation 单组动画（帧序列 + 播放参数）。
type Animation struct {
	Frames []*ebiten.Image // 帧图像序列
	FPS    float64         // 播放速率（帧/秒）
	Loop   bool            // 是否循环播放
}

// Animator 帧动画状态机。
type Animator struct {
	Anims    map[string]*Animation // 所有可用动画
	Current  string                // 当前动画名
	Frame    int                   // 当前帧索引
	Timer    float64               // 帧计时器（秒）
	Finished bool                  // 非循环动画是否已播放完
}

// NewAnimator 创建空动画控制器。
func NewAnimator() *Animator {
	return &Animator{
		Anims: make(map[string]*Animation),
	}
}

// AddAnim 注册一组动画。
func (a *Animator) AddAnim(name string, frames []*ebiten.Image, fps float64, loop bool) {
	a.Anims[name] = &Animation{Frames: frames, FPS: fps, Loop: loop}
	// 默认播放第一个添加的动画
	if a.Current == "" {
		a.Current = name
	}
}

// Play 切换到指定动画。如果已在播放则不重置。
func (a *Animator) Play(name string) {
	if a.Current == name && !a.Finished {
		return
	}
	if _, ok := a.Anims[name]; !ok {
		return
	}
	a.Current = name
	a.Frame = 0
	a.Timer = 0
	a.Finished = false
}

// PlayOnce 播放一次后切换回 fallback 动画。
func (a *Animator) PlayOnce(name, fallback string) {
	if _, ok := a.Anims[name]; !ok {
		return
	}
	a.Current = name
	a.Frame = 0
	a.Timer = 0
	a.Finished = false
}

// Update 推进动画帧。
func (a *Animator) Update(dt float64) {
	anim, ok := a.Anims[a.Current]
	if !ok || len(anim.Frames) == 0 || a.Finished {
		return
	}

	a.Timer += dt
	frameDur := 1.0 / anim.FPS
	if a.Timer >= frameDur {
		a.Timer -= frameDur
		a.Frame++
		if a.Frame >= len(anim.Frames) {
			if anim.Loop {
				a.Frame = 0
			} else {
				a.Frame = len(anim.Frames) - 1
				a.Finished = true
			}
		}
	}
}

// CurrentImage 返回当前帧图像，无动画时返回 nil。
func (a *Animator) CurrentImage() *ebiten.Image {
	anim, ok := a.Anims[a.Current]
	if !ok || len(anim.Frames) == 0 {
		return nil
	}
	idx := a.Frame
	if idx >= len(anim.Frames) {
		idx = len(anim.Frames) - 1
	}
	return anim.Frames[idx]
}

// HasAnim 检查是否有指定动画。
func (a *Animator) HasAnim(name string) bool {
	_, ok := a.Anims[name]
	return ok
}
