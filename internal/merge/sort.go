package merge

import (
	"sort"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

func calendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// importOrder controls same-calendar-day ordering for TradingView cash/position logic.
// Deposits must settle before stock buys on a given day.
func importOrder(t tv.Transaction) int {
	if t.Symbol == "$CASH" {
		switch t.Side {
		case "Deposit":
			return 0
		case "Taxes and fees":
			return 40
		case "Withdrawal":
			return 50
		}
	}
	switch t.Side {
	case "Buy":
		return 10
	case "Sell":
		return 20
	case "Dividend":
		return 25
	}
	return 30
}

// SortForImport orders transactions by time, and within each day runs deposits before buys before sells.
func SortForImport(txs []tv.Transaction) {
	sort.SliceStable(txs, func(i, j int) bool {
		ti, tj := txs[i].ClosingTime, txs[j].ClosingTime
		di, dj := calendarDay(ti), calendarDay(tj)
		if !di.Equal(dj) {
			return ti.Before(tj)
		}
		oi, oj := importOrder(txs[i]), importOrder(txs[j])
		if oi != oj {
			return oi < oj
		}
		return ti.Before(tj)
	})
}
