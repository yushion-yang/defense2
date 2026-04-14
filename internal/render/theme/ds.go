package theme

// ---------------------------------------------------------------------------
// Spacing
// ---------------------------------------------------------------------------

const (
	SpaceXS = 4
	SpaceSM = 8
	SpaceMD = 12
	SpaceLG = 16

	// Aliases (used by ui/ package)
	Gap4  = SpaceXS
	Gap8  = SpaceSM
	Gap12 = SpaceMD
	Gap16 = SpaceLG
)

// ---------------------------------------------------------------------------
// Font Sizes
// ---------------------------------------------------------------------------

const (
	FontXS = 11
	FontSM = 12
	FontMD = 14
	FontLG = 16
	FontXL = 18

	// Semantic aliases
	FontCaption = FontXS // 11
	FontBody    = FontSM // 12
	FontH2      = FontMD // 14
	FontH1      = FontXL // 18
)

// ---------------------------------------------------------------------------
// Special Font Sizes
// ---------------------------------------------------------------------------

const (
	FontTopBar       = 17
	FontMapLabel     = 16
	FontTowerName    = 10
	FontGameOver     = 52
	FontResultTitle  = 42
	FontWardenTitle  = 24
	FontSubtitle     = 13
	FontPauseTitle   = 24 // pause_menu 标题
	FontPauseBtn     = 18 // pause_menu 按钮
	FontAnnounce     = 28 // wave_announce 波次公告
	FontAnnounceLG   = 32 // wave_announce 大号公告
	FontOverlayTitle = 22 // warden_select_overlay 标题
	FontOverlayName  = 20 // warden_select_overlay 角色名
	FontDetailTitle  = 18 // warden_select_overlay 详情标题
	FontToggleIcon   = 16 // toggle_btn 图标文字
	FontDebugClose   = 11 // debug_panel 关闭按钮
)

// ---------------------------------------------------------------------------
// Line Heights
// ---------------------------------------------------------------------------

const (
	LineAbility = 16
	LineAttr    = 18
	LineTitle   = 22
	LineBtn     = 36
)

// ---------------------------------------------------------------------------
// Panel Tokens
// ---------------------------------------------------------------------------

const (
	PanelRadius   = 14
	PanelMinH     = 100
	PanelInnerPad = 14
)

// ---------------------------------------------------------------------------
// Button Tokens
// ---------------------------------------------------------------------------

const (
	ButtonH      = 30
	ButtonRadius = 12
	ButtonGap    = 8
)

// ---------------------------------------------------------------------------
// Detail Panel Tokens
// ---------------------------------------------------------------------------

const (
	DetailPad    = 14
	DetailTopPad = 14
	DetailBotPad = 14
	DetailGap    = 10
	DetailTitleH = 24
	DetailAttrH  = 22
	DetailRowH   = 20
	DetailBtnH   = 38
)
