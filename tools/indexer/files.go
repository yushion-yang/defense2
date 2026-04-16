// files.go — 文件职责表生成器。
//
// 扫描 internal/ 和 cmd/ 下所有 .go 文件，提取：
//   - 包名
//   - 行数
//   - 头注释第一行（"// filename — 描述" 格式）
// 输出按包分组的 Markdown 表格。

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileStats 文件索引统计
type FileStats struct {
	FileCount    int
	TotalLines   int
	PackageCount int
}

// fileEntry 单个文件的索引信息
type fileEntry struct {
	RelPath string // 相对项目根目录的路径
	Package string // 包名
	Lines   int    // 行数
	Summary string // 头注释摘要
}

func generateFileIndex(root, outPath string) (FileStats, error) {
	var entries []fileEntry
	scanDirs := []string{"internal", "cmd"}

	for _, dir := range scanDirs {
		absDir := filepath.Join(root, dir)
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			continue
		}
		err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 跳过无法访问的文件
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			// 跳过测试文件
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			entry, err := parseFileEntry(root, path)
			if err != nil {
				return nil // 跳过解析失败的文件
			}
			entries = append(entries, entry)
			return nil
		})
		if err != nil {
			return FileStats{}, fmt.Errorf("扫描 %s 失败: %w", dir, err)
		}
	}

	// 按包分组
	pkgMap := make(map[string][]fileEntry)
	totalLines := 0
	for _, e := range entries {
		pkgMap[e.Package] = append(pkgMap[e.Package], e)
		totalLines += e.Lines
	}

	// 包名排序
	pkgNames := make([]string, 0, len(pkgMap))
	for pkg := range pkgMap {
		pkgNames = append(pkgNames, pkg)
	}
	sort.Strings(pkgNames)

	// 写入 Markdown
	f, err := os.Create(outPath)
	if err != nil {
		return FileStats{}, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# 文件职责表\n\n")
	fmt.Fprintf(f, "> 自动生成，勿手动编辑。运行 `make index` 更新。\n\n")
	fmt.Fprintf(f, "共 %d 个文件, %d 行代码, %d 个包。\n\n", len(entries), totalLines, len(pkgNames))

	for _, pkg := range pkgNames {
		files := pkgMap[pkg]
		sort.Slice(files, func(i, j int) bool {
			return files[i].RelPath < files[j].RelPath
		})

		// 计算包总行数
		pkgLines := 0
		for _, fe := range files {
			pkgLines += fe.Lines
		}

		fmt.Fprintf(f, "## %s (%d 行)\n\n", pkg, pkgLines)
		fmt.Fprintf(f, "| 文件 | 行数 | 职责 |\n")
		fmt.Fprintf(f, "|------|------|------|\n")
		for _, fe := range files {
			fileName := filepath.Base(fe.RelPath)
			summary := fe.Summary
			if summary == "" {
				summary = "-"
			}
			// 转义 Markdown 管道符
			summary = strings.ReplaceAll(summary, "|", "\\|")
			fmt.Fprintf(f, "| `%s` | %d | %s |\n", fileName, fe.Lines, summary)
		}
		fmt.Fprintln(f)
	}

	return FileStats{
		FileCount:    len(entries),
		TotalLines:   totalLines,
		PackageCount: len(pkgNames),
	}, nil
}

// parseFileEntry 解析单个 Go 文件的基本信息
func parseFileEntry(root, path string) (fileEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileEntry{}, err
	}
	defer f.Close()

	relPath, _ := filepath.Rel(root, path)
	scanner := bufio.NewScanner(f)

	var (
		pkg     string
		summary string
		lines   int
		inFirst = true // 是否还在文件开头的注释块
	)

	for scanner.Scan() {
		lines++
		line := scanner.Text()

		// 提取包名
		if strings.HasPrefix(line, "package ") {
			pkg = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			inFirst = false
			continue
		}

		// 提取头注释第一行中的描述
		if inFirst && summary == "" && strings.HasPrefix(line, "//") {
			comment := strings.TrimPrefix(line, "//")
			comment = strings.TrimSpace(comment)
			// 匹配 "filename — 描述" 或 "filename - 描述" 格式
			if idx := strings.Index(comment, "—"); idx > 0 {
				summary = strings.TrimSpace(comment[idx+len("—"):])
			} else if idx := strings.Index(comment, " - "); idx > 0 {
				summary = strings.TrimSpace(comment[idx+3:])
			}
			// 截断过长描述
			if len(summary) > 80 {
				summary = summary[:77] + "..."
			}
		}

		// 遇到非注释行（且不是空行）结束头注释扫描
		if inFirst && !strings.HasPrefix(line, "//") && strings.TrimSpace(line) != "" {
			inFirst = false
		}
	}

	return fileEntry{
		RelPath: relPath,
		Package: pkg,
		Lines:   lines,
		Summary: summary,
	}, nil
}
