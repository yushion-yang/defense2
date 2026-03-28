// data.go — 嵌入式数据入口。
// 通过 go:embed 将 config/ 和 assets/ 下的资源文件编译进二进制，供运行时加载。
package defense2

import "embed"

//go:embed config/levels/*.json config/towers/*.json
var DataFS embed.FS

//go:embed assets/towers/*/*.svg
var AssetFS embed.FS
