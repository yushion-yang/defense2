// i18n_lint_test.go — 禁止在 Go 源码中硬编码中文字符串。
// 所有用户可见文本必须通过 i18n.T() 或 i18n.TF() 获取。
// 白名单：i18n 包本身、测试文件、config loader（JSON tag）、注释、log 语句。
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
	filepath.Join("internal", "scene", "map_editor.go"),   // 开发工具
	filepath.Join("internal", "scene", "audio_preview.go"),// 开发工具
	filepath.Join("internal", "scene", "vfx_preview.go"),  // 开发工具
	filepath.Join("internal", "scene", "wave_preview.go"), // 开发工具
	filepath.Join("cmd"),                         // CLI 入口
}

// i18nAllowedLinePatterns 行级白名单：匹配这些模式的行跳过检查。
var i18nAllowedLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`i18n\.T\(`),      // 已使用 i18n.T()
	regexp.MustCompile(`i18n\.TF\(`),     // 已使用 i18n.TF()
	regexp.MustCompile(`log\.Printf?\(`), // log 语句
	regexp.MustCompile(`fmt\.Errorf?\(`), // 错误消息（内部）
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
