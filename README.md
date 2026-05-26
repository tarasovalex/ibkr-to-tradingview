# ibkr-to-tradingview

Convert Interactive Brokers (IBKR) **Activity Statement** CSV exports into a single CSV you can import into a [TradingView portfolio](https://www.tradingview.com/support/solutions/43000756010/).

## What it does

IBKR reports activity in a multi-section CSV format. TradingView expects a flat list of transactions (symbol, side, quantity, price, commission, time). This tool reads one or more IBKR statement files, maps the relevant rows to TradingView’s format, merges them, and writes one import file.

Supported activity types include stock trades, dividends, cash deposits and withdrawals, fees, withholding tax, debit interest, and inbound stock transfers. Forex, stock-loan (SYEP), credit interest, accruals, and summary-only rows are ignored.

## Why use it

- **IBKR export limits** — Activity statements are typically exported per calendar year (or fixed date range). If your portfolio history spans multiple periods, you need several CSV files.
- **TradingView import** — Portfolio import expects a single file in TradingView’s column layout, not IBKR’s sectioned format.
- **One command** — Pass all statement files at once; the tool merges them and removes exact duplicate rows (e.g. overlapping statement periods).

## How it works

1. Parse each input CSV using IBKR’s section headers (`Trades`, `Dividends`, `Deposits & Withdrawals`, etc.).
2. Convert each supported row into a TradingView transaction (including `$CASH` rows for non-trade cash events).
3. Resolve stock symbols to `EXCHANGE:TICKER` (e.g. `NASDAQ:AAPL`) using a built-in map, with optional overrides.
4. Merge all inputs and deduplicate exact matches unless you disable deduplication.
5. Adjust transfer lots for sell cost-basis matching, reconcile USD cash to IBKR’s ending balance, then sort for import (same-day deposits before buys).
6. Write the output CSV ready for TradingView’s import dialog.

## Limitations

- **Input format** — Standard IBKR **Activity Statement** CSV only (not Flex Query or other report types).
- **Asset types** — **Stocks** only in Trades and Transfers (no options, bonds, forex, etc.).
- **Transfers** — **Transfers in** only; outbound transfers are not imported.
- **Interest** — Debit interest is imported as `$CASH` taxes/fees; positive interest and SYEP credits are skipped.
- **Cash** — USD cash is reconciled to IBKR’s ending USD balance; ILS cash is converted on deposits/fees but not tracked as a separate balance.

## Requirements

- [Go](https://go.dev/dl/) 1.22 or newer (to build from source)

## Install

### From a clone of this repository

```bash
git clone https://github.com/alext/ibkr-to-tradingview.git
cd ibkr-to-tradingview
go install ./cmd/ibkr2tv
```

The `ibkr2tv` binary is installed to your Go bin directory (usually `$(go env GOPATH)/bin`). Ensure that directory is on your `PATH`.

### Build locally (no install)

```bash
git clone https://github.com/alext/ibkr-to-tradingview.git
cd ibkr-to-tradingview
make build
./bin/ibkr2tv convert -h
```

Or install the built binary with `make install` (same as `go install ./cmd/ibkr2tv`).

### Install from module path

If the module is published and reachable by Go:

```bash
go install github.com/alext/ibkr-to-tradingview/cmd/ibkr2tv@latest
```

## Usage

### Basic conversion

Export one or more Activity Statement CSV files from IBKR (see below), then run:

```bash
ibkr2tv convert -o ./output/tradingview-portfolio.csv \
  ./statements/activity-2024.csv \
  ./statements/activity-2025.csv \
  ./statements/activity-2026-ytd.csv
```

Use as many input files as you need; order does not matter. Overlapping periods are handled by deduplicating exact duplicate transactions (same symbol, side, quantity, fill price, commission, and closing time).

On success, the tool prints `wrote … (N rows)` to stderr.

### Verbose output

```bash
ibkr2tv convert -v -o ./output/tradingview-portfolio.csv ./statements/*.csv
```

With `-v`, the tool also prints each file’s statement period, per-section counts, merge size, and warnings (e.g. unknown tickers, missing ILS/USD ratio). Re-run with `-v` if imports fail on symbols or cash looks wrong.

### Custom symbol → exchange map

TradingView portfolio symbols use **`EXCHANGE:TICKER`** format (e.g. `NASDAQ:AAPL`, `NYSE:TSM`). Defaults are embedded in the binary at build time. To override or extend without rebuilding:

```bash
ibkr2tv convert -o ./output/tradingview-portfolio.csv \
  --symbol-map ./my-symbols.json \
  ./statements/activity-2025.csv
```

Example `my-symbols.json` (maps IBKR ticker → exchange code):

```json
{
  "AAPL": "NASDAQ",
  "BRK.B": "NYSE"
}
```

Unknown tickers use `--default-exchange` (default: `NASDAQ`) and emit a warning when `-v` is set.

### Flags

| Flag | Description |
|------|-------------|
| `-o`, `--output` | Output CSV path (required) |
| `--symbol-map` | JSON file `{ "TICKER": "EXCHANGE", ... }` overriding built-in defaults |
| `--default-exchange` | Exchange for unmapped tickers (default: `NASDAQ`) |
| `--no-dedupe` | Keep duplicate rows (default: remove exact matches) |
| `-v`, `--verbose` | Per-file period, counts, and warnings |

## Output CSV

The output file has these columns (TradingView import layout):

| Column | Description |
|--------|-------------|
| Symbol | `EXCHANGE:TICKER` for stocks, or `$CASH` for cash events |
| Side | `Buy`, `Sell`, `Dividend`, `Deposit`, `Withdrawal`, or `Taxes and fees` |
| Qty | Share quantity, dividend USD amount, or cash amount |
| Fill Price | Per-share price (empty for cash/dividend rows) |
| Commission | Commission when not folded into fill price on buys |
| Closing Time | `YYYY-MM-DD H:MM:SS` |

## Import into TradingView

1. Open [TradingView](https://www.tradingview.com/) and go to your portfolio (or create one).
2. Use the portfolio menu to **Import transactions** (wording may vary slightly by UI version).
3. Select the CSV produced by `ibkr2tv convert` (`-o` path).
4. Review the preview and confirm. Fix any symbol/exchange mismatches with `--symbol-map` if TradingView does not recognize a ticker.

## Exporting from IBKR

1. Log in to **IBKR Account Management** (or Client Portal → Reports).
2. Open **Reports** → **Activity** (or **Activity Statement**).
3. Choose the date range and account, then download **CSV** (not PDF).
4. Repeat for each period you need (e.g. each calendar year plus current year-to-date).
5. Pass every downloaded file to a single `ibkr2tv convert` command.

File names vary by account and period; any path is fine as long as the content is a standard IBKR Activity Statement CSV.

## What gets converted

| IBKR section | TradingView |
|--------------|-------------|
| Trades (stocks only) | Buy / Sell |
| Dividends | Dividend (quantity = total USD received) |
| Deposits & Withdrawals | `$CASH` Deposit / Withdrawal |
| Fees, Withholding Tax, debit Interest | `$CASH` Taxes and fees |
| Transfers In (stocks) | Buy at carried-over cost basis (see below) |

**Skipped:** forex trades, SYEP and other stock-loan credits, positive/credit interest, accruals, statement summaries, non-stock trade lines, and transfers out.

## Cost basis and average price

TradingView rebuilds average cost from imported **Fill Price** values. The converter aligns with IBKR where the statement provides enough data:

- **Stock buys** use IBKR **Basis ÷ quantity** (includes commissions), not raw trade price alone.
- **Transfers in** use cost basis derived from **Open Positions** minus net trade basis in that statement period—not the transfer row’s market value.
- **Sells** keep the execution price; when IBKR reports cost removed on a sell, large inbound transfers are split into sub-lots so FIFO-style matching in TradingView can consume the same basis IBKR used.

Small differences can remain if TradingView uses average-cost rather than FIFO, or if lot matching cannot be inferred from the CSV.

## Cash balance (USD)

TradingView derives cash from `$CASH` rows plus cash flows from stock trades. The converter:

- Converts **ILS** deposits and fees to USD using each statement’s ILS/USD deposit totals.
- Pairs each **transfer in** with a same-day `$CASH` deposit (same cost as the buy) so transfers do not drain cash.
- Sorts each calendar day **deposits before stock buys** so TradingView applies wire transfers before purchases.
- Imports **Cash FX Translation** and **Payment In Lieu** from the Cash Report.
- Counts **withholding tax** outflows only (skips positive reversal lines).
- **Reconciles** to the latest statement’s **Ending Cash (USD)** with a deposit or withdrawal row if the simulated balance differs by more than **$0.02**.

ILS cash is not modeled separately; only USD cash is aligned to IBKR’s USD ending balance.

## ILS deposits

If your statement lists Israeli shekel (ILS) deposits with both `Total` (ILS) and `Total in USD` summary rows, each ILS deposit is converted to USD using that ratio. USD rows are unchanged. If the ratio lines are missing, raw amounts are used and a warning is emitted (visible with `-v`).

## Symbol map in the repository

[`symbols.json`](symbols.json) lists the default ticker → exchange mappings. The binary embeds a copy from [`internal/symbols/symbols_default.json`](internal/symbols/symbols_default.json) at **build time**—editing `symbols.json` alone does not change an already-installed binary. Use `--symbol-map` to override at runtime, or rebuild after changing the embedded file.

## Tests

```bash
go test ./...
```

Or via Make:

```bash
make test
```
