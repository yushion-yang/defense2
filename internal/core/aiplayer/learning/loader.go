// loader.go — 权重模型加载器。
//
// 从嵌入式配置文件系统加载预训练权重。
// 加载失败时自动 fallback 到代码内硬编码默认值（DefaultModel）。
package learning

import "embed"

// weightsPath 权重配置文件路径。
const weightsPath = "config/ai/weights.json"

// LoadFromFS 从嵌入式文件系统加载权重模型。
// dataFS 来自 config.GetDataFS()，由调用方传入避免 import 循环。
// 加载失败时返回 DefaultModel。
func LoadFromFS(dataFS *embed.FS) *Model {
	if dataFS == nil {
		return DefaultModel()
	}
	data, err := dataFS.ReadFile(weightsPath)
	if err != nil {
		return DefaultModel()
	}
	return LoadModel(data)
}
