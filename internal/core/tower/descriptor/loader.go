// loader.go — 能力描述符表加载器。
//
// 从 config/towers/ability-descriptors.json 加载所有能力描述符，
// 逐条调用 ParseDescriptor 编译为运行时结构，按 ID 索引存入全局表。
//
// 关联：
//   - 配置文件: config/towers/ability-descriptors.json（32 条描述符）
//   - 解析器: descriptor.go ParseDescriptor（两阶段解析 + pipeline 编译）
//   - 旧格式: config/ability_config.go LoadAbilityTable（abilities.json）
package descriptor

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// globalDescriptorTable 全局描述符表（id → *AbilityDescriptor）。
var globalDescriptorTable map[string]*AbilityDescriptor

// descriptorFileEnvelope JSON 文件顶层结构（含 _meta + descriptors 数组）。
type descriptorFileEnvelope struct {
	Descriptors []json.RawMessage `json:"descriptors"`
}

// LoadDescriptorTable 从 ability-descriptors.json 加载描述符表。
//
// 解析流程：
//  1. 读取 JSON 文件
//  2. 提取 descriptors 数组（忽略 _meta）
//  3. 逐条调用 ParseDescriptor 编译
//  4. 按 ID 索引存入全局表
func LoadDescriptorTable(dataFS fs.ReadFileFS) error {
	data, err := dataFS.ReadFile("config/towers/ability-descriptors.json")
	if err != nil {
		return fmt.Errorf("load ability-descriptors: %w", err)
	}

	var envelope descriptorFileEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("parse ability-descriptors envelope: %w", err)
	}

	table := make(map[string]*AbilityDescriptor, len(envelope.Descriptors))
	for i, raw := range envelope.Descriptors {
		desc, err := ParseDescriptor(raw)
		if err != nil {
			return fmt.Errorf("ability-descriptors[%d]: %w", i, err)
		}
		if _, dup := table[desc.ID]; dup {
			return fmt.Errorf("ability-descriptors: duplicate id %q", desc.ID)
		}
		table[desc.ID] = desc
	}

	globalDescriptorTable = table
	return nil
}

// GlobalDescriptorTable 返回全局描述符表。LoadDescriptorTable 成功后可用。
func GlobalDescriptorTable() map[string]*AbilityDescriptor {
	return globalDescriptorTable
}

// LookupDescriptor 按 ID 查找描述符。
func LookupDescriptor(id string) (*AbilityDescriptor, bool) {
	if globalDescriptorTable == nil {
		return nil, false
	}
	d, ok := globalDescriptorTable[id]
	return d, ok
}
