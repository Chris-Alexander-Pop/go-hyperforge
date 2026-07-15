/*
Package swim is an educational sketch of the SWIM gossip membership protocol
for learning and API exploration only.

Protocol probes and suspect/alive transitions are outlined against an abstract
Transport. Full piggyback gossip, dissemination, and production hardening are
not implemented. Prefer memberlist (HashiCorp) or similar for real clusters.
*/
package swim
