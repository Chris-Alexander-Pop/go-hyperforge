/*
Package websocket implements a Hub/Client WebSocket fan-out with origin allowlisting,
graceful Shutdown, concurrency-safe broadcast (SmartRWMutex), named rooms
(JoinRoom / LeaveRoom / BroadcastToRoom), and optional upgrade-time Authenticate hook
on Config.
*/
package websocket
