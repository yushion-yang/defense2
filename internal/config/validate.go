// validate.go — 配置校验。
// 对塔、敌人、事件等配置进行范围和完整性校验，
// 确保 JSON 数据不会导致运行时异常。
package config

import "fmt"

// ValidationError 配置校验错误。
type ValidationError struct {
	Field   string      // 字段路径
	Value   interface{} // 实际值
	Message string      // 错误描述
}

// Error 实现 error 接口。
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (值=%v)", e.Field, e.Message, e.Value)
}

// ValidateTowerDef 校验塔配置。
// 检查项：label 非空、buildCost 10-200、baseRange 50-10000、baseDamage >=0、baseFireRate 0.18-99。
func ValidateTowerDef(t *TowerJSON) []ValidationError {
	var errs []ValidationError

	if t.Label == "" {
		errs = append(errs, ValidationError{
			Field:   "label",
			Value:   t.Label,
			Message: "标签不能为空",
		})
	}

	if t.BuildCost < 10 || t.BuildCost > 200 {
		errs = append(errs, ValidationError{
			Field:   "buildCost",
			Value:   t.BuildCost,
			Message: "建造费用必须在 10-200 范围内",
		})
	}

	// baseRange=0 表示由 tier-presets 动态分配（如 basic 塔）
	if t.BaseRange != 0 && (t.BaseRange < 50 || t.BaseRange > 10000) {
		errs = append(errs, ValidationError{
			Field:   "baseRange",
			Value:   t.BaseRange,
			Message: "基础射程必须在 50-10000 范围内（0=动态分配）",
		})
	}

	if t.BaseDamage < 0 {
		errs = append(errs, ValidationError{
			Field:   "baseDamage",
			Value:   t.BaseDamage,
			Message: "基础伤害不能为负",
		})
	}

	if t.BaseAttackSpeed < 0 || t.BaseAttackSpeed > 20 {
		errs = append(errs, ValidationError{
			Field:   "baseAttackSpeed",
			Value:   t.BaseAttackSpeed,
			Message: "基础攻速必须在 0-20 范围内",
		})
	}

	return errs
}

// ValidateEnemyDef 校验敌人原型配置。
// 检查项：label 非空、hpScale > 0.1、speedScale >= 0、rewardScale >= 0。
func ValidateEnemyDef(e *EnemyArchetype) []ValidationError {
	var errs []ValidationError

	if e.Label == "" {
		errs = append(errs, ValidationError{
			Field:   "label",
			Value:   e.Label,
			Message: "标签不能为空",
		})
	}

	if e.HPScale <= 0.1 {
		errs = append(errs, ValidationError{
			Field:   "hpScale",
			Value:   e.HPScale,
			Message: "血量倍率必须大于 0.1",
		})
	}

	if e.SpeedScale < 0 {
		errs = append(errs, ValidationError{
			Field:   "speedScale",
			Value:   e.SpeedScale,
			Message: "速度倍率不能为负",
		})
	}

	if e.RewardScale < 0 {
		errs = append(errs, ValidationError{
			Field:   "rewardScale",
			Value:   e.RewardScale,
			Message: "奖励倍率不能为负",
		})
	}

	return errs
}

