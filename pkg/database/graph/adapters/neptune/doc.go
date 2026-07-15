/*
Package neptune implements graph.Interface against Amazon Neptune Gremlin HTTP.

Inject a Doer via NewFromClient / WithHTTPClient for tests (httptest) or custom
AWS SigV4 transports. Production New(cfg) builds a default HTTPS client.
*/
package neptune
