// resolve_config.go — 配置加载后的显示文本解析。
// 将所有配置 struct 中的 Label/Name/Description 等字段通过 i18n 覆盖，
// 确保运行时所有显示文本统一走 i18n 管线。
package i18n

// ResolveKey 尝试用 i18n 覆盖配置值。
// 如果 key 在当前 locale 或 zh fallback 中存在，返回翻译值；
// 否则返回原始 fallbackVal（保持配置 JSON 中的值）。
func ResolveKey(key, fallbackVal string) string {
	if v, ok := active[key]; ok {
		return v
	}
	if v, ok := fallback[key]; ok {
		return v
	}
	return fallbackVal
}
