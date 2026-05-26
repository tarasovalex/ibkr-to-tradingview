package tv

import (
	"fmt"
	"time"
)

// Transaction is a single TradingView portfolio import row.
type Transaction struct {
	Symbol      string
	Side        string
	Qty         string
	FillPrice   string
	Commission  string
	ClosingTime time.Time
}

// DedupeKey returns a string key for exact-match deduplication.
func (t Transaction) DedupeKey() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		t.Symbol, t.Side, t.Qty, t.FillPrice, t.Commission,
		t.ClosingTime.Format("2006-01-02 15:04:05"))
}

// ClosingTimeString formats time like TradingView example: "2024-09-17 0:00:00".
func (t Transaction) ClosingTimeString() string {
	y, m, d := t.ClosingTime.Date()
	h, min, s := t.ClosingTime.Clock()
	return fmt.Sprintf("%04d-%02d-%02d %d:%02d:%02d", y, int(m), d, h, min, s)
}
