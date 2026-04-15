// blueprint_store.go — 蓝图持久化存储（CRUD）。
//
// BlueprintStore 封装 persistence.Storage，提供蓝图的增删改查操作。
// 数据以 JSON 序列化到单个 key "tower_blueprints" 下。
// 最多保存 20 个蓝图。
//
// 关联：
//   - persistence/storage.go — Storage 接口（Get/Set/Has/Delete）
//   - blueprint.go — TowerBlueprint 数据结构
package descriptor

import (
	"fmt"

	"defense2/internal/core/persistence"
)

const maxBlueprints = 20
const blueprintsKey = "tower_blueprints"

// blueprintStoreData 持久化数据结构。
type blueprintStoreData struct {
	Blueprints []TowerBlueprint `json:"blueprints"`
	Version    int              `json:"version"`
}

// BlueprintStore 蓝图持久化存储。
type BlueprintStore struct {
	storage persistence.Storage
	data    blueprintStoreData
}

// NewBlueprintStore 创建蓝图存储，从 Storage 加载已有数据。
// 如果 storage 中无数据或读取失败，初始化为空列表。
func NewBlueprintStore(s persistence.Storage) *BlueprintStore {
	bs := &BlueprintStore{
		storage: s,
		data: blueprintStoreData{
			Blueprints: []TowerBlueprint{},
			Version:    1,
		},
	}

	// 尝试从 storage 加载已有数据
	if s.Has(blueprintsKey) {
		var loaded blueprintStoreData
		if err := s.Get(blueprintsKey, &loaded); err == nil && loaded.Blueprints != nil {
			bs.data = loaded
		}
	}

	return bs
}

// Save 保存蓝图（新增或更新同 ID 的蓝图）。
// 同 ID 已存在时替换（更新），否则追加（新增）。
// 新增时若数量已达 maxBlueprints 则返回错误。
func (bs *BlueprintStore) Save(bp TowerBlueprint) error {
	// 检查是否是更新已有蓝图
	for i, existing := range bs.data.Blueprints {
		if existing.ID == bp.ID {
			bs.data.Blueprints[i] = bp
			return bs.persist()
		}
	}

	// 新增：检查上限
	if len(bs.data.Blueprints) >= maxBlueprints {
		return fmt.Errorf("已达蓝图上限 %d，无法新增", maxBlueprints)
	}

	bs.data.Blueprints = append(bs.data.Blueprints, bp)
	return bs.persist()
}

// Get 按 ID 查找蓝图，返回副本。
// 找不到时返回错误。
func (bs *BlueprintStore) Get(id string) (*TowerBlueprint, error) {
	for _, bp := range bs.data.Blueprints {
		if bp.ID == id {
			// 返回副本，深拷贝 map 和 slice 防止外部修改泄漏
			copy := bp
			copy.Tiers = copyStringMap(bp.Tiers)
			copy.Abilities = copyStringSlice(bp.Abilities)
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("蓝图 %q 不存在", id)
}

// Delete 删除蓝图，找不到时返回错误。
func (bs *BlueprintStore) Delete(id string) error {
	for i, bp := range bs.data.Blueprints {
		if bp.ID == id {
			// 保持顺序删除
			bs.data.Blueprints = append(bs.data.Blueprints[:i], bs.data.Blueprints[i+1:]...)
			return bs.persist()
		}
	}
	return fmt.Errorf("蓝图 %q 不存在，无法删除", id)
}

// List 返回所有蓝图的副本列表。
// 空时返回空 slice（不是 nil）。
func (bs *BlueprintStore) List() []TowerBlueprint {
	result := make([]TowerBlueprint, len(bs.data.Blueprints))
	for i, bp := range bs.data.Blueprints {
		cp := bp
		cp.Tiers = copyStringMap(bp.Tiers)
		cp.Abilities = copyStringSlice(bp.Abilities)
		result[i] = cp
	}
	return result
}

// Count 返回蓝图数量。
func (bs *BlueprintStore) Count() int {
	return len(bs.data.Blueprints)
}

// persist 将当前数据写入 storage。
func (bs *BlueprintStore) persist() error {
	return bs.storage.Set(blueprintsKey, bs.data)
}

// copyStringMap 浅拷贝 map[string]string。
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// copyStringSlice 拷贝 []string。
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}
