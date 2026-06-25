<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/money</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/money"><img src="https://pkg.go.dev/badge/github.com/togo-framework/money.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Currency-safe money type for <a href="https://to-go.dev">togo</a> — integer minor units, no float rounding.</strong></p>
</div>

## Install

```bash
togo install togo-framework/money
```

A money value type (Fowler's Money pattern — like money-rails / django-money). Amounts are **integer minor units** (cents) tagged with an ISO-4217 currency, so you never lose a cent to floating point. Cross-currency arithmetic errors; allocation splits exactly.

## Usage

```go
price := money.New(1999, "USD")            // $19.99
tax   := money.FromMajor(1.60, "USD")      // $1.60
total, _ := price.Add(tax)                 // currency-checked
fmt.Println(total)                         // $21.59

// Split a bill 3 ways with no lost cents:
shares := money.New(1000, "USD").Allocate(1, 1, 1) // 334, 333, 333

// Weighted allocation (70/30):
cut := money.New(1000, "USD").Allocate(70, 30)     // 700, 300

m, _ := money.Parse("€1,234.56")           // parse strings
m.LessThan(total); m.IsZero(); m.Neg()     // compare / negate
```

### Storage & JSON
- **JSON:** `{"amount":1999,"currency":"USD"}` (also unmarshals a string like `"$12.50"`).
- **SQL:** implements `driver.Valuer` + `sql.Scanner` (stored as `"USD 1999"`).

### Formatting & currencies
`String()` renders with the currency symbol, grouping, and correct decimals (USD=2, **JPY=0**, **BHD=3**, BTC=8, …). Register custom currencies/tokens with `money.Register(money.Currency{Code:"TGO", Symbol:"⊤", Decimals:2})`.

### FX conversion
```go
eur, _ := money.Convert(price, "EUR", myRateProvider) // myRateProvider implements Rate(from,to)
```

---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
