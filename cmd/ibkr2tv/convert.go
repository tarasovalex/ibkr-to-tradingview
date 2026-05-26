package main

import (
	"fmt"
	"os"
	"github.com/alext/ibkr-to-tradingview/internal/convert"
	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/merge"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
	"github.com/spf13/cobra"
)

type convertFlags struct {
	output          string
	symbolMap       string
	defaultExchange string
	noDedupe        bool
	verbose         bool
}

func newConvertCmd() *cobra.Command {
	f := &convertFlags{
		defaultExchange: "NASDAQ",
	}
	cmd := &cobra.Command{
		Use:   "convert -o OUTPUT.csv INPUT [INPUT ...]",
		Short: "Convert one or more IBKR CSV files to a single TradingView CSV",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(f, args)
		},
	}
	cmd.Flags().StringVarP(&f.output, "output", "o", "", "output TradingView CSV path (required)")
	cmd.Flags().StringVar(&f.symbolMap, "symbol-map", "", "optional JSON map of ticker to exchange")
	cmd.Flags().StringVar(&f.defaultExchange, "default-exchange", "NASDAQ", "exchange for unmapped tickers")
	cmd.Flags().BoolVar(&f.noDedupe, "no-dedupe", false, "disable exact-match deduplication")
	cmd.Flags().BoolVarP(&f.verbose, "verbose", "v", false, "print per-file stats and warnings")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func runConvert(f *convertFlags, inputs []string) error {
	var warnings []string
	warn := func(msg string) {
		warnings = append(warnings, msg)
		if f.verbose {
			fmt.Fprintln(os.Stderr, "warning:", msg)
		}
	}

	mapper, err := symbols.New(f.defaultExchange, f.symbolMap, warn)
	if err != nil {
		return err
	}

	opts := convert.Options{Mapper: mapper, Warn: warn}
	var slices [][]tv.Transaction
	var allRows []ibkr.Row

	for _, path := range inputs {
		rows, err := ibkr.ParseFile(path)
		if err != nil {
			return err
		}
		allRows = append(allRows, rows...)
		res, err := convert.ParseRows(path, rows, opts)
		if err != nil {
			return err
		}
		if f.verbose {
			fmt.Fprintf(os.Stderr, "%s", path)
			if res.Period != "" {
				fmt.Fprintf(os.Stderr, " [%s]", res.Period)
			}
			fmt.Fprintln(os.Stderr)
			for k, n := range res.Counts {
				fmt.Fprintf(os.Stderr, "  %s: %d\n", k, n)
			}
			fmt.Fprintf(os.Stderr, "  total: %d\n", len(res.Transactions))
		}
		slices = append(slices, res.Transactions)
	}

	combined := merge.Combine(slices, !f.noDedupe)
	combined = convert.AdjustTransferLots(allRows, combined)
	combined = convert.ReconcileUSDCash(allRows, combined)
	merge.SortForImport(combined)
	if f.verbose {
		fmt.Fprintf(os.Stderr, "merged output: %d transactions\n", len(combined))
		if len(warnings) > 0 && !f.verbose {
			fmt.Fprintf(os.Stderr, "%d warning(s)\n", len(warnings))
		}
	}

	if err := tv.WriteCSV(f.output, combined); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d rows)\n", f.output, len(combined))
	return nil
}
