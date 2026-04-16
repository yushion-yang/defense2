// impact.go — 影响链分析器（2 层间接调用）。
//
// 对每个核心函数，展开上下游 2 层调用链：
//   - 上游链（被调用链）: 改了 A，谁会受影响？A ← B ← C
//   - 下游链（调用链）: A 调用了谁？A → B → C
// 输出独立的 impact.md，聚焦跨模块影响分析。

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImpactStats 影响链统计
type ImpactStats struct {
	ChainsCount int // 输出的影响链数量
}

// generateImpact 生成影响链分析文件
func generateImpact(root, outPath string, callerIndex, calleeIndex map[string][]callEdge, allFuncs []funcDecl) (ImpactStats, error) {
	f, err := os.Create(filepath.Join(filepath.Dir(outPath), "impact.md"))
	if err != nil {
		return ImpactStats{}, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# 影响链分析（2 层）\n\n")
	fmt.Fprintf(f, "> 自动生成，勿手动编辑。运行 `make index` 更新。\n")
	fmt.Fprintf(f, "> 修改函数前查此表，评估影响范围。\n\n")

	// 构建双向邻接表（去重）
	// callsOut[A] = {B, C}  表示 A 调用了 B 和 C
	// calledBy[A] = {D, E}  表示 A 被 D 和 E 调用
	callsOut := make(map[string]map[string]bool)
	calledBy := make(map[string]map[string]bool)

	for key, edges := range callerIndex {
		if callsOut[key] == nil {
			callsOut[key] = make(map[string]bool)
		}
		for _, e := range edges {
			target := e.CalleeFunc
			if e.CalleePkg != "" {
				target = e.CalleePkg + "." + e.CalleeFunc
			}
			callsOut[key][target] = true
		}
	}

	for key, edges := range calleeIndex {
		if calledBy[key] == nil {
			calledBy[key] = make(map[string]bool)
		}
		for _, e := range edges {
			caller := e.CallerPkg + "." + e.CallerFunc
			calledBy[key][caller] = true
		}
	}

	// 只对跨包函数生成影响链（同包内调用不太需要影响分析）
	// 筛选：被 ≥2 个不同包调用的函数
	type impactTarget struct {
		funcKey  string
		pkgCount int
	}
	var targets []impactTarget

	for key, callers := range calledBy {
		// 跳过标准库和第三方包函数（对影响分析无意义）
		if isStdlibCall(key) {
			continue
		}
		pkgs := make(map[string]bool)
		for caller := range callers {
			parts := strings.SplitN(caller, ".", 2)
			if len(parts) == 2 {
				pkgs[parts[0]] = true
			}
		}
		if len(pkgs) >= 2 {
			targets = append(targets, impactTarget{key, len(pkgs)})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].pkgCount != targets[j].pkgCount {
			return targets[i].pkgCount > targets[j].pkgCount
		}
		return targets[i].funcKey < targets[j].funcKey
	})

	fmt.Fprintf(f, "共 %d 个跨包函数有影响链。\n\n", len(targets))

	// 生成影响链
	chainsCount := 0
	for _, t := range targets {
		fmt.Fprintf(f, "## %s\n\n", t.funcKey)

		// 上游 2 层：谁调用了我，谁又调用了调用者
		upstreams := expandUpstream(t.funcKey, calledBy, 2)
		if len(upstreams) > 0 {
			fmt.Fprintf(f, "**上游影响链**（修改此函数，以下调用者受影响）：\n\n")
			fmt.Fprintf(f, "```\n")
			for _, chain := range upstreams {
				fmt.Fprintf(f, "%s\n", chain)
				chainsCount++
			}
			fmt.Fprintf(f, "```\n\n")
		}

		// 下游 2 层：我调用了谁，谁又调用了谁
		downstreams := expandDownstream(t.funcKey, callsOut, 2)
		if len(downstreams) > 0 {
			fmt.Fprintf(f, "**下游依赖链**（此函数依赖以下函数）：\n\n")
			fmt.Fprintf(f, "```\n")
			for _, chain := range downstreams {
				fmt.Fprintf(f, "%s\n", chain)
				chainsCount++
			}
			fmt.Fprintf(f, "```\n\n")
		}
	}

	return ImpactStats{ChainsCount: chainsCount}, nil
}

// expandUpstream BFS 展开上游调用链（最多 depth 层）
// 返回格式: ["caller1 → caller2 → target", ...]
func expandUpstream(target string, calledBy map[string]map[string]bool, depth int) []string {
	type chain struct {
		path []string
	}

	var results []string
	queue := []chain{{path: []string{target}}}

	for level := 0; level < depth; level++ {
		var nextQueue []chain
		for _, c := range queue {
			head := c.path[len(c.path)-1]
			callers := calledBy[head]
			if len(callers) == 0 {
				// 到达叶子，如果链长>1 就输出
				if len(c.path) > 1 {
					results = append(results, formatChainReverse(c.path))
				}
				continue
			}
			for caller := range callers {
				// 避免循环
				if containsStr(c.path, caller) {
					continue
				}
				newPath := make([]string, len(c.path)+1)
				copy(newPath, c.path)
				newPath[len(c.path)] = caller
				nextQueue = append(nextQueue, chain{path: newPath})
			}
		}
		if len(nextQueue) == 0 {
			break
		}
		queue = nextQueue
	}

	// 输出最终队列中的链（到达 depth 层）
	for _, c := range queue {
		if len(c.path) > 1 {
			results = append(results, formatChainReverse(c.path))
		}
	}

	sort.Strings(results)
	// 限制输出量
	if len(results) > 30 {
		results = append(results[:30], fmt.Sprintf("... 共 %d 条链（截断显示 30 条）", len(results)))
	}
	return results
}

// expandDownstream BFS 展开下游调用链（最多 depth 层）
func expandDownstream(source string, callsOut map[string]map[string]bool, depth int) []string {
	type chain struct {
		path []string
	}

	var results []string
	queue := []chain{{path: []string{source}}}

	for level := 0; level < depth; level++ {
		var nextQueue []chain
		for _, c := range queue {
			tail := c.path[len(c.path)-1]
			callees := callsOut[tail]
			if len(callees) == 0 {
				if len(c.path) > 1 {
					results = append(results, formatChain(c.path))
				}
				continue
			}
			for callee := range callees {
				if containsStr(c.path, callee) {
					continue
				}
				newPath := make([]string, len(c.path)+1)
				copy(newPath, c.path)
				newPath[len(c.path)] = callee
				nextQueue = append(nextQueue, chain{path: newPath})
			}
		}
		if len(nextQueue) == 0 {
			break
		}
		queue = nextQueue
	}

	for _, c := range queue {
		if len(c.path) > 1 {
			results = append(results, formatChain(c.path))
		}
	}

	sort.Strings(results)
	if len(results) > 30 {
		results = append(results[:30], fmt.Sprintf("... 共 %d 条链（截断显示 30 条）", len(results)))
	}
	return results
}

// formatChainReverse 将 [target, caller1, caller2] 格式化为 "caller2 → caller1 → target"
func formatChainReverse(path []string) string {
	reversed := make([]string, len(path))
	for i, p := range path {
		reversed[len(path)-1-i] = p
	}
	return strings.Join(reversed, " → ")
}

// formatChain 将 [source, callee1, callee2] 格式化为 "source → callee1 → callee2"
func formatChain(path []string) string {
	return strings.Join(path, " → ")
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// isStdlibCall 判断函数 key 是否为标准库/第三方包调用
func isStdlibCall(key string) bool {
	// 按包名前缀判断。我们的代码包名是短单词(combat/tower/scene/enemy...)，
	// 标准库包名也是短单词但有限，直接枚举常见的。
	stdPkgs := map[string]bool{
		"fmt": true, "log": true, "os": true, "math": true,
		"json": true, "strings": true, "strconv": true, "sort": true,
		"filepath": true, "time": true, "sync": true, "bytes": true,
		"io": true, "rand": true, "errors": true, "context": true,
		"slices": true, "cmp": true, "maps": true, "regexp": true,
		// 第三方
		"ebiten": true, "inpututil": true, "vector": true, "audio": true,
	}
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 2 {
		return stdPkgs[parts[0]]
	}
	return false
}
