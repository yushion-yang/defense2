// strength.go — 塔的战力(Strength)数据系统。
//
// 战力是本游戏属性计算的核心变量。每座塔的最终属性由战力驱动：
//
//	attr = Base + Potential × (Strength / 100)
//
// 战力自身分三层：
//  1. Base（固定 100）— 每座塔的起始战力
//  2. Permanent（永久加成）— 战灵、全局事件等持久效果
//  3. Temp（临时加成，按 sourceID 索引）— 链式加成、光环 buff 等，波结束清除
//
// 敌人端还有两种减益层叠加在上面：
//   - EnemyMul（乘法减益）— drainer 等敌人行为造成的百分比削弱
//   - EnemySub（减法减益）— 固定值削减
//
// 设计决策：
//   - map[string]float64 按 sourceID 索引而非按类型累加，
//     这样移除特定来源时不影响其他来源（如链断开不影响光环）
//   - ClearTransient 用内置 clear() 复用 map 内存，避免每波重分配
//   - Strength 保持线性（不加收益递减），通过敌人端克制做平衡
package strength

// StrengthData 实体战力数据（三层结构）。
type StrengthData struct {
	Base      float64            // 基础战力（固定100）
	Permanent float64            // 永久加成（全局持久）
	Temp      map[string]float64 // 临时加成（sourceID → amount，波结束清除）
	EnemyMul  map[string]float64 // 敌人乘法减益（sourceID → factor，<1为减益）
	EnemySub  map[string]float64 // 敌人减法减益（sourceID → amount）
}

// NewStrengthData 创建默认战力数据（Base=100）。
func NewStrengthData() *StrengthData {
	return &StrengthData{
		Base:     100,
		Temp:     make(map[string]float64),
		EnemyMul: make(map[string]float64),
		EnemySub: make(map[string]float64),
	}
}

// Effective 返回有效战力。
// 公式: max(0, (Base + Permanent + sum(Temp)) * product(EnemyMul) - sum(EnemySub))
func (s *StrengthData) Effective() float64 {
	// 基础 + 永久 + 临时总和
	raw := s.Base + s.Permanent
	for _, v := range s.Temp {
		raw += v
	}

	// 敌人乘法减益
	for _, factor := range s.EnemyMul {
		raw *= factor
	}

	// 敌人减法减益
	for _, amount := range s.EnemySub {
		raw -= amount
	}

	// 不低于 0
	if raw < 0 {
		return 0
	}
	return raw
}

// Ratio 返回有效战力与基准值(100)的比值。
// 100 战力返回 1.0，200 战力返回 2.0，50 战力返回 0.5。
// 用于通用属性缩放: effectiveAttr = base + potential * Ratio()
func (s *StrengthData) Ratio() float64 {
	return s.Effective() / 100.0
}

// Overflow 返回超出基准值的部分（负值时为0）。
// 用于战灵感知强度计算: Σmax(0, tower.Effective() - 100)
func (s *StrengthData) Overflow() float64 {
	eff := s.Effective()
	if eff <= 100 {
		return 0
	}
	return eff - 100
}

// AddPermanent 增加永久加成（可为负数）。
// Permanent 下限为 -Base，确保 Base+Permanent >= 0。
func (s *StrengthData) AddPermanent(amount float64) {
	s.Permanent += amount
	if s.Permanent < -s.Base {
		s.Permanent = -s.Base
	}
}

// ResetPermanent 清零永久加成。
func (s *StrengthData) ResetPermanent() {
	s.Permanent = 0
}

// SetTemp 设置临时加成（按 sourceID 覆盖）。
func (s *StrengthData) SetTemp(sourceID string, amount float64) {
	s.Temp[sourceID] = amount
}

// RemoveTemp 移除指定来源的临时加成。
func (s *StrengthData) RemoveTemp(sourceID string) {
	delete(s.Temp, sourceID)
}

// SetEnemyMul 设置敌人乘法减益。factor >= 1 时自动移除。
func (s *StrengthData) SetEnemyMul(sourceID string, factor float64) {
	if factor >= 1 {
		delete(s.EnemyMul, sourceID)
		return
	}
	s.EnemyMul[sourceID] = factor
}

// SetEnemySub 设置敌人减法减益。amount <= 0 时自动移除。
func (s *StrengthData) SetEnemySub(sourceID string, amount float64) {
	if amount <= 0 {
		delete(s.EnemySub, sourceID)
		return
	}
	s.EnemySub[sourceID] = amount
}

// RemoveEnemyDebuffs 移除指定来源的所有敌人减益（乘法 + 减法）。
func (s *StrengthData) RemoveEnemyDebuffs(sourceID string) {
	delete(s.EnemyMul, sourceID)
	delete(s.EnemySub, sourceID)
}

// ClearTransient 清除所有临时数据（Temp + EnemyMul + EnemySub），保留 Base + Permanent。
// 返回 true 如果有数据被清除（用于脏标记优化）。
func (s *StrengthData) ClearTransient() bool {
	dirty := len(s.Temp) > 0 || len(s.EnemyMul) > 0 || len(s.EnemySub) > 0
	clear(s.Temp)
	clear(s.EnemyMul)
	clear(s.EnemySub)
	return dirty
}
