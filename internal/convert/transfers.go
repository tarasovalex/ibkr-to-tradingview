package convert

import (
	"strings"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// transfers: Asset Category, Currency, Symbol, Date, Type, Direction, ..., Qty, Xfer Price, Market Value
const (
	xferIdxAssetCategory = 0
	xferIdxSymbol        = 2
	xferIdxDate          = 3
	xferIdxDirection     = 5
	xferIdxQty           = 8
	xferIdxXferPrice     = 9
	xferIdxMarketValue   = 10
)

// convertTransfer records transfer-in as a buy plus a matching $CASH deposit (same cost).
// Export sorts all same-day deposits before buys so TradingView does not partial-fill purchases.
func convertTransfer(row ibkr.Row, mapper *symbols.Mapper, transferCost map[string]float64) ([]tv.Transaction, error) {
	if strings.ToLower(ibkr.Field(row.Fields, xferIdxAssetCategory)) != "stocks" {
		return nil, nil
	}
	if ibkr.Field(row.Fields, xferIdxDirection) != "In" {
		return nil, nil
	}
	symbol := ibkr.Field(row.Fields, xferIdxSymbol)
	if symbol == "" {
		return nil, nil
	}
	qty, err := ParseNumber(ibkr.Field(row.Fields, xferIdxQty))
	if err != nil || qty <= 0 {
		return nil, nil
	}
	var price float64
	if xferPrice, err := ParseNumber(ibkr.Field(row.Fields, xferIdxXferPrice)); err == nil && xferPrice > 0 {
		price = xferPrice
	} else if transferCost != nil {
		if costPerShare, ok := transferCost[symbol]; ok && costPerShare > 0 {
			price = costPerShare
		} else {
			mv, err := ParseMarketValue(ibkr.Field(row.Fields, xferIdxMarketValue))
			if err != nil {
				return nil, nil
			}
			price = mv / qty
		}
	} else {
		mv, err := ParseMarketValue(ibkr.Field(row.Fields, xferIdxMarketValue))
		if err != nil {
			return nil, nil
		}
		price = mv / qty
	}

	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, xferIdxDate))
	if err != nil {
		return nil, err
	}

	totalCost := qty * price
	buy := tv.Transaction{
		Symbol:      mapper.Format(symbol),
		Side:        "Buy",
		Qty:         FormatQty(qty),
		FillPrice:   FormatPrice(price),
		Commission:  "",
		ClosingTime: t,
	}
	deposit := cashDeposit(totalCost, t)
	return []tv.Transaction{*deposit, buy}, nil
}
