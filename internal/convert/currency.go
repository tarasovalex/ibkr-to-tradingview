package convert

import (
	"strings"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
)

// currencyContext holds ILS→USD conversion ratios from statement summary rows.
type currencyContext struct {
	ilsDepositTotal float64
	usdDepositTotal float64
	hasDepositRatio bool
}

func scanCurrencyContext(rows []ibkr.Row) currencyContext {
	var ctx currencyContext
	for _, row := range rows {
		if row.Section != ibkr.SectionDepositsWithdrawals || row.RowType != ibkr.RowData {
			continue
		}
		currencyCol := ibkr.Field(row.Fields, depIdxCurrency)
		switch currencyCol {
		case "Total in USD", "Total Deposits & Withdrawals in USD":
			if v, err := ParseNumber(ibkr.Field(row.Fields, depIdxAmount)); err == nil {
				ctx.usdDepositTotal = Abs(v)
				ctx.hasDepositRatio = ctx.ilsDepositTotal > 0 && ctx.usdDepositTotal > 0
			}
		case "ILS":
			desc := ibkr.Field(row.Fields, depIdxDescription)
			if desc == "" || strings.HasPrefix(desc, "Total") {
				continue
			}
			if v, err := ParseNumber(ibkr.Field(row.Fields, depIdxAmount)); err == nil {
				ctx.ilsDepositTotal += Abs(v)
			}
		}
	}
	if ctx.ilsDepositTotal > 0 && ctx.usdDepositTotal > 0 {
		ctx.hasDepositRatio = true
	}
	return ctx
}

func (ctx currencyContext) ilsToUSD(ilsAmount float64, warn func(string)) float64 {
	if ctx.hasDepositRatio {
		return ilsAmount * (ctx.usdDepositTotal / ctx.ilsDepositTotal)
	}
	if warn != nil {
		warn("ILS amount without deposit USD ratio; using raw ILS value")
	}
	return ilsAmount
}
