package convert

import (
	"strings"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// dividends: Currency, Date, Description, Amount
const (
	divIdxDate        = 1
	divIdxDescription = 2
	divIdxAmount      = 3
)

func convertDividend(row ibkr.Row, mapper *symbols.Mapper) (*tv.Transaction, error) {
	if ibkr.Field(row.Fields, 0) == "Total" {
		return nil, nil
	}
	desc := ibkr.Field(row.Fields, divIdxDescription)
	if strings.HasPrefix(desc, "Total") || desc == "" {
		return nil, nil
	}
	ticker := ExtractDividendTicker(desc)
	if ticker == "" {
		return nil, nil
	}
	amount, err := ParseNumber(ibkr.Field(row.Fields, divIdxAmount))
	if err != nil {
		return nil, nil
	}
	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, divIdxDate))
	if err != nil {
		return nil, err
	}
	return &tv.Transaction{
		Symbol:      mapper.Format(ticker),
		Side:        "Dividend",
		Qty:         FormatMoney(amount),
		FillPrice:   "",
		Commission:  "",
		ClosingTime: t,
	}, nil
}
