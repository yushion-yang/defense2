package i18n

import (
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"config/i18n/zh.json": &fstest.MapFile{
			Data: []byte(`{
				"settings.title": "设    置",
				"settings.sfx_volume": "音效音量",
				"settings.back": "返回",
				"format.wave": "第 %d 波"
			}`),
		},
		"config/i18n/en.json": &fstest.MapFile{
			Data: []byte(`{
				"settings.title": "Settings",
				"settings.sfx_volume": "SFX Volume",
				"settings.back": "Back",
				"format.wave": "Wave %d"
			}`),
		},
	}
}

func TestT_ExistingKey(t *testing.T) {
	if err := Init(testFS(), "zh"); err != nil {
		t.Fatal(err)
	}
	got := T("settings.title")
	if got != "设    置" {
		t.Errorf("T('settings.title') = %q, want %q", got, "设    置")
	}
}

func TestT_EnglishLocale(t *testing.T) {
	if err := Init(testFS(), "en"); err != nil {
		t.Fatal(err)
	}
	got := T("settings.title")
	if got != "Settings" {
		t.Errorf("T('settings.title') = %q, want %q", got, "Settings")
	}
}

func TestT_FallbackToZh(t *testing.T) {
	// Init with a locale that doesn't exist — falls back to zh
	if err := Init(testFS(), "ja"); err != nil {
		t.Fatal(err)
	}
	got := T("settings.back")
	if got != "返回" {
		t.Errorf("T('settings.back') with ja locale = %q, want %q", got, "返回")
	}
}

func TestT_FallbackToKey(t *testing.T) {
	if err := Init(testFS(), "en"); err != nil {
		t.Fatal(err)
	}
	got := T("nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T('nonexistent.key') = %q, want %q", got, "nonexistent.key")
	}
}

func TestTF_Format(t *testing.T) {
	if err := Init(testFS(), "en"); err != nil {
		t.Fatal(err)
	}
	got := TF("format.wave", 5)
	if got != "Wave 5" {
		t.Errorf("TF('format.wave', 5) = %q, want %q", got, "Wave 5")
	}
}

func TestTF_FormatZh(t *testing.T) {
	if err := Init(testFS(), "zh"); err != nil {
		t.Fatal(err)
	}
	got := TF("format.wave", 3)
	if got != "第 3 波" {
		t.Errorf("TF('format.wave', 3) = %q, want %q", got, "第 3 波")
	}
}

func TestSetLocale_Switches(t *testing.T) {
	if err := Init(testFS(), "zh"); err != nil {
		t.Fatal(err)
	}
	if T("settings.back") != "返回" {
		t.Fatal("expected zh")
	}
	SetLocale("en")
	if T("settings.back") != "Back" {
		t.Errorf("after SetLocale('en'), T('settings.back') = %q", T("settings.back"))
	}
	if Locale() != "en" {
		t.Errorf("Locale() = %q, want %q", Locale(), "en")
	}
}

func TestAvailable(t *testing.T) {
	if err := Init(testFS(), "zh"); err != nil {
		t.Fatal(err)
	}
	avail := Available()
	if len(avail) != 2 {
		t.Fatalf("Available() = %v, want 2 locales", avail)
	}
	if avail[0] != "en" || avail[1] != "zh" {
		t.Errorf("Available() = %v, want [en zh]", avail)
	}
}

func TestInit_EmptyLocaleDefaultsToZh(t *testing.T) {
	if err := Init(testFS(), ""); err != nil {
		t.Fatal(err)
	}
	if Locale() != "zh" {
		t.Errorf("Locale() = %q, want %q", Locale(), "zh")
	}
}
