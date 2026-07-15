// Package currency provides currency conversion and exchange rate services.
//
// The memory adapter keeps a static USD-relative rate table. LiveRateProvider
// is the extension point for Open Exchange Rates / similar feeds (not shipped).
// Use FormatMoney / commerce.Format for payment display; FormatAmount is a
// float bridge for FX-only display paths.
package currency
