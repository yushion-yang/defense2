// callgraph.go — 调用图生成器。
//
// 基于 Go AST 静态分析 internal/core/ 下的导出函数调用关系。
// 不做完整类型推导（太重），而是基于函数名 + 包选择器匹配。
// 结果按被调用方分组，展示"谁调用了我"和"我调用了谁"。

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// CallGraphStats 调用图统计
type CallGraphStats struct {
	FuncCount int
	EdgeCount int
}

// callEdge 一条调用关系
type callEdge struct {
	CallerPkg  string // 调用者的包短名
	CallerFunc string // 调用者函数名
	CallerFile string // 调用者文件（相对路径）
	CalleePkg  string // 被调用者的包选择器（可能是空字符串表示同包）
	CalleeFunc string // 被调用者函数名
}

// funcDecl 一个函数声明
type funcDecl struct {
	Pkg      string
	Name     string
	Receiver string // 方法接收者类型（无则为空）
	File     string
	Line     int
}

// CallGraphData 调用图分析的完整结果，供 impact 分析复用
type CallGraphData struct {
	CallerIndex map[string][]callEdge // 调用者 → 被调用者列表
	CalleeIndex map[string][]callEdge // 被调用者 → 调用者列表
	AllFuncs    []funcDecl
}

func generateCallGraph(root, outPath string) (CallGraphStats, CallGraphData, error) {
	coreDir := filepath.Join(root, "internal", "core")
	sceneDir := filepath.Join(root, "internal", "scene")

	// 收集所有函数声明和调用边
	var allFuncs []funcDecl
	var allEdges []callEdge

	scanDirs := []string{coreDir, sceneDir}
	for _, dir := range scanDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		funcs, edges, err := analyzeDir(root, dir)
		if err != nil {
			return CallGraphStats{}, CallGraphData{}, fmt.Errorf("分析 %s 失败: %w", dir, err)
		}
		allFuncs = append(allFuncs, funcs...)
		allEdges = append(allEdges, edges...)
	}

	// 构建"被调用者 → 调用者列表"索引
	calleeIndex := make(map[string][]callEdge) // key: "pkg.Func"
	for _, e := range allEdges {
		key := e.CalleeFunc
		if e.CalleePkg != "" {
			key = e.CalleePkg + "." + e.CalleeFunc
		}
		calleeIndex[key] = append(calleeIndex[key], e)
	}

	// 构建"调用者 → 被调用者列表"索引
	callerIndex := make(map[string][]callEdge) // key: "pkg.Func"
	for _, e := range allEdges {
		key := e.CallerPkg + "." + e.CallerFunc
		callerIndex[key] = append(callerIndex[key], e)
	}

	// 写入 Markdown
	f, err := os.Create(outPath)
	if err != nil {
		return CallGraphStats{}, CallGraphData{}, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# 核心调用图\n\n")
	fmt.Fprintf(f, "> 自动生成，勿手动编辑。运行 `make index` 更新。\n")
	fmt.Fprintf(f, "> 仅包含 internal/core/ 和 internal/scene/ 的导出函数。\n\n")

	// 按包分组输出函数列表 + 调用关系
	pkgFuncs := make(map[string][]funcDecl)
	for _, fd := range allFuncs {
		pkgFuncs[fd.Pkg] = append(pkgFuncs[fd.Pkg], fd)
	}

	pkgNames := make([]string, 0, len(pkgFuncs))
	for pkg := range pkgFuncs {
		pkgNames = append(pkgNames, pkg)
	}
	sort.Strings(pkgNames)

	// 高频被调用函数（热点）
	fmt.Fprintf(f, "## 热点函数（被调用 ≥3 次）\n\n")
	fmt.Fprintf(f, "| 函数 | 被调用次数 | 调用者 |\n")
	fmt.Fprintf(f, "|------|-----------|--------|\n")

	type hotFunc struct {
		name    string
		count   int
		callers []string
	}
	var hots []hotFunc
	for key, edges := range calleeIndex {
		if len(edges) >= 3 {
			callerSet := make(map[string]bool)
			for _, e := range edges {
				callerSet[e.CallerPkg+"."+e.CallerFunc] = true
			}
			callers := make([]string, 0, len(callerSet))
			for c := range callerSet {
				callers = append(callers, c)
			}
			sort.Strings(callers)
			hots = append(hots, hotFunc{key, len(edges), callers})
		}
	}
	sort.Slice(hots, func(i, j int) bool { return hots[i].count > hots[j].count })
	for _, h := range hots {
		callerStr := strings.Join(h.callers, ", ")
		if len(callerStr) > 100 {
			callerStr = callerStr[:97] + "..."
		}
		fmt.Fprintf(f, "| `%s` | %d | %s |\n", h.name, h.count, callerStr)
	}
	fmt.Fprintln(f)

	// 按包输出详细调用关系
	for _, pkg := range pkgNames {
		funcs := pkgFuncs[pkg]
		sort.Slice(funcs, func(i, j int) bool { return funcs[i].Name < funcs[j].Name })

		fmt.Fprintf(f, "## %s\n\n", pkg)

		for _, fd := range funcs {
			funcKey := fd.Pkg + "." + fd.Name
			callees := callerIndex[funcKey]
			callers := calleeIndex[fd.Name]
			// 也检查带包前缀的 key
			callers2 := calleeIndex[funcKey]
			if len(callers2) > len(callers) {
				callers = callers2
			}

			if len(callees) == 0 && len(callers) == 0 {
				continue // 跳过孤立函数
			}

			recv := ""
			if fd.Receiver != "" {
				recv = "(" + fd.Receiver + ") "
			}
			fmt.Fprintf(f, "### %s%s\n\n", recv, fd.Name)
			fmt.Fprintf(f, "📍 `%s:%d`\n\n", fd.File, fd.Line)

			if len(callees) > 0 {
				fmt.Fprintf(f, "**调用 →**\n")
				seen := make(map[string]bool)
				for _, e := range callees {
					target := e.CalleeFunc
					if e.CalleePkg != "" {
						target = e.CalleePkg + "." + e.CalleeFunc
					}
					if !seen[target] {
						fmt.Fprintf(f, "- `%s`\n", target)
						seen[target] = true
					}
				}
				fmt.Fprintln(f)
			}

			if len(callers) > 0 {
				fmt.Fprintf(f, "**← 被调用**\n")
				seen := make(map[string]bool)
				for _, e := range callers {
					caller := e.CallerPkg + "." + e.CallerFunc
					if !seen[caller] {
						fmt.Fprintf(f, "- `%s` (`%s`)\n", caller, e.CallerFile)
						seen[caller] = true
					}
				}
				fmt.Fprintln(f)
			}
		}
	}

	return CallGraphStats{
			FuncCount: len(allFuncs),
			EdgeCount: len(allEdges),
		}, CallGraphData{
			CallerIndex: callerIndex,
			CalleeIndex: calleeIndex,
			AllFuncs:    allFuncs,
		}, nil
}

// analyzeDir 分析目录下所有 Go 文件的函数声明和调用
func analyzeDir(root, dir string) ([]funcDecl, []callEdge, error) {
	var funcs []funcDecl
	var edges []callEdge

	fset := token.NewFileSet()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // 跳过解析失败的文件
		}

		relPath, _ := filepath.Rel(root, path)
		pkgName := file.Name.Name

		ast.Inspect(file, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// 只索引导出函数
			if !isExported(fd.Name.Name) {
				return true
			}

			recv := ""
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				recv = receiverType(fd.Recv.List[0].Type)
			}

			pos := fset.Position(fd.Pos())
			funcs = append(funcs, funcDecl{
				Pkg:      pkgName,
				Name:     fd.Name.Name,
				Receiver: recv,
				File:     relPath,
				Line:     pos.Line,
			})

			// 扫描函数体中的调用
			if fd.Body == nil {
				return true
			}
			ast.Inspect(fd.Body, func(cn ast.Node) bool {
				ce, ok := cn.(*ast.CallExpr)
				if !ok {
					return true
				}
				calleePkg, calleeFunc := extractCallTarget(ce)
				if calleeFunc == "" || !isExported(calleeFunc) {
					return true
				}
				edges = append(edges, callEdge{
					CallerPkg:  pkgName,
					CallerFunc: fd.Name.Name,
					CallerFile: relPath,
					CalleePkg:  calleePkg,
					CalleeFunc: calleeFunc,
				})
				return true
			})

			return true
		})
		return nil
	})

	return funcs, edges, err
}

// extractCallTarget 从 CallExpr 提取 [pkg.]func 名
func extractCallTarget(ce *ast.CallExpr) (pkg, funcName string) {
	switch fun := ce.Fun.(type) {
	case *ast.Ident:
		// 同包调用: FuncName()
		return "", fun.Name
	case *ast.SelectorExpr:
		// 跨包或方法调用: pkg.Func() 或 obj.Method()
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name, fun.Sel.Name
		}
	}
	return "", ""
}

// receiverType 提取方法接收者的类型名
func receiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
