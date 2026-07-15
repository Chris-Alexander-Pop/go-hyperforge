/*
Package websocket implements a Hub/Client WebSocket fan-out with origin allowlisting,
graceful Shutdown, and concurrency-safe broadcast (SmartRWMutex).

Rooms and per-connection auth are not implemented yet; authenticate at the HTTP upgrade
boundary before calling ServeWs.
*/
package websocket
