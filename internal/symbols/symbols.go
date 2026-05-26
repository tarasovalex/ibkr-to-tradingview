package symbols

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

//go:embed symbols_default.json
var defaultJSON []byte

// Mapper resolves ticker symbols to TradingView EXCHANGE:TICKER format.
type Mapper struct {
	mu              sync.Mutex
	exchangeByTicker map[string]string
	defaultExchange  string
	warned           map[string]bool
	warn             func(string)
}

// New creates a mapper with embedded defaults and optional override file.
func New(defaultExchange string, overridePath string, warn func(string)) (*Mapper, error) {
	m := &Mapper{
		exchangeByTicker: make(map[string]string),
		defaultExchange:  defaultExchange,
		warned:           make(map[string]bool),
		warn:             warn,
	}
	if m.warn == nil {
		m.warn = func(string) {}
	}
	if err := json.Unmarshal(defaultJSON, &m.exchangeByTicker); err != nil {
		return nil, fmt.Errorf("parse embedded symbols: %w", err)
	}
	if overridePath != "" {
		data, err := os.ReadFile(overridePath)
		if err != nil {
			return nil, fmt.Errorf("read symbol map %s: %w", overridePath, err)
		}
		var override map[string]string
		if err := json.Unmarshal(data, &override); err != nil {
			return nil, fmt.Errorf("parse symbol map: %w", err)
		}
		for k, v := range override {
			m.exchangeByTicker[k] = v
		}
	}
	return m, nil
}

// Format returns EXCHANGE:TICKER for a bare IBKR symbol.
func (m *Mapper) Format(ticker string) string {
	ex := m.exchangeByTicker[ticker]
	if ex == "" {
		ex = m.defaultExchange
		m.mu.Lock()
		if !m.warned[ticker] {
			m.warned[ticker] = true
			m.warn(fmt.Sprintf("unknown ticker %q, using %s:%s", ticker, ex, ticker))
		}
		m.mu.Unlock()
	}
	return ex + ":" + ticker
}
