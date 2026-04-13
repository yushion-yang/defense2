// i18n 提供轻量级国际化支持。
// 从 config/i18n/*.json 加载扁平 key-value 对，通过 T(key) 查询翻译。
// Fallback 链: 当前语言 → zh → key 本身。
package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// locales 存储所有已加载语言的翻译表。
var locales map[string]map[string]string

// active 是当前语言的翻译表（只读，无需加锁）。
var active map[string]string

// fallback 是 zh 语言的翻译表（兜底）。
var fallback map[string]string

// currentLoc 当前语言代码。
var currentLoc string

// Init 从嵌入 FS 加载所有 config/i18n/*.json 并设置活跃语言。
// locale 为空时默认 "zh"。
func Init(dataFS fs.FS, locale string) error {
	if locale == "" {
		locale = "zh"
	}

	locales = make(map[string]map[string]string)

	entries, err := fs.ReadDir(dataFS, "config/i18n")
	if err != nil {
		return fmt.Errorf("i18n: read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		code := strings.TrimSuffix(entry.Name(), ".json")
		data, err := fs.ReadFile(dataFS, filepath.Join("config/i18n", entry.Name()))
		if err != nil {
			return fmt.Errorf("i18n: read %s: %w", entry.Name(), err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("i18n: parse %s: %w", entry.Name(), err)
		}
		table := make(map[string]string, len(raw))
		for k, v := range raw {
			if s, ok := v.(string); ok {
				table[k] = s
			}
		}
		locales[code] = table
	}

	// zh 永远作为 fallback
	if zh, ok := locales["zh"]; ok {
		fallback = zh
	} else {
		fallback = map[string]string{}
	}

	return SetLocale(locale)
}

// T 返回 key 对应的翻译文本。
// Fallback: active locale → zh → key 本身。
func T(key string) string {
	if v, ok := active[key]; ok {
		return v
	}
	if v, ok := fallback[key]; ok {
		return v
	}
	return key
}

// TF 返回格式化的翻译文本（等价于 fmt.Sprintf(T(key), args...)）。
func TF(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// Locale 返回当前语言代码。
func Locale() string {
	return currentLoc
}

// TFromLocale 从指定 locale 查询翻译（不改变当前语言）。
func TFromLocale(locale, key string) string {
	if table, ok := locales[locale]; ok {
		if v, found := table[key]; found {
			return v
		}
	}
	return T(key)
}

// SetLocale 切换活跃语言。locale 不存在时回退到 zh。
func SetLocale(locale string) error {
	if table, ok := locales[locale]; ok {
		active = table
		currentLoc = locale
		return nil
	}
	// 不存在则回退 zh
	if table, ok := locales["zh"]; ok {
		active = table
		currentLoc = "zh"
		return nil
	}
	// 无任何语言可用
	active = map[string]string{}
	currentLoc = locale
	return nil
}

// Available 返回所有已加载的语言代码（已排序）。
func Available() []string {
	codes := make([]string, 0, len(locales))
	for k := range locales {
		codes = append(codes, k)
	}
	sort.Strings(codes)
	return codes
}
