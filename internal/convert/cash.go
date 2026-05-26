package convert

import (
	"strings"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

const cashSymbol = "$CASH"

// deposit columns: Currency, Settle Date, Description, Amount
const (
	depIdxCurrency    = 0
	depIdxSettleDate  = 1
	depIdxDescription = 2
	depIdxAmount      = 3
)

func convertDeposit(row ibkr.Row, ctx currencyContext, warn func(string)) (*tv.Transaction, error) {
	currency := ibkr.Field(row.Fields, depIdxCurrency)
	if currency == "" || currency == "Total" || currency == "Total in USD" ||
		strings.HasPrefix(currency, "Total ") {
		return nil, nil
	}
	desc := ibkr.Field(row.Fields, depIdxDescription)
	if strings.HasPrefix(desc, "Total") {
		return nil, nil
	}
	amount, err := ParseNumber(ibkr.Field(row.Fields, depIdxAmount))
	if err != nil {
		return nil, nil
	}
	if amount == 0 {
		return nil, nil
	}

	amt := Abs(amount)
	if currency == "ILS" {
		amt = ctx.ilsToUSD(amt, warn)
	}

	side := "Deposit"
	if amount < 0 {
		side = "Withdrawal"
	}

	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, depIdxSettleDate))
	if err != nil {
		return nil, err
	}

	return &tv.Transaction{
		Symbol:      cashSymbol,
		Side:        side,
		Qty:         FormatMoney(amt),
		FillPrice:   "",
		Commission:  "",
		ClosingTime: t,
	}, nil
}

// fees: Subtitle, Currency, Date, Description, Amount
const (
	feeIdxCurrency = 1
	feeIdxDate     = 2
	feeIdxAmount   = 4
)

func convertFee(row ibkr.Row, ctx currencyContext, warn func(string)) (*tv.Transaction, error) {
	if ibkr.Field(row.Fields, 0) == "Total" || strings.Contains(ibkr.Field(row.Fields, 0), "Total") {
		return nil, nil
	}
	desc := ibkr.Field(row.Fields, 3)
	if strings.HasPrefix(desc, "Total") {
		return nil, nil
	}
	amount, err := ParseNumber(ibkr.Field(row.Fields, feeIdxAmount))
	if err != nil || amount == 0 {
		return nil, nil
	}
	amt := Abs(amount)
	if ibkr.Field(row.Fields, feeIdxCurrency) == "ILS" {
		amt = ctx.ilsToUSD(amt, warn)
	}
	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, feeIdxDate))
	if err != nil {
		return nil, err
	}
	return cashTaxesFees(amt, t), nil
}

// withholding: Currency, Date, Description, Amount
const (
	whIdxDate   = 1
	whIdxAmount = 3
)

func convertWithholding(row ibkr.Row) (*tv.Transaction, error) {
	if ibkr.Field(row.Fields, 0) == "Total" {
		return nil, nil
	}
	desc := ibkr.Field(row.Fields, 2)
	if strings.HasPrefix(desc, "Total") {
		return nil, nil
	}
	amount, err := ParseNumber(ibkr.Field(row.Fields, whIdxAmount))
	if err != nil || amount == 0 {
		return nil, nil
	}
	// Skip positive rows (reversals/duplicates); keep tax outflows only.
	if amount > 0 {
		return nil, nil
	}
	dateStr := ibkr.Field(row.Fields, whIdxDate)
	if dateStr == "" {
		return nil, nil
	}
	t, err := ParseIBKRDateTime(dateStr)
	if err != nil {
		return nil, err
	}
	return cashTaxesFees(Abs(amount), t), nil
}

// interest: Currency, Date, Description, Amount
const (
	intIdxDate   = 1
	intIdxDesc   = 2
	intIdxAmount = 3
)

func convertInterest(row ibkr.Row) (*tv.Transaction, error) {
	if ibkr.Field(row.Fields, 0) == "Total" {
		return nil, nil
	}
	desc := ibkr.Field(row.Fields, intIdxDesc)
	if strings.HasPrefix(desc, "Total") {
		return nil, nil
	}
	amount, err := ParseNumber(ibkr.Field(row.Fields, intIdxAmount))
	if err != nil || amount == 0 {
		return nil, nil
	}
	// Skip positive SYEP credits; include debit interest as fees.
	if amount > 0 {
		return nil, nil
	}
	t, err := ParseIBKRDateTime(ibkr.Field(row.Fields, intIdxDate))
	if err != nil {
		return nil, err
	}
	return cashTaxesFees(Abs(amount), t), nil
}

func cashTaxesFees(amount float64, t time.Time) *tv.Transaction {
	return &tv.Transaction{
		Symbol:      cashSymbol,
		Side:        "Taxes and fees",
		Qty:         FormatMoney(amount),
		FillPrice:   "",
		Commission:  "",
		ClosingTime: t,
	}
}
