/*
Package data provides data processing and search capabilities.

This package organizes data functionality into the following subpackages:

  - search: Full-text search (Elasticsearch, Meilisearch, Algolia; memory for tests)
  - bigdata: Large-scale data processing (MapReduce, Spark submit wrapper, OLAP, warehouse adapters)

Planned (not yet packaged at this top level):

  - etl: Extract-Transform-Load pipelines (see bigdata/pipeline/etl for a scaffold)
  - processing: Data transformation utilities

Typesense and OpenSearch adapters are planned; OpenSearch is API-compatible with the
Elasticsearch adapter in many deployments but has no dedicated adapter yet.

For search operations:

	import "github.com/chris-alexander-pop/system-design-library/pkg/data/search"

For big data processing:

	import "github.com/chris-alexander-pop/system-design-library/pkg/data/bigdata"
*/
package data
