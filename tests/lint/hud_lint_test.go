// hud_lint_test.go — HUD 组件规范检测。
// 确保 hud/ 包只通过 ui/ 组件渲染，禁止直接调用底层绘图/文本 API。
// 违规代码可在行尾添加 //nolint:hud 豁免。
package lint_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// hudForbiddenPattern 定义一条 HUD 规范规则。
type hudForbiddenPattern struct {
	re      *regexp.Regexp
	message string
}

var hudForbidden = []hudForbiddenPattern{
	// 规则 1: 禁止 import core/game（应使用 theme.CanvasW/H）
	{regexp.MustCompile(`"defense2/internal/core/game"`), "hud/ must not import core/game; use theme.CanvasW/H instead"},

	// 规则 2: 禁止直接调用 fm.Draw*Text（应使用 ui.Label / ui.Paragraph）
	{regexp.MustCompile(`fm\.Draw(Bold|Centered|CenteredV|CenteredVBold|CenteredBold|Right)?Text\(`), "hud/ must not call fm.Draw*Text directly; use ui.Label/ui.Paragraph"},

	// 规则 3: 禁止直接调用底层绘图 API（应使用 ui.Panel/ui.Overlay 等组件）
	{regexp.MustCompile(`draw\.(RoundRect|FilledRect|StrokeRoundRect|FilledCircle)\(`), "hud/ must not call draw.RoundRect/FilledRect/etc directly; use ui.Panel/ui.Overlay"},

	// 规则 4: 禁止 1200/540 字面量（应使用 theme.CanvasW/H）
	{regexp.MustCompile(`\b1200\b`), "hud/ must not use literal 1200; use theme.CanvasW"},
	{regexp.MustCompile(`\b540\b`), "hud/ must not use literal 540; use theme.CanvasH"},
}

// hudExemptFiles 尚未迁移的 hud 文件豁免清单。
// 每迁移一个文件就从此清单移除，最终清空。
var hudExemptFiles = map[string]bool{}

func TestHUDComponentCompliance(t *testing.T) {
	root := findProjectRoot(t)
	hudDir := filepath.Join(root, "internal", "render", "hud")
	violations := 0

	err := filepath.Walk(hudDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		basename := filepath.Base(path)

		// 跳过豁免文件
		if hudExemptFiles[basename] {
			return nil
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

			// 跳过 //nolint:hud 豁免
			if strings.Contains(line, "//nolint:hud") {
				continue
			}

			for _, fp := range hudForbidden {
				if fp.re.MatchString(line) {
					t.Errorf("%s:%d: %s\n  > %s", basename, lineNum, fp.message, strings.TrimSpace(line))
					violations++
				}
			}
		}
		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if violations > 0 {
		t.Fatalf("\n%d HUD compliance violation(s) found. Use ui.* components or add //nolint:hud exemption.", violations)
	}

	// 提醒清理豁免清单
	if len(hudExemptFiles) > 0 {
		t.Logf("NOTE: %d hud files still exempt from lint. Remove from hudExemptFiles after migration.", len(hudExemptFiles))
	}
}
