package tv

import (
	"encoding/csv"
	"io"
	"os"
)

// WriteCSV writes transactions to a TradingView-compatible CSV file.
func WriteCSV(path string, txs []Transaction) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteCSVWriter(f, txs)
}

// WriteCSVWriter writes transactions to w.
func WriteCSVWriter(w io.Writer, txs []Transaction) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"Symbol", "Side", "Qty", "Fill Price", "Commission", "Closing Time"}); err != nil {
		return err
	}
	for _, t := range txs {
		if err := cw.Write([]string{
			t.Symbol,
			t.Side,
			t.Qty,
			t.FillPrice,
			t.Commission,
			t.ClosingTimeString(),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
