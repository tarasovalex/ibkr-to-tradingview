package convert

import (
	"fmt"

	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// Result holds parsed transactions and metadata.
type Result struct {
	Period       string
	Transactions []tv.Transaction
	Counts       map[string]int
	Warnings     []string
}

// Options configures file conversion.
type Options struct {
	Mapper               *symbols.Mapper
	Warn                 func(string)
	TransferCostPerShare map[string]float64
}

// ParseFile converts one IBKR activity statement to TradingView transactions.
func ParseFile(path string, opts Options) (*Result, error) {
	rows, err := ibkr.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ParseRows(path, rows, opts)
}

// ParseRows converts parsed IBKR rows.
func ParseRows(source string, rows []ibkr.Row, opts Options) (*Result, error) {
	if opts.Warn == nil {
		opts.Warn = func(string) {}
	}
	res := &Result{
		Counts: make(map[string]int),
	}
	curCtx := scanCurrencyContext(rows)
	if opts.TransferCostPerShare == nil {
		opts.TransferCostPerShare = scanTransferCostPerShare(rows, opts.Warn)
	}

	for _, row := range rows {
		if row.Section == ibkr.SectionStatement && row.RowType == ibkr.RowData {
			if ibkr.Field(row.Fields, 0) == "Period" {
				res.Period = ibkr.Field(row.Fields, 1)
			}
		}
	}

	for _, row := range rows {
		if !ibkr.IsDataRow(row.RowType) {
			continue
		}
		var tx *tv.Transaction
		var err error
		var kind string

		switch row.Section {
		case ibkr.SectionTrades:
			tx, err = convertTrade(row, opts.Mapper)
			kind = "trade"
		case ibkr.SectionDividends:
			tx, err = convertDividend(row, opts.Mapper)
			kind = "dividend"
		case ibkr.SectionDepositsWithdrawals:
			tx, err = convertDeposit(row, curCtx, opts.Warn)
			kind = "deposit"
		case ibkr.SectionFees:
			tx, err = convertFee(row, curCtx, opts.Warn)
			kind = "fee"
		case ibkr.SectionWithholdingTax:
			tx, err = convertWithholding(row)
			kind = "withholding"
		case ibkr.SectionInterest:
			tx, err = convertInterest(row)
			kind = "interest"
		case ibkr.SectionTransfers:
			xferTxs, err := convertTransfer(row, opts.Mapper, opts.TransferCostPerShare)
			if err != nil {
				return nil, fmt.Errorf("%s: %s row: %w", source, row.Section, err)
			}
			for _, xtx := range xferTxs {
				res.Transactions = append(res.Transactions, xtx)
			}
			if len(xferTxs) > 0 {
				res.Counts["transfer"]++
			}
			continue
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %s row: %w", source, row.Section, err)
		}
		if tx == nil {
			continue
		}
		res.Transactions = append(res.Transactions, *tx)
		res.Counts[kind]++
	}

	if periodEnd := scanStatementPeriodEnd(rows); !periodEnd.IsZero() {
		res.Transactions = append(res.Transactions, ParseCashReportRows(rows, periodEnd)...)
	}
	return res, nil
}
