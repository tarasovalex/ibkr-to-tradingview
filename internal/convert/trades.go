package convert

import (
	"strings"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// trade columns (Stocks section): DataDiscriminator, Asset Category, Currency, Symbol, Date/Time, Quantity, T. Price, ...
const (
	tradeIdxDiscriminator = 0
	tradeIdxAssetCategory = 1
	tradeIdxSymbol        = 3
	tradeIdxDateTime      = 4
	tradeIdxQuantity      = 5
	tradeIdxTradePrice    = 6
	tradeIdxCommFee       = 9
	tradeIdxBasis         = 10
)

func convertTrade(row ibkr.Row, mapper *symbols.Mapper) (*tv.Transaction, error) {
	if row.RowType != ibkr.RowData || ibkr.Field(row.Fields, tradeIdxDiscriminator) != "Order" {
		return nil, nil
	}
	if strings.ToLower(ibkr.Field(row.Fields, tradeIdxAssetCategory)) != "stocks" {
		return nil, nil
	}
	symbol := ibkr.Field(row.Fields, tradeIdxSymbol)
	if symbol == "" {
		return nil, nil
	}
	qty, err := ParseNumber(ibkr.Field(row.Fields, tradeIdxQuantity))
	if err != nil {
		return nil, nil
	}
	tradePrice, err := ParseNumber(ibkr.Field(row.Fields, tradeIdxTradePrice))
	if err != nil {
		return nil, nil
	}
	comm, _ := ParseNumber(ibkr.Field(row.Fields, tradeIdxCommFee))
	basis, _ := ParseNumber(ibkr.Field(row.Fields, tradeIdxBasis))

	side := "Buy"
	if qty < 0 {
		side = "Sell"
	}
	qtyAbs := Abs(qty)

	fillPrice := tradePrice
	commission := CommissionString(comm)
	// Buys: Basis is total cost (price + commission). Use it so TV avg cost matches IBKR.
	if side == "Buy" {
		if p, ok := fillPriceFromBasis(basis, qtyAbs); ok {
			fillPrice = p
			commission = ""
		}
	}

	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, tradeIdxDateTime))
	if err != nil {
		return nil, err
	}

	return &tv.Transaction{
		Symbol:      mapper.Format(symbol),
		Side:        side,
		Qty:         FormatQty(qtyAbs),
		FillPrice:   FormatPrice(fillPrice),
		Commission:  commission,
		ClosingTime: t,
	}, nil
}
