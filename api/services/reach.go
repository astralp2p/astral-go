package services

import "fmt"

// Reach selects which nodes a discovery covers.
type Reach string

const (
	// ReachLocal discovers the providers the queried node hosts. It is the
	// default.
	ReachLocal Reach = "local"
	// ReachSwarm also carries the discovery to every member of the node's
	// local swarm, in the caller's name.
	ReachSwarm Reach = "swarm"
)

// ParseReach reads a reach argument. An empty string is ReachLocal.
func ParseReach(s string) (Reach, error) {
	switch Reach(s) {
	case "", ReachLocal:
		return ReachLocal, nil
	case ReachSwarm:
		return ReachSwarm, nil
	}
	return "", fmt.Errorf("unknown reach %q", s)
}
