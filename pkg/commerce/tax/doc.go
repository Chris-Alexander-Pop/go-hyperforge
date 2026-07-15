// Package tax provides tax calculation interfaces.
//
// Adapters: memory (multi-jurisdiction rates), taxjar (HTTP taxes API), and
// avalara (AvaTax createTransaction). Amounts use commerce.Money (int64 minor units).
package tax
