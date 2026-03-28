// passive.go — 被动技能系统（充能式终极技能）。
// 支持时间充能、命中充能、击杀充能三种来源，充满后可手动释放。
package skill

// PassiveSkillDef 被动技能定义。
type PassiveSkillDef struct {
	Type         string  // 技能类型标识
	ChargeMax    float64 // 最大充能值
	ChargePerSec float64 // 每秒自动充能
	ChargeOnHit  float64 // 每次命中充能
	ChargeOnKill float64 // 每次击杀充能
}

// PassiveSkillState 被动技能运行时状态。
type PassiveSkillState struct {
	Def    PassiveSkillDef // 技能定义
	Charge float64         // 当前充能值
	Ready  bool            // 是否充满可释放
}

// ── 4 种预设被动技能默认参数 ──

var passiveDefaults = map[string]PassiveSkillDef{
	"missile-barrage": {
		Type:         "missile-barrage",
		ChargeMax:    100,
		ChargePerSec: 8,
		ChargeOnHit:  0,
		ChargeOnKill: 0,
	},
	"judgment-beam": {
		Type:         "judgment-beam",
		ChargeMax:    120,
		ChargePerSec: 6,
		ChargeOnHit:  0,
		ChargeOnKill: 0,
	},
	"chain-lightning": {
		Type:         "chain-lightning",
		ChargeMax:    120,
		ChargePerSec: 8,
		ChargeOnHit:  0,
		ChargeOnKill: 0,
	},
	"judgment-rain": {
		Type:         "judgment-rain",
		ChargeMax:    150,
		ChargePerSec: 6,
		ChargeOnHit:  0,
		ChargeOnKill: 0,
	},
}

// GetPassiveDefaults 获取预设被动技能默认参数。
// 找不到返回零值 PassiveSkillDef。
func GetPassiveDefaults(typeName string) PassiveSkillDef {
	if def, ok := passiveDefaults[typeName]; ok {
		return def
	}
	return PassiveSkillDef{Type: typeName}
}

// NewPassiveSkill 创建被动技能运行时状态。
func NewPassiveSkill(def PassiveSkillDef) *PassiveSkillState {
	return &PassiveSkillState{
		Def:    def,
		Charge: 0,
		Ready:  false,
	}
}

// Tick 每帧更新充能（时间充能）。
func (p *PassiveSkillState) Tick(dt float64) {
	if p.Ready {
		return // 已满不再充能
	}
	p.Charge += p.Def.ChargePerSec * dt
	if p.Charge >= p.Def.ChargeMax {
		p.Charge = p.Def.ChargeMax
		p.Ready = true
	}
}

// AddCharge 增加充能（命中/击杀充能）。
func (p *PassiveSkillState) AddCharge(amount float64) {
	if p.Ready {
		return
	}
	p.Charge += amount
	if p.Charge >= p.Def.ChargeMax {
		p.Charge = p.Def.ChargeMax
		p.Ready = true
	}
}

// TryRelease 尝试释放技能。如果已充满则重置充能并返回 true。
func (p *PassiveSkillState) TryRelease() bool {
	if !p.Ready {
		return false
	}
	p.Charge = 0
	p.Ready = false
	return true
}

// Progress 返回充能进度（0~1）。
func (p *PassiveSkillState) Progress() float64 {
	if p.Def.ChargeMax <= 0 {
		return 0
	}
	ratio := p.Charge / p.Def.ChargeMax
	if ratio > 1 {
		ratio = 1
	}
	return ratio
}
