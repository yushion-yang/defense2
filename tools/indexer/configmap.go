// configmap.go — JSON 配置字段 → Go 代码消费者映射生成器。
//
// 两阶段匹配策略：
//   1. 扫描 Go 代码中的 dataFS.ReadFile("config/xxx.json") 调用，
//      建立 Go 文件 → JSON 文件的关联关系（高可信度）
//   2. 在关联的 Go 文件中搜索 struct tag 和字符串字面量
//   3. 在非关联文件中只搜索 struct tag（低噪声）
//   4. 按 (JSON文件, key, Go文件) 去重

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ConfigMapStats 配置映射统计
type ConfigMapStats struct {
	JSONFiles int
	Mappings  int
}

// configMapping 一条配置→代码映射
type configMapping struct {
	JSONFile string // JSON 文件相对路径
	Key      string // JSON 字段路径
	GoFile   string // Go 消费者文件
	GoLine   int    // 行号
	Context  string // 匹配行内容（截断）
}

// jsonKey JSON 配置文件中的一个字段
type jsonKey struct {
	file string // 相对路径
	path string // 字段路径
}

// readFilePattern 匹配 dataFS.ReadFile("config/...") 调用
var readFilePattern = regexp.MustCompile(`(?:dataFS|assetFS)\.ReadFile\("(config/[^"]+)"\)`)

func generateConfigMap(root, outPath string) (ConfigMapStats, error) {
	configDir := filepath.Join(root, "config")
	internalDir := filepath.Join(root, "internal")

	// 1. 收集 JSON 文件中的 key
	var keys []jsonKey
	jsonCount := 0

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		// 跳过不需要索引的目录
		for _, skip := range []string{"scenarios/", "i18n/", "mascot/", "autoplay/"} {
			if strings.Contains(relPath, skip) {
				return nil
			}
		}

		jsonCount++
		for _, k := range extractJSONKeys(path) {
			keys = append(keys, jsonKey{file: relPath, path: k})
		}
		return nil
	})
	if err != nil {
		return ConfigMapStats{}, fmt.Errorf("扫描 config/ 失败: %w", err)
	}

	// 2. 构建待搜索的 key 表（过滤噪声）
	// key: JSON 字段最后一段 → 对应的 jsonKey 列表
	searchKeys := make(map[string][]jsonKey)
	for _, jk := range keys {
		parts := strings.Split(jk.path, ".")
		lastKey := parts[len(parts)-1]
		if len(lastKey) <= 5 || strings.HasPrefix(lastKey, "_") || isGenericKey(lastKey) {
			continue
		}
		searchKeys[lastKey] = append(searchKeys[lastKey], jk)
	}

	// 3. 扫描 Go 文件，建立 Go 文件 → JSON 文件关联
	// goFileToJSONs["internal/config/enemy_config.go"] = {"config/enemies/enemies-core.json": true}
	goFileToJSONs := make(map[string]map[string]bool)

	err = filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		goRel, _ := filepath.Rel(root, path)
		refs := findReadFileRefs(path)
		if len(refs) > 0 {
			goFileToJSONs[goRel] = make(map[string]bool)
			for _, ref := range refs {
				goFileToJSONs[goRel][ref] = true
			}
		}
		return nil
	})
	if err != nil {
		return ConfigMapStats{}, fmt.Errorf("扫描 Go 文件关联失败: %w", err)
	}

	// 4. 构建 JSON 文件 → Go loader 文件的反向索引
	// 如果一个 JSON 文件已经有关联的 Go loader，则只在关联文件中搜索（精确模式）
	jsonHasLoader := make(map[string]bool) // "config/enemies/abilities.json" → true
	for _, jsons := range goFileToJSONs {
		for jsonFile := range jsons {
			jsonHasLoader[jsonFile] = true
		}
	}

	// 5. 搜索映射
	var mappings []configMapping
	dedup := make(map[string]bool) // key: "jsonFile|jsonKey|goFile"

	err = filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		goRel, _ := filepath.Rel(root, path)
		linkedJSONs := goFileToJSONs[goRel]
		isConfigPkg := strings.Contains(goRel, "internal/config/")

		found := searchFileForKeys(path, goRel, searchKeys, isConfigPkg, linkedJSONs, jsonHasLoader)
		for _, m := range found {
			dedupKey := m.JSONFile + "|" + m.Key + "|" + m.GoFile
			if !dedup[dedupKey] {
				dedup[dedupKey] = true
				mappings = append(mappings, m)
			}
		}
		return nil
	})
	if err != nil {
		return ConfigMapStats{}, fmt.Errorf("扫描 internal/ 失败: %w", err)
	}

	// 5. 写入 Markdown
	f, err := os.Create(outPath)
	if err != nil {
		return ConfigMapStats{}, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# 配置 → 代码映射\n\n")
	fmt.Fprintf(f, "> 自动生成，勿手动编辑。运行 `make index` 更新。\n\n")
	fmt.Fprintf(f, "扫描 %d 个 JSON 配置文件，找到 %d 条映射。\n\n", jsonCount, len(mappings))

	// 按 JSON 文件分组
	byJSON := make(map[string][]configMapping)
	for _, m := range mappings {
		byJSON[m.JSONFile] = append(byJSON[m.JSONFile], m)
	}

	jsonFiles := make([]string, 0, len(byJSON))
	for jf := range byJSON {
		jsonFiles = append(jsonFiles, jf)
	}
	sort.Strings(jsonFiles)

	for _, jf := range jsonFiles {
		ms := byJSON[jf]
		sort.Slice(ms, func(i, j int) bool {
			if ms[i].Key != ms[j].Key {
				return ms[i].Key < ms[j].Key
			}
			return ms[i].GoFile < ms[j].GoFile
		})

		fmt.Fprintf(f, "## %s\n\n", jf)
		fmt.Fprintf(f, "| JSON 字段 | Go 文件 | 行 | 上下文 |\n")
		fmt.Fprintf(f, "|-----------|---------|-----|--------|\n")
		for _, m := range ms {
			ctx := m.Context
			if len(ctx) > 60 {
				ctx = ctx[:57] + "..."
			}
			ctx = strings.ReplaceAll(ctx, "|", "\\|")
			fmt.Fprintf(f, "| `%s` | `%s` | %d | %s |\n", m.Key, m.GoFile, m.GoLine, ctx)
		}
		fmt.Fprintln(f)
	}

	return ConfigMapStats{
		JSONFiles: jsonCount,
		Mappings:  len(mappings),
	}, nil
}

// findReadFileRefs 从 Go 文件中提取 dataFS.ReadFile("config/...") 引用的 JSON 路径
func findReadFileRefs(goPath string) []string {
	data, err := os.ReadFile(goPath)
	if err != nil {
		return nil
	}
	matches := readFilePattern.FindAllStringSubmatch(string(data), -1)
	var refs []string
	for _, m := range matches {
		if len(m) >= 2 {
			refs = append(refs, m[1])
		}
	}
	return refs
}

// extractJSONKeys 提取 JSON 文件的顶层和二级 key
func extractJSONKeys(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var keys []string
	for k, v := range raw {
		keys = append(keys, k)
		var nested map[string]json.RawMessage
		if json.Unmarshal(v, &nested) == nil {
			for nk := range nested {
				keys = append(keys, k+"."+nk)
			}
		}
	}
	return keys
}

// searchFileForKeys 在 Go 文件中搜索 JSON key 的引用。
//
// 匹配策略：
//   - struct tag `json:"key"` — 始终搜索
//   - 字符串字面量 "key" — 仅当 isConfigPkg=true 时搜索
//
// 过滤策略：
//   - 如果 Go 文件有 linkedJSONs：只为关联的 JSON 文件生成映射
//   - 如果 Go 文件没有 linkedJSONs：只为没有 loader 的 JSON 文件生成映射
//     （已有 loader 的 JSON 文件应该只出现在对应 loader 的结果中）
func searchFileForKeys(goPath, goRel string, searchKeys map[string][]jsonKey, isConfigPkg bool, linkedJSONs map[string]bool, jsonHasLoader map[string]bool) []configMapping {
	f, err := os.Open(goPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var results []configMapping
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for key, jks := range searchKeys {
			if !containsKeyReference(line, key, isConfigPkg) {
				continue
			}
			for _, jk := range jks {
				if len(linkedJSONs) > 0 {
					// 精确模式：Go 文件有关联，只输出关联 JSON 的映射
					if !linkedJSONs[jk.file] {
						continue
					}
				} else {
					// 宽泛模式：Go 文件无关联，只输出没有 loader 的 JSON 的映射
					// 已有 loader 的 JSON 不应出现在无关文件的结果中
					if jsonHasLoader[jk.file] {
						continue
					}
				}
				results = append(results, configMapping{
					JSONFile: jk.file,
					Key:      jk.path,
					GoFile:   goRel,
					GoLine:   lineNum,
					Context:  strings.TrimSpace(line),
				})
			}
		}
	}

	return results
}

// containsKeyReference 检查行中是否包含对 key 的精确引用
func containsKeyReference(line, key string, allowStringLiteral bool) bool {
	// struct tag: `json:"key"` 或 `json:"key,omitempty"`
	if strings.Contains(line, `json:"`+key+`"`) || strings.Contains(line, `json:"`+key+`,`) {
		return true
	}
	// 字符串字面量: 仅在 config 包中启用
	if allowStringLiteral {
		if strings.Contains(line, `"`+key+`"`) {
			return true
		}
	}
	return false
}

// isGenericKey 检查是否为通用编程词汇
func isGenericKey(key string) bool {
	generic := map[string]bool{
		"format": true, "string": true, "number": true, "boolean": true,
		"object": true, "default": true, "description": true, "enabled": true,
		"active": true, "status": true, "values": true, "config": true,
		"options": true, "params": true, "result": true, "output": true,
		"source": true, "target": true, "action": true, "update": true,
		"create": true, "delete": true, "remove": true, "filter": true,
		"offset": true, "length": true, "height": true, "weight": true,
		"volume": true, "repeat": true, "duration": true, "position": true,
		"playback": true, "version": true, "category": true, "display": true,
	}
	return generic[strings.ToLower(key)]
}
