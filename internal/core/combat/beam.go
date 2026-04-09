// beam.go — 光束数据与对象池。
// Slice-compaction pool. Chosen for: low entity count (beams are rare,
// typically <10 active), simpler code than fixed-size pools, and no
// fixed capacity limit — the slice grows on demand and compacts each frame.
// Beam 是纯视觉对象，用于 wideBeam 的渲染（无碰撞逻辑）。
package combat

// Beam 光束视觉对象。
type Beam struct {
	X1, Y1  float64  // 起点
	X2, Y2  float64  // 终点
	Width   float64  // 宽度（像素）
	Color   [3]uint8 // 颜色 RGB
	Life    float64  // 剩余显示时间（秒）
	MaxLife float64  // 总显示时间（秒）
	Wide    bool     // 宽光束标志（wideBeam）
}

// Alpha 返回当前透明度 (0-1)。
func (b *Beam) Alpha() float64 {
	if b.MaxLife <= 0 {
		return 0
	}
	return b.Life / b.MaxLife
}

// BeamPool 光束对象池。
type BeamPool struct {
	beams []Beam
}

// NewBeamPool 创建光束池。
func NewBeamPool() *BeamPool {
	return &BeamPool{
		beams: make([]Beam, 0, 32),
	}
}

// Add 添加一个光束。
func (p *BeamPool) Add(b Beam) {
	p.beams = append(p.beams, b)
}

// Update 更新所有光束生命周期，移除过期的。
func (p *BeamPool) Update(dt float64) {
	n := 0
	for i := range p.beams {
		p.beams[i].Life -= dt
		if p.beams[i].Life > 0 {
			p.beams[n] = p.beams[i]
			n++
		}
	}
	p.beams = p.beams[:n]
}

// Each 遍历所有存活光束。
func (p *BeamPool) Each(fn func(b *Beam)) {
	for i := range p.beams {
		fn(&p.beams[i])
	}
}

// Count 返回存活光束数。
func (p *BeamPool) Count() int {
	return len(p.beams)
}
