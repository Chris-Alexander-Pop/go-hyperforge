// Package tax provides tax calculation interfaces.
//
// A TaxJar (and optionally Avalara) adapter is planned but not yet shipped.
// The memory adapter models multi-jurisdiction rates (country/state) so callers
// can structure for jurisdiction-aware calculation today. Amounts use
// commerce.Money (int64 minor units).
package tax
