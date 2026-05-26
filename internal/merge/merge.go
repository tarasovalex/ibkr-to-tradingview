package merge

import (
	"sort"

	"github.com/alext/ibkr-to-tradingview/internal/tv"
)

// Combine concatenates slices, sorts by closing time, and optionally dedupes.
func Combine(slices [][]tv.Transaction, dedupe bool) []tv.Transaction {
	var all []tv.Transaction
	for _, s := range slices {
		all = append(all, s...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ClosingTime.Before(all[j].ClosingTime)
	})
	if !dedupe {
		return all
	}
	seen := make(map[string]struct{}, len(all))
	out := make([]tv.Transaction, 0, len(all))
	for _, t := range all {
		k := t.DedupeKey()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, t)
	}
	return out
}
