// Package services contains Hyperforge microservices.
//
// See SERVICE_CATALOG.md for the full service catalog and docs/services.md
// for layout and bootstrap conventions.
//
// Runnable services (v1 memory CRUD unless noted):
//   - Identity: auth, user, gateway, permission
//   - Communication: notification, email, sms
//   - Commerce: product, cart, order, payment, inventory
//   - Platform: appconfig, audit, workflow, discovery, featureflag, secretmanager, ratelimitersvc
//   - AI: llmgateway, agentruntime, toolregistry, contextmanager, embeddingsvc, vectorsearch, promptengine
//   - Observability: metricscollector, logaggregator, tracecollector, alerting
//   - Content: searchsvc, mediasvc
//   - platform helpers: bootstrap, memstore, crudserver
package services
