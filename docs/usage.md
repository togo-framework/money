# money — usage

## Create
```go
money.New(1999, "USD")        // exact minor units → $19.99
money.FromMajor(19.99, "USD") // from major units (rounded to the currency precision)
money.Parse("€1,234.56")      // parse "$12.50", "12.50 USD", "USD 1,234.56"
```

## Arithmetic (currency-checked)
```go
total, err := a.Add(b)   // err on currency mismatch
d, _ := a.Sub(b)
a.Mul(1.5)               // scale, rounded
a.Neg(); a.Abs()
```

## Allocate without losing cents
```go
money.New(1000,"USD").Allocate(1,1,1) // 334,333,333 (sums to 1000)
money.New(1000,"USD").Allocate(70,30) // 700,300
money.New(1001,"USD").Split(3)        // even split, remainder distributed
```

## Compare
`Equal`, `LessThan`, `GreaterThan`, `Cmp`, `IsZero`, `IsNegative`, `IsPositive`.

## Format & currencies
`String()` / `money.Format(minor, code)` use the currency's symbol + decimals (USD=2, JPY=0, BHD=3, BTC=8). `money.Register(money.Currency{...})` adds custom ones; `money.Decimals(code)` / `money.Symbol(code)`.

## JSON & SQL
- JSON: `{"amount":1999,"currency":"USD"}` (string form `"$12.50"` also unmarshals).
- SQL: `driver.Valuer` + `sql.Scanner` (column stores `"USD 1999"`).

## FX
```go
type RateProvider interface { Rate(from, to string) (float64, error) }
eur, _ := money.Convert(usd, "EUR", provider)
```
