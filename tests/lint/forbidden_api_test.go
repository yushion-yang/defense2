// forbidden_api_test.go — 禁止直接调用 Ebitengine 底层 API。
// 确保所有渲染和输入调用走 draw.* 包装器，保证 HiDPI 自动缩放。
// 详见 docs/rendering-hidpi.md。
package lint_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// forbiddenPattern 定义一条禁止规则。
type forbiddenPattern struct {
	re      *regexp.Regexp
	message string
}

var forbidden = []forbiddenPattern{
	// 鼠标/触摸 — 必须用 draw.CursorPos() / draw.TouchPos()
	{regexp.MustCompile(`ebiten\.CursorPosition\(`), "use draw.CursorPos() instead of ebiten.CursorPosition()"},
	{regexp.MustCompile(`ebiten\.TouchPosition\(`), "use draw.TouchPos() instead of ebiten.TouchPosition()"},

	// 绘图 — 必须用 draw.Line / draw.FilledRect / draw.FilledCircle
	{regexp.MustCompile(`vector\.StrokeLine\(`), "use draw.Line() instead of vector.StrokeLine()"},
	{regexp.MustCompile(`vector\.DrawFilledRect\(`), "use draw.FilledRect() instead of vector.DrawFilledRect()"},
	{regexp.MustCompile(`vector\.DrawFilledCircle\(`), "use draw.FilledCircle() instead of vector.DrawFilledCircle()"},
}

// allowedPaths 白名单：draw 包本身是封装层，允许调用底层 API。
var allowedPaths = []string{
	filepath.Join("internal", "render", "draw"),
}

func TestForbiddenAPICalls(t *testing.T) {
	root := findProjectRoot(t)
	violations := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只检查 .go 文件
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 跳过测试文件、vendor、node_modules
		rel, _ := filepath.Rel(root, path)
		if strings.Contains(rel, "vendor") || strings.Contains(rel, "node_modules") {
			return nil
		}
		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}

		// 跳过白名单路径
		for _, allowed := range allowedPaths {
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

			for _, fp := range forbidden {
				if fp.re.MatchString(line) {
					t.Errorf("%s:%d: %s\n  > %s", rel, lineNum, fp.message, strings.TrimSpace(line))
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
		t.Fatalf("\n%d forbidden API call(s) found. See docs/rendering-hidpi.md for the correct wrappers.", violations)
	}
}

// findProjectRoot 向上查找包含 go.mod 的目录。
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find project root (go.mod)")
		}
		dir = parent
	}
}
