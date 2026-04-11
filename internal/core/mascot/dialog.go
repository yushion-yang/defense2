package mascot

// Line is one speech bubble message within a dialog.
type Line struct {
	Text        string  `json:"text"`
	Expression  string  `json:"expression"`  // "idle"/"talk"/"happy"/"surprised"
	AutoAdvance float64 `json:"autoAdvance"` // seconds; 0 = click to advance
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
