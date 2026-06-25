package money

import (
	"fmt"
	"strconv"
	"strings"
)

// Currency describes how a currency is formatted.
type Currency struct {
	Code     string
	Symbol   string
	Decimals int // minor-unit exponent (USD=2, JPY=0, BHD=3)
}

// registry of common ISO-4217 currencies. Unknown codes default to 2 decimals.
var registry = map[string]Currency{
	"USD": {"USD", "$", 2},
	"EUR": {"EUR", "€", 2},
	"GBP": {"GBP", "£", 2},
	"JPY": {"JPY", "¥", 0},
	"CNY": {"CNY", "¥", 2},
	"AED": {"AED", "د.إ", 2},
	"SAR": {"SAR", "﷼", 2},
	"EGP": {"EGP", "E£", 2},
	"INR": {"INR", "₹", 2},
	"CAD": {"CAD", "$", 2},
	"AUD": {"AUD", "$", 2},
	"CHF": {"CHF", "CHF", 2},
	"BHD": {"BHD", "BD", 3},
	"KWD": {"KWD", "KD", 3},
	"BTC": {"BTC", "₿", 8},
}

// symbols maps code → symbol for parsing/formatting.
var symbols = func() map[string]string {
	m := map[string]string{}
	for c, cur := range registry {
		m[c] = cur.Symbol
	}
	return m
}()

// canonicalSymbol maps an ambiguous symbol to its preferred currency, so
// parsing "$12.50" is deterministic (→ USD, not a random $-currency).
var canonicalSymbol = map[string]string{
	"$": "USD",
	"¥": "JPY",
	"£": "GBP",
	"€": "EUR",
}

// symbolFor finds the currency whose symbol prefixes s, deterministically:
// longest matching symbol first, with a canonical winner for shared symbols.
func symbolFor(s string) (code, sym string, ok bool) {
	best := ""
	for c, cur := range registry {
		if cur.Symbol == "" || !strings.HasPrefix(s, cur.Symbol) {
			continue
		}
		switch {
		case len(cur.Symbol) > len(best):
			best, code = cur.Symbol, c
		case len(cur.Symbol) == len(best):
			// tie on symbol length → prefer the canonical currency, then lexically
			if canonicalSymbol[cur.Symbol] == c {
				code = c
			} else if canonicalSymbol[cur.Symbol] != code && c < code {
				code = c
			}
		}
	}
	if best == "" {
		return "", "", false
	}
	if pref, has := canonicalSymbol[best]; has {
		code = pref
	}
	return code, best, true
}

// Register adds or overrides a currency definition (e.g. a custom token).
func Register(c Currency) {
	c.Code = normCur(c.Code)
	registry[c.Code] = c
	symbols[c.Code] = c.Symbol
}

// Get returns a currency definition (with a 2-decimal default for unknowns).
func Get(code string) Currency {
	code = normCur(code)
	if c, ok := registry[code]; ok {
		return c
	}
	return Currency{Code: code, Symbol: code + " ", Decimals: 2}
}

// Decimals returns the minor-unit exponent for a currency code.
func Decimals(code string) int { return Get(code).Decimals }

// Symbol returns the currency symbol.
func Symbol(code string) string { return Get(code).Symbol }

// Format renders minor units in a currency, e.g. Format(123450, "USD") → "$1,234.50".
func Format(minorUnits int64, code string) string {
	cur := Get(code)
	neg := minorUnits < 0
	if neg {
		minorUnits = -minorUnits
	}
	scale := pow10(cur.Decimals)
	whole := minorUnits / scale
	frac := minorUnits % scale

	s := group(strconv.FormatInt(whole, 10))
	if cur.Decimals > 0 {
		fracStr := strconv.FormatInt(frac, 10)
		for len(fracStr) < cur.Decimals {
			fracStr = "0" + fracStr
		}
		s = s + "." + fracStr
	}
	out := cur.Symbol + s
	if neg {
		out = "-" + out
	}
	return out
}

// group inserts thousands separators into an integer string.
func group(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if n > pre {
			b.WriteByte(',')
		}
	}
	for i := pre; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// --- optional FX service ---

// RateProvider returns the exchange rate from one currency to another.
type RateProvider interface {
	Rate(from, to string) (float64, error)
}

// Convert converts m to another currency using a RateProvider.
func Convert(m Money, to string, rp RateProvider) (Money, error) {
	to = normCur(to)
	if m.currency == to {
		return m, nil
	}
	rate, err := rp.Rate(m.currency, to)
	if err != nil {
		return Money{}, fmt.Errorf("money: convert %s→%s: %w", m.currency, to, err)
	}
	// convert via major units; FromMajor re-scales to the target's precision.
	major := float64(m.amount) / float64(pow10(Decimals(m.currency)))
	return FromMajor(major*rate, to), nil
}
