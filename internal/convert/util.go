package convert

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dividendTickerRE = regexp.MustCompile(`^([A-Z0-9.]+)\(`)

var tzSuffixRE = regexp.MustCompile(`\s+[A-Z]{2,5}$`)

// ParseIBKRDateTime parses IBKR date/time strings (with or without timezone suffix).
func ParseIBKRDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	s = strings.ReplaceAll(s, ", ", " ")
	s = tzSuffixRE.ReplaceAllString(s, "")
	if s == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable datetime %q", s)
}

// ParseIBKRPeriodEnd parses the end date from e.g. "January 1, 2026 - May 22, 2026".
func ParseIBKRPeriodEnd(period string) (time.Time, error) {
	period = strings.TrimSpace(period)
	period = strings.Trim(period, `"`)
	i := strings.LastIndex(period, " - ")
	if i < 0 {
		return time.Time{}, fmt.Errorf("no period range in %q", period)
	}
	end := strings.TrimSpace(period[i+3:])
	for _, layout := range []string{"January 2, 2006", "Jan 2, 2006"} {
		if t, err := time.ParseInLocation(layout, end, time.Local); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable period end %q", end)
}

// ParseNumber strips commas and parses a float.
func ParseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.Trim(s, `"`)
	if s == "" || s == "--" {
		return 0, fmt.Errorf("empty number")
	}
	return strconv.ParseFloat(s, 64)
}

// FormatPrice formats a per-share price for CSV export.
func FormatPrice(v float64) string {
	return FormatNumber(v)
}

// CalendarDay returns midnight on the transaction's local calendar date.
func CalendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// FormatNumber formats a number for CSV without unnecessary trailing zeros.
func FormatNumber(v float64) string {
	if math.Abs(v-math.Round(v)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	return s
}

// FormatQty formats quantity preserving decimals when needed.
func FormatQty(v float64) string {
	return FormatNumber(v)
}

// FormatMoney formats USD cash amounts with up to 2 decimal places.
func FormatMoney(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

// Abs returns absolute value.
func Abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// CommissionString returns commission field; empty if zero.
func CommissionString(fee float64) string {
	fee = Abs(fee)
	if fee < 1e-9 {
		return ""
	}
	return FormatNumber(fee)
}

// ExtractDividendTicker parses "TSM(US8740391003) Cash Dividend..." -> TSM.
func ExtractDividendTicker(desc string) string {
	m := dividendTickerRE.FindStringSubmatch(desc)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ParseMarketValue parses "1,227.94" style values.
func ParseMarketValue(s string) (float64, error) {
	return ParseNumber(s)
}
