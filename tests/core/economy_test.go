// economy_test.go — 经济系统单元测试。
package core_test

import (
	"testing"

	"defense2/internal/core/economy"
)

func TestSellRefund(t *testing.T) {
	cfg := economy.DefaultConfig()
	r := cfg.SellRefund(80)
	// SellRefundRatio = 0.7 → int(80 * 0.7) = 56
	if r != 56 {
		t.Fatalf("sell refund for 80 cost expected 56, got %d", r)
	}
}
