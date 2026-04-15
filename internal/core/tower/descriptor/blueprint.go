// blueprint.go — 定制炮塔蓝图数据结构与校验。
//
// TowerBlueprint 是玩家自定义塔的完整描述：攻击方式、属性档位、专精、能力列表。
// 由预算系统（budget.go）约束总强度，由 ValidateBlueprint 校验合法性。
//
// StrengthConfig 定义强度升级规则，与 config.StrengthConfig 结构相同但解耦，
// 避免 descriptor 包依赖 config 包。
package descriptor

import "fmt"

// StrengthConfig 强度升级配置。
// 与 config.StrengthConfig 字段一致，独立定义以保持 descriptor 包零外部依赖。
type StrengthConfig struct {
	Cost         int     `json:"cost"`         // 单次购买费用
	Amount       float64 `json:"amount"`       // 单次增加的强度值
	MaxPurchases int     `json:"maxPurchases"` // 最大购买次数（-1=无限）
}

// TowerBlueprint 定制炮塔蓝图。
// 完整描述一个玩家自定义塔的所有配置项。
type TowerBlueprint struct {
	// ── 身份信息 ──
	ID        string `json:"id"`
	Name      string `json:"name"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt,omitempty"`

	// ── 攻击方式 ──
	AttackStyle string `json:"attackStyle"`
	SpriteKey   string `json:"spriteKey,omitempty"`

	// ── 属性档位 ──
	// key: damage/atkSpeed/range，value: S/B/D
	Tiers map[string]string `json:"tiers"`

	// ── 专精属性 ──
	// damage/atkSpeed/range 之一，空字符串表示无专精
	Specialty string `json:"specialty"`

	// ── 能力列表 ──
	Abilities []string `json:"abilities"`

	// ── 经济 ──
	BuildCost int            `json:"buildCost"`
	Strength  StrengthConfig `json:"strength"`
}

// ValidationError 校验错误条目。
type ValidationError struct {
	Field   string // 出错的字段
	Message string // 错误描述
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// validTiers 合法的属性档位。
var validTiers = map[string]bool{"S": true, "B": true, "D": true}

// validSpecialties 合法的专精属性。
var validSpecialties = map[string]bool{
	"":         true,
	"damage":   true,
	"atkSpeed": true,
	"range":    true,
}

// ValidateBlueprint 校验蓝图合法性，返回所有发现的错误。
// 检查项：
//  1. 所有 tier 值必须是 S/B/D
//  2. specialty 必须是 damage/atkSpeed/range 或空
//  3. 能力数量不超过 maxSlots
//  4. 无重复能力
//  5. 预算不超上限
func ValidateBlueprint(bp TowerBlueprint, rules BudgetRules, abilityCosts map[string]int) []ValidationError {
	var errs []ValidationError

	// 检查档位合法性
	for attr, tier := range bp.Tiers {
		if !validTiers[tier] {
			errs = append(errs, ValidationError{
				Field:   "tiers",
				Message: fmt.Sprintf("invalid tier %q for %s", tier, attr),
			})
		}
	}

	// 检查专精合法性
	if !validSpecialties[bp.Specialty] {
		errs = append(errs, ValidationError{
			Field:   "specialty",
			Message: fmt.Sprintf("invalid specialty %q", bp.Specialty),
		})
	}

	// 检查能力数量
	if len(bp.Abilities) > rules.MaxSlots {
		errs = append(errs, ValidationError{
			Field:   "abilities",
			Message: fmt.Sprintf("too many abilities: %d > %d", len(bp.Abilities), rules.MaxSlots),
		})
	}

	// 检查重复能力
	seen := map[string]bool{}
	for _, ab := range bp.Abilities {
		if seen[ab] {
			errs = append(errs, ValidationError{
				Field:   "abilities",
				Message: fmt.Sprintf("duplicate ability: %s", ab),
			})
			break
		}
		seen[ab] = true
	}

	// 检查预算
	result := CalcBudget(bp, rules, abilityCosts)
	if result.Used > result.Cap {
		errs = append(errs, ValidationError{
			Field:   "budget",
			Message: fmt.Sprintf("over budget: %d > %d", result.Used, result.Cap),
		})
	}

	return errs
}
