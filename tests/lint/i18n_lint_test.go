// i18n_lint_test.go — 强制文本国际化。
// 1. TestNoHardcodedChineseStrings: 禁止 Go 源码硬编码中文字符串。
// 2. TestNoRawStringInDrawCalls: 禁止文本渲染函数接收裸字符串字面量。
// 所有用户可见文本必须通过 i18n.T() 或 i18n.TF() 获取。
package lint_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reChinese 匹配引号内包含中文字符（\u4e00-\u9fff）的字符串字面量。
var reChinese = regexp.MustCompile(`"[^"]*[\x{4e00}-\x{9fff}][^"]*"`)

// i18nAllowedPaths 白名单路径：这些包允许包含中文字符串。
var i18nAllowedPaths = []string{
	filepath.Join("internal", "i18n"),           // i18n 包本身
	filepath.Join("internal", "config"),          // config loader（JSON 注释字段）
	filepath.Join("internal", "core", "mascot"),  // 萌妹系统从 JSON 加载
	filepath.Join("internal", "autoplay"),         // autoplay 测试报告
	filepath.Join("internal", "core", "aiplayer"), // AI 玩家本地文案模板（Phase 3 改为 JSON 加载）
	filepath.Join("internal", "scene", "map_editor.go"),   // 开发工具
	filepath.Join("internal", "scene", "audio_preview.go"),// 开发工具
	filepath.Join("internal", "scene", "vfx_preview.go"),  // 开发工具
	filepath.Join("internal", "scene", "wave_preview.go"), // 开发工具
	filepath.Join("internal", "scene", "test_select.go"),  // 测试模式选择器（开发工具）
	filepath.Join("internal", "scene", "lang_select.go"),  // 首次运行语言选择（故意双语）
	filepath.Join("cmd"),                         // CLI 入口
}

// i18nAllowedLinePatterns 行级白名单：匹配这些模式的行跳过检查。
var i18nAllowedLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`i18n\.T\(`),      // 已使用 i18n.T()
	regexp.MustCompile(`i18n\.TF\(`),     // 已使用 i18n.TF()
	regexp.MustCompile(`log\.Printf?\(`), // log 语句
	regexp.MustCompile(`fmt\.Errorf?\(`),  // 错误消息（内部）
	regexp.MustCompile(`fmt\.Printf?\(`), // debug 打印（内部）
	regexp.MustCompile(`fmt\.Sprintf\(`),            // 格式化（内部，debug用）
	regexp.MustCompile(`lines\s*=\s*append\(lines`), // debug info panel 行（drawEnemyInfoPanel）
	regexp.MustCompile(`speedInfo\s*\+=`),           // debug speed info 拼接
	regexp.MustCompile(`label\s*\+=`),               // debug label 拼接
	regexp.MustCompile(`//`),             // 行内注释
	regexp.MustCompile(`t\.Fatalf?\(`),   // 测试断言
	regexp.MustCompile(`t\.Errorf?\(`),   // 测试断言
	regexp.MustCompile(`t\.Skipf?\(`),    // 测试断言
	regexp.MustCompile(`panic\(`),        // panic 消息（内部）
}

func TestNoHardcodedChineseStrings(t *testing.T) {
	root := findProjectRoot(t)
	violations := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		// 跳过测试文件
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		// 跳过 vendor/.claude/worktrees
		if strings.Contains(rel, "vendor") || strings.Contains(rel, ".claude") {
			return nil
		}
		// 跳过白名单路径
		for _, allowed := range i18nAllowedPaths {
			if strings.Contains(rel, allowed) {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 跳过纯注释行
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			// 无中文字符 → 跳过
			if !reChinese.MatchString(line) {
				continue
			}

			// 检查行级白名单
			allowed := false
			for _, pat := range i18nAllowedLinePatterns {
				if pat.MatchString(line) {
					allowed = true
					break
				}
			}
			if allowed {
				continue
			}

			t.Errorf("%s:%d: hardcoded Chinese string (use i18n.T())\n  > %s", rel, lineNum, trimmed)
			violations++
		}
		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if violations > 0 {
		t.Logf("\n%d hardcoded Chinese string(s) found. Wrap with i18n.T(\"key\") and add entries to config/i18n/*.json.", violations)
	}
}

// ── TestNoRawStringInDrawCalls ─────────────────────────────────

// reDrawCall 匹配文本渲染函数调用（FontManager.Draw*Text、ShowToast、SpawnText、ui.Button 等）。
var reDrawCall = regexp.MustCompile(
	`\.(Draw(?:Centered)?(?:V)?(?:Bold)?(?:Right)?Text|DrawWrappedText)\(` +
		`|ShowToast\(` +
		`|SpawnText\(` +
		`|ui\.Button(?:WithState)?\(` +
		`|ui\.Badge\(` +
		`|ui\.IconCard\(`,
)

// reRawStringLiteral 匹配引号内的字符串字面量（非空）。
var reRawStringLiteral = regexp.MustCompile(`"([^"]+)"`)

// reI18nWrapped 匹配 i18n.T("...")/i18n.TF("...", ...) 中的 key 字符串。
// 只剥离 i18n 调用中的引号 key 部分，保留其他裸字符串供后续检查。
var reI18nWrapped = regexp.MustCompile(`i18n\.TF?\("[^"]*"`)

// reExemptLiteral 匹配无需国际化的字符串：纯格式占位符、纯符号/装饰、单位后缀、数字格式。
var reExemptLiteral = regexp.MustCompile(
	`^(%[.\-\d]*[dfsegvx]|` + // fmt 格式占位符
		`[^\p{L}]{0,4}|` + // ≤4 字符且不含字母（符号/装饰）
		`.{1}|` + // 单字符全部豁免（单位符号如 G/s/x）
		`%.+[dfseg])$`, // 数字格式串如 "%.0f", "%d/%d"
)

// TestNoRawStringInDrawCalls 确保文本渲染函数不接收裸字符串字面量。
// 所有展示给用户的文本必须来自 i18n.T()/TF() 或已解析的变量，不允许硬编码。
func TestNoRawStringInDrawCalls(t *testing.T) {
	root := findProjectRoot(t)
	violations := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		// 跳过测试文件
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		// 跳过 vendor/.claude
		if strings.Contains(rel, "vendor") || strings.Contains(rel, ".claude") {
			return nil
		}
		// 跳过白名单路径（dev tools 等）
		for _, allowed := range i18nAllowedPaths {
			if strings.Contains(rel, allowed) {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 跳过注释行
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			// 必须是文本渲染函数调用
			if !reDrawCall.MatchString(line) {
				continue
			}

			// 剥离 i18n.T()/TF() 包裹的部分，只检查剩余裸字符串
			stripped := reI18nWrapped.ReplaceAllString(line, "")

			// 查找剩余的字符串字面量
			matches := reRawStringLiteral.FindAllStringSubmatch(stripped, -1)
			for _, m := range matches {
				lit := m[1]
				// 豁免：纯符号/装饰/格式串
				if reExemptLiteral.MatchString(lit) {
					continue
				}
				t.Errorf("%s:%d: raw string %q in Draw/Text call (use i18n.T())\n  > %s", rel, lineNum, lit, trimmed)
				violations++
			}
		}
		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if violations > 0 {
		t.Logf("\n%d raw string literal(s) in text rendering calls. Use i18n.T(\"key\") instead.", violations)
	}
}
