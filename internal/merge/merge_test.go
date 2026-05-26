package merge_test

import (
	"testing"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/merge"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

func TestCombineDedupe(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tx := tv.Transaction{
		Symbol: "NASDAQ:AAPL", Side: "Buy", Qty: "1", FillPrice: "100",
		Commission: "", ClosingTime: t1,
	}
	out := merge.Combine([][]tv.Transaction{{tx}, {tx}}, true)
	if len(out) != 1 {
		t.Fatalf("dedupe: got %d want 1", len(out))
	}
	out2 := merge.Combine([][]tv.Transaction{{tx}, {tx}}, false)
	if len(out2) != 2 {
		t.Fatalf("no dedupe: got %d want 2", len(out2))
	}
}
