package mascot

// Line is one speech bubble message within a dialog.
// Text 为多语言映射：{"zh": "中文", "en": "English"}。
// 运行时通过 Guide.ResolveText 按当前 locale 选择，缺失时 fallback 到 "zh"。
type Line struct {
	Text        map[string]string `json:"text"`
	Expression  string            `json:"expression"`  // "idle"/"talk"/"happy"/"surprised"
	AutoAdvance float64           `json:"autoAdvance"` // seconds; 0 = click to advance
}

// Dialog is a triggered conversation sequence.
type Dialog struct {
	ID       string `json:"id"`
	Scene    string `json:"scene"`    // "title"/"stage"/"select"/"*"
	Trigger  string `json:"trigger"`  // event name; "scene_enter" = on scene switch
	Lines    []Line `json:"lines"`
	Once     bool   `json:"once"`     // show only once ever (persisted)
	Priority int    `json:"priority"` // higher wins when multiple match
}
