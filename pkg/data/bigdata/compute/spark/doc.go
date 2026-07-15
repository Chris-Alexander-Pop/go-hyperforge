/*
Package spark provides a thin wrapper around local spark-submit / spark-sql CLIs.

This is a development/testing helper that shells out to a local Spark installation
($SPARK_HOME/bin/spark-submit). It is not a Spark Connect or Livy client.

Planned (not implemented here):
  - Apache Spark Connect gRPC client
  - Livy REST job submission
*/
package spark
