/*
Package swim is an educational sketch of the SWIM gossip membership protocol
for learning and API exploration.

Protocol probes, Suspect/Alive/Dead transitions, event emission, Stop, and
incarnation refute are outlined against an abstract Transport. Full piggyback
gossip dissemination and production hardening are not implemented. Prefer
memberlist (HashiCorp) or similar for real clusters.
*/
package swim
