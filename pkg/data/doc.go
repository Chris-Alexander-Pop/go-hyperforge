/*
Package data provides data processing and search capabilities.

This package organizes data functionality into the following subpackages:

  - search: Full-text search (Elasticsearch, OpenSearch, Meilisearch, Algolia, Typesense; memory for tests)
  - bigdata: Large-scale data processing (MapReduce, Spark submit wrapper, OLAP, warehouse adapters)

Planned (not yet packaged at this top level):

  - etl: Extract-Transform-Load pipelines (see bigdata/pipeline/etl for a scaffold)
  - processing: Data transformation utilities

For search operations:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"

For big data processing:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
*/
package data
