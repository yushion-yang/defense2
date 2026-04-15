// ability_store.go — 自定义能力持久化存储（CRUD）。
//
// AbilityStore 封装 persistence.Storage，提供自定义能力的增删改查操作。
// 数据以 JSON 序列化到单个 key "custom_abilities" 下。
// 最多保存 50 个自定义能力。
//
// 关联：
//   - persistence/storage.go — Storage 接口（Get/Set/Has/Delete）
//   - descriptor.go — AbilityDescriptor 数据结构
//   - blueprint_store.go — 同模式的蓝图存储（参考实现）
package descriptor

import (
	"encoding/json"
	"fmt"

	"defense2/internal/core/persistence"
)

const maxCustomAbilities = 50
const customAbilitiesKey = "custom_abilities"

// CustomAbility 玩家自定义能力。
type CustomAbility struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Desc      AbilityDescriptor `json:"descriptor"`
	CreatedAt string            `json:"createdAt,omitempty"`
}

// abilityStoreData 持久化数据结构。
type abilityStoreData struct {
	Abilities []CustomAbility `json:"abilities"`
	Version   int             `json:"version"`
}

// AbilityStore 自定义能力持久化存储。
type AbilityStore struct {
	storage persistence.Storage
	data    abilityStoreData
}

// NewAbilityStore 创建能力存储，从 Storage 加载已有数据。
// 如果 storage 中无数据或读取失败，初始化为空列表。
func NewAbilityStore(s persistence.Storage) *AbilityStore {
	as := &AbilityStore{
		storage: s,
		data: abilityStoreData{
			Abilities: []CustomAbility{},
			Version:   1,
		},
	}

	// 尝试从 storage 加载已有数据
	if s.Has(customAbilitiesKey) {
		var loaded abilityStoreData
		if err := s.Get(customAbilitiesKey, &loaded); err == nil && loaded.Abilities != nil {
			as.data = loaded
		}
	}

	return as
}

// Save 保存自定义能力（新增或更新同 ID 的能力）。
// 同 ID 已存在时替换（更新），否则追加（新增）。
// 新增时若数量已达 maxCustomAbilities 则返回错误。
func (as *AbilityStore) Save(ca CustomAbility) error {
	// 检查是否是更新已有能力
	for i, existing := range as.data.Abilities {
		if existing.ID == ca.ID {
			as.data.Abilities[i] = ca
			return as.persist()
		}
	}

	// 新增：检查上限
	if len(as.data.Abilities) >= maxCustomAbilities {
		return fmt.Errorf("已达自定义能力上限 %d，无法新增", maxCustomAbilities)
	}

	as.data.Abilities = append(as.data.Abilities, ca)
	return as.persist()
}

// Get 按 ID 查找自定义能力，返回副本。
// 找不到时返回错误。
func (as *AbilityStore) Get(id string) (*CustomAbility, error) {
	for _, ca := range as.data.Abilities {
		if ca.ID == id {
			cp := copyCustomAbility(ca)
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("自定义能力 %q 不存在", id)
}

// Delete 删除自定义能力，找不到时返回错误。
func (as *AbilityStore) Delete(id string) error {
	for i, ca := range as.data.Abilities {
		if ca.ID == id {
			// 保持顺序删除
			as.data.Abilities = append(as.data.Abilities[:i], as.data.Abilities[i+1:]...)
			return as.persist()
		}
	}
	return fmt.Errorf("自定义能力 %q 不存在，无法删除", id)
}

// List 返回所有自定义能力的副本列表。
// 空时返回空 slice（不是 nil）。
func (as *AbilityStore) List() []CustomAbility {
	result := make([]CustomAbility, len(as.data.Abilities))
	for i, ca := range as.data.Abilities {
		result[i] = copyCustomAbility(ca)
	}
	return result
}

// Count 返回自定义能力数量。
func (as *AbilityStore) Count() int {
	return len(as.data.Abilities)
}

// persist 将当前数据写入 storage。
func (as *AbilityStore) persist() error {
	return as.storage.Set(customAbilitiesKey, as.data)
}

// copyCustomAbility 深拷贝 CustomAbility，防止外部修改泄漏到 store 内部。
func copyCustomAbility(ca CustomAbility) CustomAbility {
	cp := ca
	cp.Desc.Tags = copyStringSlice(ca.Desc.Tags)
	cp.Desc.AttackParams = copyRawMessage(ca.Desc.AttackParams)
	cp.Desc.Pipelines = copyPipelines(ca.Desc.Pipelines)
	return cp
}

// copyRawMessage 拷贝 json.RawMessage（底层为 []byte）。
func copyRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	cp := make(json.RawMessage, len(raw))
	copy(cp, raw)
	return cp
}

// copyPipelines 浅拷贝 Pipeline 切片。
// Pipeline 内的 Condition/Selector/Effect 是值语义接口，无需深拷贝。
func copyPipelines(ps []Pipeline) []Pipeline {
	if ps == nil {
		return nil
	}
	cp := make([]Pipeline, len(ps))
	copy(cp, ps)
	return cp
}
