package convert_test

import (
	"testing"

	"github.com/alext/ibkr-to-tradingview/internal/convert"
)

func TestParseIBKRDateTime(t *testing.T) {
	tm, err := convert.ParseIBKRDateTime(`"2026-01-13, 10:18:27"`)
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2026 || tm.Month() != 1 || tm.Day() != 13 || tm.Hour() != 10 {
		t.Fatalf("unexpected time: %v", tm)
	}
	tm, err = convert.ParseIBKRDateTime(`"2026-05-24, 05:32:17 EDT"`)
	if err != nil {
		t.Fatal(err)
	}
	if tm.Day() != 24 || tm.Hour() != 5 {
		t.Fatalf("unexpected time with TZ: %v", tm)
	}
}

func TestParseIBKRPeriodEnd(t *testing.T) {
	tm, err := convert.ParseIBKRPeriodEnd(`January 1, 2026 - May 22, 2026`)
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2026 || tm.Month() != 5 || tm.Day() != 22 || tm.Hour() != 0 {
		t.Fatalf("unexpected period end: %v", tm)
	}
}

func TestExtractDividendTicker(t *testing.T) {
	got := convert.ExtractDividendTicker("TSM(US8740391003) Cash Dividend USD 0.79 per Share")
	if got != "TSM" {
		t.Fatalf("got %q want TSM", got)
	}
}
