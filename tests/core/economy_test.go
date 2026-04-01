// economy_test.go — 经济系统单元测试。
package core_test

import (
	"testing"

	"defense2/internal/core/economy"
)

func TestInterestGold(t *testing.T) {
	cfg := economy.DefaultConfig()
	// 200 * 0.05 = 10
	g := cfg.InterestGold(200)
	if g != 10 {
		t.Fatalf("interest on 200 expected 10, got %d", g)
	}
	// 2000 * 0.05 = 100 → 上限 50
	g = cfg.InterestGold(2000)
	if g != 50 {
		t.Fatalf("interest capped at 50, got %d", g)
	}
}

func TestSellRefund(t *testing.T) {
	cfg := economy.DefaultConfig()
	r := cfg.SellRefund(80)
	if r != 40 {
		t.Fatalf("sell refund for 80 cost expected 40, got %d", r)
	}
}
