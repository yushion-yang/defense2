// sprite.go — AI 精灵状态机。
//
// 管理精灵在地图上的位置和运动状态。
// 状态: Idle(巡视) → Moving(移动到目标) → Idle
// 渲染相关的视觉效果(bob动画等)也在此计算。
package aiplayer

import "math"

// SpriteState 精灵状态。
type SpriteState int

const (
	SpriteIdle      SpriteState = iota // 空闲巡视
	SpriteMoving                       // 移动到目标
	SpriteThinking                     // 思考中（决策前停顿）
	SpriteExecuting                    // 执行操作
)

const (
	spriteSpeed   = 120.0 // 移动速度 (px/s)
	bobAmplitude  = 3.0   // idle 上下浮动幅度
	bobFrequency  = 2.0   // 浮动频率 (Hz)
	arrivalThresh = 2.0   // 到达判定阈值
)

// Sprite AI 精灵。
type Sprite struct {
	x, y       float64
	targetX    float64
	targetY    float64
	state      SpriteState
	time       float64 // 累计时间（用于动画）
	stateTimer float64 // 当前状态剩余时间
}

// NewSprite 创建精灵。
func NewSprite(x, y float64) *Sprite {
	return &Sprite{
		x: x, y: y,
		targetX: x, targetY: y,
		state: SpriteIdle,
	}
}

func (s *Sprite) X() float64         { return s.x }
func (s *Sprite) Y() float64         { return s.y }
func (s *Sprite) State() SpriteState { return s.state }

// DrawY 返回渲染用 Y 坐标（含 bob 动画偏移）。
func (s *Sprite) DrawY() float64 {
	if s.state == SpriteIdle {
		return s.y + bobAmplitude*math.Sin(s.time*bobFrequency*2*math.Pi)
	}
	return s.y
}

// MoveTo 设置移动目标。
func (s *Sprite) MoveTo(x, y float64) {
	s.targetX = x
	s.targetY = y
	s.state = SpriteMoving
}

// SetThinking 进入思考状态（定时后回到 Idle）。
func (s *Sprite) SetThinking(duration float64) {
	s.state = SpriteThinking
	s.stateTimer = duration
}

// SetExecuting 进入执行状态（定时后回到 Idle）。
func (s *Sprite) SetExecuting(duration float64) {
	s.state = SpriteExecuting
	s.stateTimer = duration
}

// Tick 每帧更新。
func (s *Sprite) Tick(dt float64) {
	s.time += dt

	switch s.state {
	case SpriteIdle:
		// bob 动画由 DrawY 处理
	case SpriteMoving:
		s.tickMoving(dt)
	case SpriteThinking, SpriteExecuting:
		s.stateTimer -= dt
		if s.stateTimer <= 0 {
			s.state = SpriteIdle
		}
	}
}

func (s *Sprite) tickMoving(dt float64) {
	dx := s.targetX - s.x
	dy := s.targetY - s.y
	dist := math.Hypot(dx, dy)

	if dist < arrivalThresh {
		s.x = s.targetX
		s.y = s.targetY
		s.state = SpriteIdle
		return
	}

	step := spriteSpeed * dt
	if step >= dist {
		s.x = s.targetX
		s.y = s.targetY
		s.state = SpriteIdle
		return
	}

	s.x += dx / dist * step
	s.y += dy / dist * step
}
