---
name: money
description: Work with money/currency in a togo app using the money plugin — create amounts in minor units, do currency-safe arithmetic, allocate without losing cents, format per currency, and store in JSON/SQL.
---

# togo money

Use the `money` value type for any monetary amount — never use floats for money.

## Create & compute
```go
price := money.New(1999, "USD")      // $19.99 in minor units
total, err := price.Add(tax)          // err if currencies differ
shares := total.Allocate(1,1,1)       // split with no lost cents
```

## Rules
- Store amounts as **minor units** (cents); use `money.Money` columns (Scan/Value) or `{amount,currency}` JSON.
- Never add/compare across currencies without converting (`money.Convert(m, to, rateProvider)`).
- Use `Allocate`/`Split` for divisions (taxes, splits, discounts) so totals reconcile exactly.
- Format for display with `String()`; JPY has 0 decimals, BHD/KWD have 3, BTC has 8 — the registry handles it.

## Custom currency/token
`money.Register(money.Currency{Code:"TGO", Symbol:"⊤", Decimals:2})`.
