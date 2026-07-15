/*
Package chord is an educational sketch of the Chord DHT for learning and API
exploration.

Node helpers implement Create/Join/Stabilize/Notify against a Transport
(InProcessTransport for in-process rings). Finger-table repair and persistence
are incomplete. This is not a production DHT — prefer a mature library or
managed discovery/KV for real deployments.
*/
package chord
