package convert

import (
	"strings"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

const sectionCashReport = "Cash Report"

// cash report row: activity label, currency/summary, amount, ...
const (
	cashRptIdxLabel    = 0
	cashRptIdxCurrency = 1
	cashRptIdxAmount   = 2
)

// ParseCashReportRows emits $CASH rows from Cash Report (FX translation, payment in lieu, etc.).
func ParseCashReportRows(rows []ibkr.Row, periodEnd time.Time) []tv.Transaction {
	var out []tv.Transaction
	for _, row := range rows {
		if row.Section != sectionCashReport || row.RowType != ibkr.RowData {
			continue
		}
		label := ibkr.Field(row.Fields, cashRptIdxLabel)
		currency := ibkr.Field(row.Fields, cashRptIdxCurrency)
		if currency != "USD" {
			continue
		}
		amount, err := ParseNumber(ibkr.Field(row.Fields, cashRptIdxAmount))
		if err != nil || amount == 0 {
			continue
		}
		switch {
		case strings.Contains(label, "Cash FX Translation"):
			out = append(out, *cashFlow(amount, periodEnd))
		case strings.Contains(label, "Payment In Lieu"):
			out = append(out, *cashDeposit(amount, periodEnd))
		}
	}
	return out
}

func cashFlow(amount float64, t time.Time) *tv.Transaction {
	if amount > 0 {
		return cashDeposit(amount, t)
	}
	return cashTaxesFees(Abs(amount), t)
}

func cashDeposit(amount float64, t time.Time) *tv.Transaction {
	side := "Deposit"
	if amount < 0 {
		side = "Withdrawal"
		amount = Abs(amount)
	}
	return &tv.Transaction{
		Symbol:      cashSymbol,
		Side:        side,
		Qty:         FormatMoney(amount),
		FillPrice:   "",
		Commission:  "",
		ClosingTime: t,
	}
}

// scanEndingCashUSD returns the last USD "Ending Cash" from Cash Report rows.
func scanEndingCashUSD(rows []ibkr.Row) (float64, bool) {
	var ending float64
	found := false
	for _, row := range rows {
		if row.Section != sectionCashReport || row.RowType != ibkr.RowData {
			continue
		}
		if ibkr.Field(row.Fields, cashRptIdxLabel) != "Ending Cash" {
			continue
		}
		if ibkr.Field(row.Fields, cashRptIdxCurrency) != "USD" {
			continue
		}
		v, err := ParseNumber(ibkr.Field(row.Fields, cashRptIdxAmount))
		if err != nil {
			continue
		}
		ending = v
		found = true
	}
	return ending, found
}

// scanStatementPeriodEnd returns the last statement period end date (midnight, local).
func scanStatementPeriodEnd(rows []ibkr.Row) time.Time {
	var last time.Time
	for _, row := range rows {
		if row.Section != ibkr.SectionStatement || row.RowType != ibkr.RowData {
			continue
		}
		if ibkr.Field(row.Fields, 0) != "Period" {
			continue
		}
		t, err := ParseIBKRPeriodEnd(ibkr.Field(row.Fields, 1))
		if err != nil {
			continue
		}
		if last.IsZero() || t.After(last) {
			last = t
		}
	}
	return last
}

func latestTransactionTime(txs []tv.Transaction) time.Time {
	var last time.Time
	for _, tx := range txs {
		if last.IsZero() || tx.ClosingTime.After(last) {
			last = tx.ClosingTime
		}
	}
	return last
}

// closingTimeForCash picks a statement date suitable for TradingView (not in the future).
func closingTimeForCash(rows []ibkr.Row, txs []tv.Transaction) time.Time {
	t := scanStatementPeriodEnd(rows)
	if t.IsZero() {
		t = latestTransactionTime(txs)
	}
	if !t.IsZero() {
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if t.IsZero() || t.After(today) {
		return today
	}
	return t
}

// SimulateUSDCash estimates USD cash from TradingView import rules.
func SimulateUSDCash(txs []tv.Transaction) float64 {
	var cash float64
	for _, tx := range txs {
		qty, err := ParseNumber(tx.Qty)
		if err != nil {
			continue
		}
		price, _ := ParseNumber(tx.FillPrice)
		comm, _ := ParseNumber(tx.Commission)

		if tx.Symbol == cashSymbol {
			switch tx.Side {
			case "Deposit":
				cash += qty
			case "Withdrawal":
				cash -= qty
			case "Taxes and fees":
				cash -= qty
			}
			continue
		}
		switch tx.Side {
		case "Buy":
			cash -= qty*price + comm
		case "Sell":
			cash += qty*price - comm
		case "Dividend":
			cash += qty
		}
	}
	return cash
}

// ReconcileUSDCash appends a deposit or withdrawal so cash matches IBKR ending USD cash.
func ReconcileUSDCash(rows []ibkr.Row, txs []tv.Transaction) []tv.Transaction {
	target, ok := scanEndingCashUSD(rows)
	if !ok {
		return txs
	}
	current := SimulateUSDCash(txs)
	delta := target - current
	if Abs(delta) < 0.02 {
		return txs
	}
	t := closingTimeForCash(rows, txs)
	if delta > 0 {
		txs = append(txs, *cashDeposit(delta, t))
	} else {
		txs = append(txs, tv.Transaction{
			Symbol:      cashSymbol,
			Side:        "Withdrawal",
			Qty:         FormatMoney(Abs(delta)),
			ClosingTime: t,
		})
	}
	return txs
}
