// main.go — 项目索引生成器入口。
//
// 生成三种 Markdown 索引供 AI 辅助开发使用：
//   - files.md     文件职责表（包/文件/行数/头注释）
//   - callgraph.md 核心模块导出函数调用图
//   - configmap.md JSON 配置字段 → Go 消费者映射
//
// 用法: go run tools/indexer/main.go [-out docs/index]

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "docs/index", "输出目录")
	flag.Parse()

	// 定位项目根目录（从 go.mod 位置推断）
	root, err := findProjectRoot()
	if err != nil {
		log.Fatalf("无法定位项目根目录: %v", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	start := time.Now()

	// 1. 文件职责表
	fmt.Println("==> 生成文件职责表...")
	fileStats, err := generateFileIndex(root, filepath.Join(*outDir, "files.md"))
	if err != nil {
		log.Fatalf("生成文件索引失败: %v", err)
	}
	fmt.Printf("    %d 个文件, %d 行代码\n", fileStats.FileCount, fileStats.TotalLines)

	// 2. 调用图
	fmt.Println("==> 生成调用图...")
	cgStats, cgData, err := generateCallGraph(root, filepath.Join(*outDir, "callgraph.md"))
	if err != nil {
		log.Fatalf("生成调用图失败: %v", err)
	}
	fmt.Printf("    %d 个函数, %d 条调用关系\n", cgStats.FuncCount, cgStats.EdgeCount)

	// 3. 影响链分析（基于调用图数据）
	fmt.Println("==> 生成影响链...")
	impStats, err := generateImpact(root, filepath.Join(*outDir, "impact.md"), cgData.CallerIndex, cgData.CalleeIndex, cgData.AllFuncs)
	if err != nil {
		log.Fatalf("生成影响链失败: %v", err)
	}
	fmt.Printf("    %d 条影响链\n", impStats.ChainsCount)

	// 4. 配置映射
	fmt.Println("==> 生成配置映射...")
	cmStats, err := generateConfigMap(root, filepath.Join(*outDir, "configmap.md"))
	if err != nil {
		log.Fatalf("生成配置映射失败: %v", err)
	}
	fmt.Printf("    %d 个 JSON 文件, %d 个字段映射\n", cmStats.JSONFiles, cmStats.Mappings)

	// 5. 摘要
	elapsed := time.Since(start)
	writeSummary(*outDir, fileStats, cgStats, impStats, cmStats, elapsed)
	fmt.Printf("==> 完成 (%.1fs), 输出: %s/\n", elapsed.Seconds(), *outDir)
}

// findProjectRoot 向上查找包含 go.mod 的目录
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到 go.mod")
		}
		dir = parent
	}
}

func writeSummary(outDir string, fs FileStats, cg CallGraphStats, imp ImpactStats, cm ConfigMapStats, elapsed time.Duration) {
	f, err := os.Create(filepath.Join(outDir, "summary.md"))
	if err != nil {
		log.Printf("写入 summary 失败: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# 索引摘要\n\n")
	fmt.Fprintf(f, "生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "| 指标 | 值 |\n|------|---|\n")
	fmt.Fprintf(f, "| Go 文件数 | %d |\n", fs.FileCount)
	fmt.Fprintf(f, "| 总代码行数 | %d |\n", fs.TotalLines)
	fmt.Fprintf(f, "| 包数量 | %d |\n", fs.PackageCount)
	fmt.Fprintf(f, "| 索引函数数 | %d |\n", cg.FuncCount)
	fmt.Fprintf(f, "| 调用关系数 | %d |\n", cg.EdgeCount)
	fmt.Fprintf(f, "| 影响链数 | %d |\n", imp.ChainsCount)
	fmt.Fprintf(f, "| JSON 配置文件 | %d |\n", cm.JSONFiles)
	fmt.Fprintf(f, "| 配置字段映射 | %d |\n", cm.Mappings)
	fmt.Fprintf(f, "| 生成耗时 | %.1fs |\n", elapsed.Seconds())
	fmt.Fprintf(f, "\n## 文件列表\n\n")
	fmt.Fprintf(f, "- [files.md](files.md) — 文件职责表\n")
	fmt.Fprintf(f, "- [callgraph.md](callgraph.md) — 核心调用图\n")
	fmt.Fprintf(f, "- [impact.md](impact.md) — 影响链分析（2 层）\n")
	fmt.Fprintf(f, "- [configmap.md](configmap.md) — 配置→代码映射\n")
}
