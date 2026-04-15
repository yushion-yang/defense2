// descriptor_trigger_test.go — Trigger 类型解析测试。
//
// 验证 ParseTrigger 能正确将字符串映射为 TriggerType 常量，
// 并对无效输入返回错误。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

func TestParseTriggerValid(t *testing.T) {
	cases := []struct {
		input string
		want  descriptor.TriggerType
	}{
		{"onHit", descriptor.TriggerOnHit},
		{"onTick", descriptor.TriggerOnTick},
		{"onKill", descriptor.TriggerOnKill},
		{"onPlace", descriptor.TriggerOnPlace},
	}
	for _, tc := range cases {
		got, err := descriptor.ParseTrigger(tc.input)
		if err != nil {
			t.Errorf("ParseTrigger(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseTrigger(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseTriggerInvalid(t *testing.T) {
	invalids := []string{"", "onDamage", "hit", "OnHit", "on_hit"}
	for _, s := range invalids {
		_, err := descriptor.ParseTrigger(s)
		if err == nil {
			t.Errorf("ParseTrigger(%q) expected error, got nil", s)
		}
	}
}

func TestTriggerTypeString(t *testing.T) {
	cases := []struct {
		trigger descriptor.TriggerType
		want    string
	}{
		{descriptor.TriggerOnHit, "onHit"},
		{descriptor.TriggerOnTick, "onTick"},
		{descriptor.TriggerOnKill, "onKill"},
		{descriptor.TriggerOnPlace, "onPlace"},
	}
	for _, tc := range cases {
		got := tc.trigger.String()
		if got != tc.want {
			t.Errorf("TriggerType(%d).String() = %q, want %q", tc.trigger, got, tc.want)
		}
	}
}

func TestTriggerTypeStringUnknown(t *testing.T) {
	unknown := descriptor.TriggerType(99)
	got := unknown.String()
	if got == "" {
		t.Error("unknown TriggerType.String() should not be empty")
	}
}
