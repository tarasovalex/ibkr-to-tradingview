package convert

import (
	"strconv"

	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// SimulateFIFOAvgCost estimates remaining average cost using FIFO lot consumption.
func SimulateFIFOAvgCost(txs []tv.Transaction, symbol string) (shares float64, avgCost float64, ok bool) {
	type lot struct {
		qty   float64
		price float64
	}
	var lots []lot
	for _, tx := range txs {
		if tx.Symbol != symbol {
			continue
		}
		qty, err := strconv.ParseFloat(tx.Qty, 64)
		if err != nil || qty <= 0 {
			continue
		}
		price, err := strconv.ParseFloat(tx.FillPrice, 64)
		if err != nil {
			continue
		}
		comm := 0.0
		if tx.Commission != "" {
			comm, _ = strconv.ParseFloat(tx.Commission, 64)
		}
		switch tx.Side {
		case "Buy":
			lots = append(lots, lot{qty: qty, price: price + comm/qty})
		case "Sell":
			rem := qty
			for rem > 0 && len(lots) > 0 {
				if lots[0].qty <= rem {
					rem -= lots[0].qty
					lots = lots[1:]
				} else {
					lots[0].qty -= rem
					rem = 0
				}
			}
		}
	}
	var totalCost float64
	for _, l := range lots {
		totalCost += l.qty * l.price
		shares += l.qty
	}
	if shares <= 0 {
		return 0, 0, false
	}
	return shares, totalCost / shares, true
}

// SimulateAvgCost computes weighted-average cost for a symbol from buy/sell transactions.
func SimulateAvgCost(txs []tv.Transaction, symbol string) (shares float64, avgCost float64, ok bool) {
	var cost float64
	for _, tx := range txs {
		if tx.Symbol != symbol {
			continue
		}
		qty, err := strconv.ParseFloat(tx.Qty, 64)
		if err != nil || qty <= 0 {
			continue
		}
		price, err := strconv.ParseFloat(tx.FillPrice, 64)
		if err != nil {
			continue
		}
		comm := 0.0
		if tx.Commission != "" {
			comm, _ = strconv.ParseFloat(tx.Commission, 64)
		}
		switch tx.Side {
		case "Buy":
			cost += qty*price + comm
			shares += qty
		case "Sell":
			if shares <= 0 {
				continue
			}
			avg := cost / shares
			cost -= qty * avg
			shares -= qty
		}
	}
	if shares <= 0 {
		return 0, 0, false
	}
	return shares, cost / shares, true
}
