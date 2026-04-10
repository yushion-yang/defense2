// buff_config.go — buff 堆叠规则配置加载。
// 从 config/systems/buff-stack.json 读取规则并调用 buff.InitGlobalRules 初始化全局单例。
package config

import (
	"fmt"

	"defense2/internal/core/buff"
)

// LoadBuffRules 从 config/systems/buff-stack.json 加载 buff 堆叠规则。
// 必须在 SetDataFS() 之后调用。
func LoadBuffRules() error {
	if dataFS == nil {
		return fmt.Errorf("load buff rules: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/systems/buff-stack.json")
	if err != nil {
		return fmt.Errorf("load buff rules: %w", err)
	}
	if err := buff.InitGlobalRules(data); err != nil {
		return fmt.Errorf("init buff rules: %w", err)
	}
	return nil
}
