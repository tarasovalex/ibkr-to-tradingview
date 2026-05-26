package convert

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

type sellBasisInfo struct {
	symbol string
	qty    float64
	cost   float64 // positive cost removed (abs of IBKR Basis)
	time   time.Time
}

func scanSellBasis(rows []ibkr.Row) []sellBasisInfo {
	var sells []sellBasisInfo
	for _, row := range rows {
		if row.Section != ibkr.SectionTrades || row.RowType != ibkr.RowData {
			continue
		}
		if ibkr.Field(row.Fields, tradeIdxDiscriminator) != "Order" {
			continue
		}
		if strings.ToLower(ibkr.Field(row.Fields, tradeIdxAssetCategory)) != "stocks" {
			continue
		}
		qty, err := ParseNumber(ibkr.Field(row.Fields, tradeIdxQuantity))
		if err != nil || qty >= 0 {
			continue
		}
		basis, err := ParseNumber(ibkr.Field(row.Fields, tradeIdxBasis))
		if err != nil {
			continue
		}
		cost := Abs(basis)
		if cost < 1e-9 {
			continue
		}
		t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, tradeIdxDateTime))
		if err != nil {
			continue
		}
		sells = append(sells, sellBasisInfo{
			symbol: ibkr.Field(row.Fields, tradeIdxSymbol),
			qty:    Abs(qty),
			cost:   cost,
			time:   t,
		})
	}
	sort.Slice(sells, func(i, j int) bool {
		return sells[i].time.Before(sells[j].time)
	})
	return sells
}

// AdjustTransferLots splits inbound transfer buys so FIFO-style portfolio tools
// consume the same cost basis IBKR reports on sells.
func AdjustTransferLots(rows []ibkr.Row, txs []tv.Transaction) []tv.Transaction {
	sells := scanSellBasis(rows)
	if len(sells) == 0 {
		return txs
	}

	out := make([]tv.Transaction, len(txs))
	copy(out, txs)

	for _, s := range sells {
		tvSymbol := tvSymbolForTicker(out, s.symbol)
		if tvSymbol == "" {
			continue
		}
		idx := findSplittableTransferBuy(out, tvSymbol, s.qty)
		if idx < 0 {
			continue
		}
			out, _ = splitTransferBuy(out, idx, s.qty, s.cost)
	}
	return out
}

// findTransferDeposit locates the $CASH deposit paired with a transfer buy (same day, same total cost).
func findTransferDeposit(txs []tv.Transaction, day time.Time, totalCost float64) int {
	for i, tx := range txs {
		if tx.Symbol != cashSymbol || tx.Side != "Deposit" {
			continue
		}
		if !CalendarDay(tx.ClosingTime).Equal(day) {
			continue
		}
		amt, err := ParseNumber(tx.Qty)
		if err != nil {
			continue
		}
		if Abs(amt-totalCost) < 0.02 {
			return i
		}
	}
	return -1
}

func tvSymbolForTicker(txs []tv.Transaction, ticker string) string {
	suffix := ":" + ticker
	for _, tx := range txs {
		if tx.Side == "Buy" && strings.HasSuffix(tx.Symbol, suffix) {
			return tx.Symbol
		}
	}
	return ""
}

func findSplittableTransferBuy(txs []tv.Transaction, tvSymbol string, sellQty float64) int {
	for i, tx := range txs {
		if tx.Symbol != tvSymbol || tx.Side != "Buy" || tx.Commission != "" {
			continue
		}
		qty, err := strconv.ParseFloat(tx.Qty, 64)
		if err != nil || qty < sellQty {
			continue
		}
		// Inbound transfers are whole-share lots without commission.
		if qty >= 1 && qty == float64(int64(qty)) {
			return i
		}
	}
	return -1
}

func splitTransferBuy(txs []tv.Transaction, idx int, sellQty, sellCost float64) ([]tv.Transaction, bool) {
	orig := txs[idx]
	totalQty, err := strconv.ParseFloat(orig.Qty, 64)
	if err != nil || totalQty <= sellQty {
		return txs, false
	}
	totalPrice, err := strconv.ParseFloat(orig.FillPrice, 64)
	if err != nil {
		return txs, false
	}
	totalCost := totalQty * totalPrice
	if sellCost > totalCost {
		return txs, false
	}
	restQty := totalQty - sellQty
	restCost := totalCost - sellCost

	day := CalendarDay(orig.ClosingTime)
	depIdx := findTransferDeposit(txs, day, totalCost)

	first := orig
	first.Qty = FormatQty(sellQty)
	first.FillPrice = FormatPrice(sellCost / sellQty)

	second := orig
	second.Qty = FormatQty(restQty)
	second.FillPrice = FormatPrice(restCost / restQty)

	out := make([]tv.Transaction, 0, len(txs)+1)
	out = append(out, txs[:idx]...)
	out = append(out, first, second)
	out = append(out, txs[idx+1:]...)

	if depIdx >= 0 {
		// Adjust deposit index if it was after the replaced buy.
		if depIdx > idx {
			depIdx++
		}
		out[depIdx].Qty = FormatMoney(sellCost)
		out = append(out, *cashDeposit(restCost, orig.ClosingTime))
	}
	return out, depIdx >= 0
}
