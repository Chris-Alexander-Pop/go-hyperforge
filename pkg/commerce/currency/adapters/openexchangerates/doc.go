// Package openexchangerates provides a live FX converter via Open Exchange Rates
// (or the free Frankfurter API when BaseURL / NewFrankfurter is used).
//
// Implements currency.Converter and currency.LiveRateProvider. Optional rate
// caching uses pkg/cache when Config.Cache and CacheTTL are set.
package openexchangerates
