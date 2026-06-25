package money

import (
	"encoding/json"
	"testing"
)

func TestConstructors(t *testing.T) {
	if m := New(1999, "usd"); m.Amount() != 1999 || m.Currency() != "USD" {
		t.Fatalf("New = %+v", m)
	}
	if m := FromMajor(19.99, "USD"); m.Amount() != 1999 {
		t.Fatalf("FromMajor(19.99) = %d, want 1999", m.Amount())
	}
	if m := FromMajor(1000, "JPY"); m.Amount() != 1000 {
		t.Fatalf("FromMajor(1000 JPY) = %d, want 1000 (0 decimals)", m.Amount())
	}
	if m := FromMajor(1.5, "BHD"); m.Amount() != 1500 {
		t.Fatalf("FromMajor(1.5 BHD) = %d, want 1500 (3 decimals)", m.Amount())
	}
}

func TestAddSubCurrencyMismatch(t *testing.T) {
	a, b := New(1000, "USD"), New(500, "USD")
	if sum, err := a.Add(b); err != nil || sum.Amount() != 1500 {
		t.Fatalf("Add = %+v, %v", sum, err)
	}
	if d, err := a.Sub(b); err != nil || d.Amount() != 500 {
		t.Fatalf("Sub = %+v, %v", d, err)
	}
	if _, err := a.Add(New(1, "EUR")); err == nil {
		t.Fatal("expected currency-mismatch error")
	}
}

func TestMulAndCompare(t *testing.T) {
	if m := New(1000, "USD").Mul(1.5); m.Amount() != 1500 {
		t.Fatalf("Mul = %d", m.Amount())
	}
	if m := New(333, "USD").Mul(3); m.Amount() != 999 {
		t.Fatalf("Mul int = %d", m.Amount())
	}
	a, b := New(100, "USD"), New(200, "USD")
	if !a.LessThan(b) || !b.GreaterThan(a) || a.Equal(b) {
		t.Fatal("comparison wrong")
	}
	if !New(0, "USD").IsZero() || !New(-1, "USD").IsNegative() {
		t.Fatal("zero/negative wrong")
	}
}

func TestAllocateNoLostCents(t *testing.T) {
	parts := New(1000, "USD").Allocate(1, 1, 1) // 10.00 into 3
	if len(parts) != 3 {
		t.Fatalf("got %d parts", len(parts))
	}
	var sum int64
	for _, p := range parts {
		sum += p.Amount()
	}
	if sum != 1000 {
		t.Fatalf("parts sum to %d, want 1000", sum)
	}
	// 1000/3 → 334,333,333 (remainder to the first)
	if parts[0].Amount() != 334 || parts[1].Amount() != 333 || parts[2].Amount() != 333 {
		t.Fatalf("uneven allocation: %d,%d,%d", parts[0].Amount(), parts[1].Amount(), parts[2].Amount())
	}
	// weighted
	w := New(1000, "USD").Allocate(70, 30)
	if w[0].Amount() != 700 || w[1].Amount() != 300 {
		t.Fatalf("weighted = %d,%d", w[0].Amount(), w[1].Amount())
	}
}

func TestSplit(t *testing.T) {
	parts := New(1001, "USD").Split(3)
	var sum int64
	for _, p := range parts {
		sum += p.Amount()
	}
	if sum != 1001 {
		t.Fatalf("split sum %d", sum)
	}
}

func TestFormat(t *testing.T) {
	cases := map[string]Money{
		"$1,234.50": New(123450, "USD"),
		"€1,000.00": New(100000, "EUR"),
		"¥1,000":    New(1000, "JPY"),
		"-$5.00":    New(-500, "USD"),
		"$0.09":     New(9, "USD"),
	}
	for want, m := range cases {
		if got := m.String(); got != want {
			t.Errorf("Format(%d %s) = %q, want %q", m.Amount(), m.Currency(), got, want)
		}
	}
	if got := New(1500, "BHD").String(); got != "BD1.500" {
		t.Errorf("BHD 3-decimal format = %q", got)
	}
}

func TestParse(t *testing.T) {
	cases := map[string]Money{
		"$12.50":       New(1250, "USD"),
		"12.50 USD":    New(1250, "USD"),
		"USD 1,234.56": New(123456, "USD"),
		"€9.99":        New(999, "EUR"),
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || !got.Equal(want) {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	m := New(1999, "USD")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"amount":1999,"currency":"USD"}` {
		t.Fatalf("marshal = %s", b)
	}
	var back Money
	if err := json.Unmarshal(b, &back); err != nil || !back.Equal(m) {
		t.Fatalf("unmarshal = %+v, %v", back, err)
	}
	// string form
	var fromStr Money
	if err := json.Unmarshal([]byte(`"$12.50"`), &fromStr); err != nil || fromStr.Amount() != 1250 {
		t.Fatalf("unmarshal string = %+v, %v", fromStr, err)
	}
}

func TestScanValue(t *testing.T) {
	m := New(1999, "USD")
	v, err := m.Value()
	if err != nil || v.(string) != "USD 1999" {
		t.Fatalf("Value = %v, %v", v, err)
	}
	var back Money
	if err := back.Scan("USD 1999"); err != nil || !back.Equal(m) {
		t.Fatalf("Scan = %+v, %v", back, err)
	}
	if err := back.Scan(nil); err != nil || !back.IsZero() {
		t.Fatalf("Scan nil = %+v, %v", back, err)
	}
}

type fixedRate struct{ rate float64 }

func (f fixedRate) Rate(from, to string) (float64, error) { return f.rate, nil }

func TestConvert(t *testing.T) {
	usd := New(1000, "USD") // $10.00
	eur, err := Convert(usd, "EUR", fixedRate{0.9})
	if err != nil || eur.Amount() != 900 || eur.Currency() != "EUR" {
		t.Fatalf("Convert = %+v, %v", eur, err)
	}
	// same currency is a no-op
	if same, _ := Convert(usd, "USD", fixedRate{2}); !same.Equal(usd) {
		t.Fatal("same-currency convert should be identity")
	}
}
