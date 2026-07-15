// Package currency provides currency conversion and exchange rate services.
//
// The memory adapter keeps a static USD-relative rate table. Live FX is available
// via adapters/openexchangerates (Open Exchange Rates or free Frankfurter), which
// implements Converter and LiveRateProvider with optional pkg/cache.
// Use FormatMoney / commerce.Format for payment display; FormatAmount is a
// float bridge for FX-only display paths.
package currency
