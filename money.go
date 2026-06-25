// Package money is a currency-safe money value type for togo (Fowler's Money
// pattern, like money-rails / django-money). Amounts are stored as integer
// minor units (e.g. cents) to avoid floating-point rounding errors, tagged with
// an ISO-4217 currency. Arithmetic across mismatched currencies errors;
// allocation splits an amount without losing a cent.
//
//	price := money.New(1999, "USD")          // $19.99
//	total, _ := price.Add(money.New(100, "USD"))
//	parts := total.Allocate(1, 1, 1)         // 3 even parts, remainder distributed
package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Money is an amount in a currency, held as integer minor units.
type Money struct {
	amount   int64
	currency string
}

// New returns Money for an exact number of minor units (cents) in cur.
func New(minorUnits int64, cur string) Money {
	return Money{amount: minorUnits, currency: normCur(cur)}
}

// FromMajor returns Money from a major-unit float (e.g. 19.99), rounded to the
// currency's decimal precision.
func FromMajor(major float64, cur string) Money {
	cur = normCur(cur)
	scale := pow10(Decimals(cur))
	return Money{amount: int64(math.Round(major * float64(scale))), currency: cur}
}

// Amount returns the integer minor units.
func (m Money) Amount() int64 { return m.amount }

// Currency returns the ISO-4217 code.
func (m Money) Currency() string { return m.currency }

// Major returns the value in major units as a float (lossy — for display only).
func (m Money) Major() float64 { return float64(m.amount) / float64(pow10(Decimals(m.currency))) }

func (m Money) sameCurrency(o Money) error {
	if m.currency != o.currency {
		return fmt.Errorf("money: currency mismatch %s vs %s", m.currency, o.currency)
	}
	return nil
}

// Add returns m+o (currencies must match).
func (m Money) Add(o Money) (Money, error) {
	if err := m.sameCurrency(o); err != nil {
		return Money{}, err
	}
	return Money{m.amount + o.amount, m.currency}, nil
}

// Sub returns m-o (currencies must match).
func (m Money) Sub(o Money) (Money, error) {
	if err := m.sameCurrency(o); err != nil {
		return Money{}, err
	}
	return Money{m.amount - o.amount, m.currency}, nil
}

// Mul scales the amount by a factor (rounded to the nearest minor unit).
func (m Money) Mul(factor float64) Money {
	return Money{int64(math.Round(float64(m.amount) * factor)), m.currency}
}

// Neg returns the negated amount.
func (m Money) Neg() Money { return Money{-m.amount, m.currency} }

// Abs returns the absolute amount.
func (m Money) Abs() Money {
	if m.amount < 0 {
		return Money{-m.amount, m.currency}
	}
	return m
}

// Allocate distributes the amount across the given integer ratios, giving each
// its floor share and handing out the remainder one minor unit at a time (so
// the parts sum back to exactly the original — no lost cents).
func (m Money) Allocate(ratios ...int) []Money {
	total := 0
	for _, r := range ratios {
		total += r
	}
	if total <= 0 {
		return nil
	}
	out := make([]Money, len(ratios))
	var allocated int64
	for i, r := range ratios {
		share := m.amount * int64(r) / int64(total)
		out[i] = Money{share, m.currency}
		allocated += share
	}
	remainder := m.amount - allocated
	for i := 0; remainder != 0; i = (i + 1) % len(out) {
		if remainder > 0 {
			out[i].amount++
			remainder--
		} else {
			out[i].amount--
			remainder++
		}
	}
	return out
}

// Split divides the amount into n even parts (remainder distributed).
func (m Money) Split(n int) []Money {
	if n <= 0 {
		return nil
	}
	ratios := make([]int, n)
	for i := range ratios {
		ratios[i] = 1
	}
	return m.Allocate(ratios...)
}

// Cmp returns -1, 0, or +1 comparing m to o (currencies must match; panics on mismatch).
func (m Money) Cmp(o Money) int {
	if err := m.sameCurrency(o); err != nil {
		panic(err)
	}
	switch {
	case m.amount < o.amount:
		return -1
	case m.amount > o.amount:
		return 1
	default:
		return 0
	}
}

// Equal reports whether m and o are the same currency and amount.
func (m Money) Equal(o Money) bool       { return m.currency == o.currency && m.amount == o.amount }
func (m Money) LessThan(o Money) bool    { return m.Cmp(o) < 0 }
func (m Money) GreaterThan(o Money) bool { return m.Cmp(o) > 0 }
func (m Money) IsZero() bool             { return m.amount == 0 }
func (m Money) IsNegative() bool         { return m.amount < 0 }
func (m Money) IsPositive() bool         { return m.amount > 0 }

// String renders the money with its currency symbol and grouping, e.g. "$1,234.50".
func (m Money) String() string { return Format(m.amount, m.currency) }

// Parse reads a money string like "$12.50", "12.50 USD", "USD 1,234.56".
func Parse(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, fmt.Errorf("money: empty string")
	}
	cur := ""
	// trailing/leading 3-letter code
	fields := strings.Fields(s)
	if len(fields) == 2 {
		if isCode(fields[0]) {
			cur, s = fields[0], fields[1]
		} else if isCode(fields[1]) {
			cur, s = fields[1], fields[0]
		}
	}
	// leading symbol
	if cur == "" {
		for code, sym := range symbols {
			if sym != "" && strings.HasPrefix(s, sym) {
				cur = code
				s = strings.TrimPrefix(s, sym)
				break
			}
		}
	}
	if cur == "" {
		cur = "USD"
	}
	cur = normCur(cur)
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	major, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Money{}, fmt.Errorf("money: cannot parse amount %q: %w", s, err)
	}
	return FromMajor(major, cur), nil
}

// MarshalJSON emits {"amount":<minor units>,"currency":"USD"}.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}{m.amount, m.currency})
}

// UnmarshalJSON accepts the object form or a string like "$12.50".
func (m *Money) UnmarshalJSON(b []byte) error {
	var obj struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(b, &obj); err == nil && obj.Currency != "" {
		m.amount, m.currency = obj.Amount, normCur(obj.Currency)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("money: invalid JSON %s", b)
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// Value implements driver.Valuer — stored as "<currency> <minor units>".
func (m Money) Value() (driver.Value, error) {
	return fmt.Sprintf("%s %d", m.currency, m.amount), nil
}

// Scan implements sql.Scanner for the "<currency> <minor units>" form.
func (m *Money) Scan(src any) error {
	if src == nil {
		*m = Money{}
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("money: cannot scan %T", src)
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return fmt.Errorf("money: bad stored value %q", s)
	}
	amt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}
	m.amount, m.currency = amt, normCur(parts[0])
	return nil
}

func normCur(c string) string { return strings.ToUpper(strings.TrimSpace(c)) }

func isCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'z' || (r > 'Z' && r < 'a') {
			return false
		}
	}
	return true
}

func pow10(n int) int64 {
	p := int64(1)
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}
