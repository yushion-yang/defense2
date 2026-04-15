// init.go — 描述符能力的双轨注册初始化。
//
// 在 InitConfigAbilities() 之后调用，加载描述符表并将每个描述符
// 注册为 tower.Ability，覆盖 ConfigAbility 的同名注册。
// 如果描述符加载失败，ConfigAbility 仍作为 fallback 生效。
//
// 关联：
//   - 调用方: scene/loading.go (phaseConfigs), scene/game.go (HeadlessMode)
//   - 加载器: loader.go LoadDescriptorTable
//   - 适配器: descriptor_ability.go NewDescriptorAbility
//   - 被覆盖: abilities/config_ability.go RegisterConfigAbilities
package descriptor

import (
	"fmt"
	"io/fs"
	"log"

	"defense2/internal/core/tower"
)

// RegisterCustomAbilities 将自定义能力注册到 tower.Registry。
// 在 stage 初始化时调用，确保蓝图引用的自定义能力在运行时可执行。
// 返回成功注册的能力数量。
func RegisterCustomAbilities(store *AbilityStore) int {
	if store == nil {
		return 0
	}
	registered := 0
	for _, ca := range store.List() {
		desc := ca.Desc
		desc.ID = ca.ID // 确保 ID 一致
		ability := NewDescriptorAbility(&desc)
		tower.Register(ability)
		registered++
	}
	if registered > 0 {
		log.Printf("[descriptor] registered %d custom abilities", registered)
	}
	return registered
}

// InitDescriptorAbilities 加载描述符表并将所有描述符注册为 tower.Ability。
// 覆盖 ConfigAbility 的注册。如果某个描述符解析失败，保留 ConfigAbility 并 log warning。
func InitDescriptorAbilities(dataFS fs.ReadFileFS) error {
	if err := LoadDescriptorTable(dataFS); err != nil {
		return fmt.Errorf("load descriptor table: %w", err)
	}

	table := GlobalDescriptorTable()
	registered := 0
	for _, desc := range table {
		ability := NewDescriptorAbility(desc)
		tower.Register(ability) // 覆盖 ConfigAbility 的同名注册
		registered++
	}

	log.Printf("[descriptor] registered %d descriptor abilities (overriding ConfigAbility)", registered)
	return nil
}
