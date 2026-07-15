/*
Package bigdata provides interfaces for analytical query clients and related helpers.

Features:
  - Generic Client interface for data warehouses (BigQuery, Redshift; Snowflake planned)
  - Compute helpers (in-process MapReduce; local spark-submit wrapper — not Spark Connect)
  - Data formats (Avro, Parquet)
  - Pipeline scaffolds (DAG executor, ETL scaffold under pipeline/etl)

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/data/bigdata/adapters/bigquery"

	client := bigquery.New(cfg)
	res, err := client.Query(ctx, "SELECT * FROM users LIMIT 10")
*/
package bigdata
