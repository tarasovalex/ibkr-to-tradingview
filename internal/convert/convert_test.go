package convert_test

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/alext/ibkr-to-tradingview/internal/convert"
	"github.com/alext/ibkr-to-tradingview/internal/ibkr"
	"github.com/alext/ibkr-to-tradingview/internal/merge"
	"github.com/alext/ibkr-to-tradingview/internal/symbols"
	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

func TestParseTradesSnippet(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "testdata", "trades_snippet.csv")
	res, err := convert.ParseFile(path, convert.Options{Mapper: mapper})
	if err != nil {
		t.Fatal(err)
	}
	if res.Counts["trade"] != 2 {
		t.Fatalf("trade count: got %d want 2", res.Counts["trade"])
	}
	if res.Counts["dividend"] != 1 {
		t.Fatalf("dividend count: got %d want 1", res.Counts["dividend"])
	}
	var foundSell, foundDiv bool
	for _, tx := range res.Transactions {
		if tx.Symbol == "NASDAQ:NVDA" && tx.Side == "Sell" && tx.Qty == "15" {
			foundSell = true
		}
		if tx.Symbol == "NYSE:TSM" && tx.Side == "Dividend" && tx.Qty == "26.04" {
			foundDiv = true
		}
	}
	if !foundSell {
		t.Error("missing NVDA sell")
	}
	if !foundDiv {
		t.Error("missing TSM dividend")
	}
}

func TestParseDepositsILSRatio(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "testdata", "deposits_snippet.csv")
	res, err := convert.ParseFile(path, convert.Options{Mapper: mapper})
	if err != nil {
		t.Fatal(err)
	}
	if res.Counts["deposit"] != 3 {
		t.Fatalf("deposit count: got %d want 3", res.Counts["deposit"])
	}
	ratio := 3920.5 / 12000
	wantILS := 7000 * ratio
	var ilsDeposit string
	for _, tx := range res.Transactions {
		if tx.Side == "Deposit" && tx.Qty == "41" {
			continue
		}
		if tx.Side == "Deposit" && tx.Symbol == "$CASH" && ilsDeposit == "" {
			ilsDeposit = tx.Qty
		}
	}
	if ilsDeposit == "" {
		t.Fatal("missing scaled ILS deposit")
	}
	got, err := strconv.ParseFloat(ilsDeposit, 64)
	if err != nil {
		t.Fatal(err)
	}
	if abs(got-wantILS) > 0.01 {
		t.Fatalf("scaled deposit: got %v want ~%v", got, wantILS)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestNVDATransferUsesCostBasis(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "testdata", "activity_2025.csv")
	res, err := convert.ParseFile(path, convert.Options{Mapper: mapper})
	if err != nil {
		t.Fatal(err)
	}
	want := 6257.30 / 110 // derived from Open Positions minus trade basis
	for _, tx := range res.Transactions {
		if tx.Symbol != "NASDAQ:NVDA" || tx.Side != "Buy" || tx.Qty != "110" {
			continue
		}
		got, err := strconv.ParseFloat(tx.FillPrice, 64)
		if err != nil {
			t.Fatal(err)
		}
		if abs(got-want) > 0.02 {
			t.Fatalf("transfer fill price: got %v want ~%v (market value was 126.63)", got, want)
		}
		return
	}
	t.Fatal("missing NVDA transfer buy")
}

func TestMergedNVDAAvgCostNearIBKR(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join("..", "..", "testdata")
	var slices [][]tv.Transaction
	var rows []ibkr.Row
	for _, name := range []string{"activity_2025.csv", "activity_2026_ytd.csv"} {
		path := filepath.Join(base, name)
		fileRows, err := ibkr.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, fileRows...)
		res, err := convert.ParseRows(path, fileRows, convert.Options{Mapper: mapper})
		if err != nil {
			t.Fatal(err)
		}
		slices = append(slices, res.Transactions)
	}
	combined := merge.Combine(slices, true)
	combined = convert.AdjustTransferLots(rows, combined)
	merge.SortForImport(combined)
	shares, avg, ok := convert.SimulateFIFOAvgCost(combined, "NASDAQ:NVDA")
	if !ok {
		t.Fatal("no NVDA position")
	}
	const ibkrAvg = 84.631169257
	const ibkrShares = 130.0074
	if abs(shares-ibkrShares) > 0.01 {
		t.Fatalf("shares: got %v want %v", shares, ibkrShares)
	}
	if abs(avg-ibkrAvg) > 0.05 {
		t.Fatalf("FIFO avg cost: got %v want ~%v", avg, ibkrAvg)
	}
}

func TestUSDCashReconcilesToIBKR(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join("..", "..", "testdata")
	var slices [][]tv.Transaction
	var rows []ibkr.Row
	for _, name := range []string{"activity_2025.csv", "activity_2026_ytd.csv"} {
		path := filepath.Join(base, name)
		fileRows, err := ibkr.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, fileRows...)
		res, err := convert.ParseRows(path, fileRows, convert.Options{Mapper: mapper})
		if err != nil {
			t.Fatal(err)
		}
		slices = append(slices, res.Transactions)
	}
	combined := merge.Combine(slices, true)
	combined = convert.AdjustTransferLots(rows, combined)
	combined = convert.ReconcileUSDCash(rows, combined)
	merge.SortForImport(combined)

	const ibkrUSD = 22692.188654552
	got := convert.SimulateUSDCash(combined)
	if abs(got-ibkrUSD) > 0.05 {
		t.Fatalf("USD cash: got %v want ~%v", got, ibkrUSD)
	}
	last := combined[len(combined)-1]
	if last.Symbol != "$CASH" {
		t.Fatal("expected reconcile row last")
	}
	now := time.Now()
	if last.ClosingTime.After(now) {
		t.Fatalf("reconcile closing time in future: %v", last.ClosingTime)
	}
}

func TestUSDCashReconcileIgnoresInputFileOrder(t *testing.T) {
	mapper, err := symbols.New("NASDAQ", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join("..", "..", "testdata")
	names := []string{"activity_2026_ytd.csv", "activity_2025.csv"} // reversed
	var slices [][]tv.Transaction
	var rows []ibkr.Row
	for _, name := range names {
		path := filepath.Join(base, name)
		fileRows, err := ibkr.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, fileRows...)
		res, err := convert.ParseRows(path, fileRows, convert.Options{Mapper: mapper})
		if err != nil {
			t.Fatal(err)
		}
		slices = append(slices, res.Transactions)
	}
	combined := merge.Combine(slices, true)
	combined = convert.AdjustTransferLots(rows, combined)
	combined = convert.ReconcileUSDCash(rows, combined)

	const ibkrUSD = 22692.188654552
	got := convert.SimulateUSDCash(combined)
	if abs(got-ibkrUSD) > 0.05 {
		t.Fatalf("USD cash: got %v want ~%v", got, ibkrUSD)
	}
	// Must not apply a multi-thousand withdrawal from the older statement's ending cash.
	for _, tx := range combined {
		if tx.Symbol == "$CASH" && tx.Side == "Withdrawal" {
			qty, _ := strconv.ParseFloat(tx.Qty, 64)
			if qty > 1000 {
				t.Fatalf("spurious large withdrawal %.2f when inputs are reversed", qty)
			}
		}
	}
}
