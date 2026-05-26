package convert

import (
	"strings"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
)

// open position summary: DataDiscriminator, Asset Category, Currency, Symbol, Quantity, Mult, Cost Price, Cost Basis, ...
const (
	openIdxDiscriminator = 0
	openIdxAssetCategory   = 1
	openIdxSymbol          = 3
	openIdxCostBasis       = 7
)

// scanTransferCostPerShare derives acquisition cost per share for inbound transfers.
// IBKR lists market value on transfer rows but carries tax cost in Open Positions.
// transferBasis = endingCostBasis - netBasisFromTradesInPeriod (buys add, sells subtract).
func scanTransferCostPerShare(rows []ibkr.Row, warn func(string)) map[string]float64 {
	openCost := make(map[string]float64)
	tradeBasisNet := make(map[string]float64)
	xferQty := make(map[string]float64)

	for _, row := range rows {
		if !ibkr.IsDataRow(row.RowType) {
			continue
		}
		switch row.Section {
		case ibkr.SectionOpenPositions:
			if ibkr.Field(row.Fields, openIdxDiscriminator) != "Summary" {
				continue
			}
			if strings.ToLower(ibkr.Field(row.Fields, openIdxAssetCategory)) != "stocks" {
				continue
			}
			symbol := ibkr.Field(row.Fields, openIdxSymbol)
			if symbol == "" {
				continue
			}
			cost, err := ParseNumber(ibkr.Field(row.Fields, openIdxCostBasis))
			if err != nil {
				continue
			}
			openCost[symbol] = cost

		case ibkr.SectionTrades:
			if ibkr.Field(row.Fields, tradeIdxDiscriminator) != "Order" {
				continue
			}
			if strings.ToLower(ibkr.Field(row.Fields, tradeIdxAssetCategory)) != "stocks" {
				continue
			}
			symbol := ibkr.Field(row.Fields, tradeIdxSymbol)
			if symbol == "" {
				continue
			}
			basis, err := ParseNumber(ibkr.Field(row.Fields, tradeIdxBasis))
			if err != nil {
				continue
			}
			tradeBasisNet[symbol] += basis

		case ibkr.SectionTransfers:
			if strings.ToLower(ibkr.Field(row.Fields, xferIdxAssetCategory)) != "stocks" {
				continue
			}
			if ibkr.Field(row.Fields, xferIdxDirection) != "In" {
				continue
			}
			symbol := ibkr.Field(row.Fields, xferIdxSymbol)
			qty, err := ParseNumber(ibkr.Field(row.Fields, xferIdxQty))
			if err != nil || qty <= 0 {
				continue
			}
			xferQty[symbol] += qty
		}
	}

	out := make(map[string]float64)
	for symbol, qty := range xferQty {
		endCost, ok := openCost[symbol]
		if !ok {
			if warn != nil {
				warn("transfer " + symbol + ": no Open Positions cost basis; using market value")
			}
			continue
		}
		transferBasis := endCost - tradeBasisNet[symbol]
		if transferBasis <= 0 || qty <= 0 {
			if warn != nil {
				warn("transfer " + symbol + ": could not derive cost basis; using market value")
			}
			continue
		}
		out[symbol] = transferBasis / qty
	}
	return out
}

// fillPriceFromBasis returns cost per share from IBKR Basis (includes commissions on buys).
func fillPriceFromBasis(basis, qty float64) (float64, bool) {
	if qty == 0 || basis == 0 {
		return 0, false
	}
	return Abs(basis) / Abs(qty), true
}
