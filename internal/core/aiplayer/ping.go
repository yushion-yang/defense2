// ping.go -- Ping 交互系统。
//
// 玩家点击 AI 区域格子发送建议信号，AI 评估并回应（同意/拒绝/延迟）。
// 设计：冷却机制防止刷屏，AI 根据 Compliance 个性 + 位置评分决定是否采纳。
package aiplayer

// PingRequest 玩家发送的 ping 请求。
type PingRequest struct {
	Row, Col int
	X, Y     float64
	Time     float64 // 发送时间
}

// PingState ping 系统状态。
type PingState struct {
	pending   *PingRequest // 待处理的 ping
	cooldown  float64      // 冷却剩余时间
	responded bool         // 是否已回应
}

const pingCooldown = 10.0 // 冷却 10 秒

// NewPingState 创建 ping 状态。
func NewPingState() *PingState {
	return &PingState{}
}

// Send 发送 ping（冷却中返回 false）。
func (p *PingState) Send(row, col int, x, y float64) bool {
	if p.cooldown > 0 {
		return false
	}
	p.pending = &PingRequest{Row: row, Col: col, X: x, Y: y}
	p.cooldown = pingCooldown
	p.responded = false
	return true
}

// Tick 每帧更新冷却。
func (p *PingState) Tick(dt float64) {
	if p.cooldown > 0 {
		p.cooldown -= dt
	}
}

// HasPending 是否有待处理的 ping。
func (p *PingState) HasPending() bool {
	return p.pending != nil && !p.responded
}

// Consume 消费待处理的 ping，返回请求内容。
func (p *PingState) Consume() *PingRequest {
	if p.pending == nil || p.responded {
		return nil
	}
	p.responded = true
	req := p.pending
	p.pending = nil
	return req
}

// OnCooldown 是否在冷却中。
func (p *PingState) OnCooldown() bool {
	return p.cooldown > 0
}
