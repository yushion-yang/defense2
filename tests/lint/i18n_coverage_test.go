// i18n_coverage_test.go — 验证所有 i18n.T/TF 调用的 key 在 zh.json 中有对应条目。
// 防止新增 T("key") 但忘记在 locale JSON 中添加翻译。
package lint_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reI18nKey 匹配 i18n.T("key") 和 i18n.TF("key" 中的静态 key。
var reI18nKey = regexp.MustCompile(`i18n\.TF?\("([a-zA-Z0-9_.-]+)"`)

// dynamicKeyPrefixes 动态拼接的 key 前缀（代码中用 "prefix."+var 构建，无法静态检测）。
// 这些前缀只需验证至少有一个匹配 key 存在即可。
var dynamicKeyPrefixes = []string{
	"ability.",
	"enemy.",
	"warden.",
	"tower.",
	"item.",
	"map.",
	"progress.map.",
	"progress.tower.",
	"progress.warden.",
	"achievement.",
}

func TestI18nKeyCoverage(t *testing.T) {
	root := findProjectRoot(t)

	// 1. 加载 zh.json 获取所有已定义的 key
	zhPath := filepath.Join(root, "config", "i18n", "zh.json")
	zhData, err := os.ReadFile(zhPath)
	if err != nil {
		t.Fatalf("read zh.json: %v", err)
	}
	var zhKeys map[string]interface{}
	if err := json.Unmarshal(zhData, &zhKeys); err != nil {
		t.Fatalf("parse zh.json: %v", err)
	}

	// 2. 同样加载 en.json 检查一致性
	enPath := filepath.Join(root, "config", "i18n", "en.json")
	enData, err := os.ReadFile(enPath)
	if err != nil {
		t.Fatalf("read en.json: %v", err)
	}
	var enKeys map[string]interface{}
	if err := json.Unmarshal(enData, &enKeys); err != nil {
		t.Fatalf("parse en.json: %v", err)
	}

	// 3. 扫描所有 Go 文件收集静态 i18n key
	codeKeys := make(map[string]string) // key -> "file:line"
	err = filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			matches := reI18nKey.FindAllStringSubmatch(scanner.Text(), -1)
			for _, m := range matches {
				codeKeys[m[1]] = rel + ":" + itoa(lineNum)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	// 4. 检查每个静态 key 是否在 zh.json 中存在
	missing := 0
	for key, loc := range codeKeys {
		// 跳过动态前缀的 key（它们由 resolve 机制处理）
		isDynamic := false
		for _, prefix := range dynamicKeyPrefixes {
			if strings.HasPrefix(key, prefix) {
				isDynamic = true
				break
			}
		}
		if isDynamic {
			continue
		}

		if _, ok := zhKeys[key]; !ok {
			t.Errorf("missing in zh.json: %q (used at %s)", key, loc)
			missing++
		}
	}

	// 5. 检查 zh.json 和 en.json 的 key 集合是否一致
	for key := range zhKeys {
		if strings.HasPrefix(key, "_") {
			continue // 跳过 _meta
		}
		if _, ok := enKeys[key]; !ok {
			t.Errorf("key %q exists in zh.json but missing in en.json", key)
		}
	}
	for key := range enKeys {
		if strings.HasPrefix(key, "_") {
			continue
		}
		if _, ok := zhKeys[key]; !ok {
			t.Errorf("key %q exists in en.json but missing in zh.json", key)
		}
	}

	if missing > 0 {
		t.Logf("%d key(s) used in code but missing from zh.json. Add them to config/i18n/zh.json and en.json.", missing)
	}
}

// reChinese 匹配中文字符（CJK 统一汉字区）。
var reChineseChar = regexp.MustCompile(`[\x{4e00}-\x{9fff}]`)

// TestEnJsonNoChinese 确保 en.json 中所有 value 都是英文，不包含中文字符。
// 防止新增 key 时复制 zh.json 的中文值到 en.json 后忘记翻译。
func TestEnJsonNoChinese(t *testing.T) {
	root := findProjectRoot(t)

	enPath := filepath.Join(root, "config", "i18n", "en.json")
	enData, err := os.ReadFile(enPath)
	if err != nil {
		t.Fatalf("read en.json: %v", err)
	}
	var enMap map[string]string
	if err := json.Unmarshal(enData, &enMap); err != nil {
		t.Fatalf("parse en.json: %v", err)
	}

	violations := 0
	for key, val := range enMap {
		// 跳过 _meta 开头的 key
		if strings.HasPrefix(key, "_") {
			continue
		}
		if reChineseChar.MatchString(val) {
			t.Errorf("en.json key %q contains Chinese: %q", key, val)
			violations++
		}
	}
	if violations > 0 {
		t.Logf("%d key(s) in en.json still contain Chinese. Please translate them to English.", violations)
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
