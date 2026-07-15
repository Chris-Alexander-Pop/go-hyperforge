// Package warehouse provides an analytics.Sink backed by pkg/data/bigdata.Client.
//
// Events are inserted as SQL rows into a caller-owned table. Compatible with
// BigQuery, Redshift, Snowflake, and other bigdata.Client adapters.
package warehouse
