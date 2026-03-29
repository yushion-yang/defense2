// strength.go — 战力数据系统。
// 三层结构：基础 + 永久加成 + 临时加成，外加敌人减益层。
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
func (s *StrengthData) AddPermanent(amount float64) {
	s.Permanent += amount
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
func (s *StrengthData) ClearTransient() {
	s.Temp = make(map[string]float64)
	s.EnemyMul = make(map[string]float64)
	s.EnemySub = make(map[string]float64)
}
