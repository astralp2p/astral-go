package user

import "github.com/astralp2p/astral-go/api/auth"

// IsNodeContract reports whether a contract grants swarm membership.
func IsNodeContract(c *auth.Contract) bool {
	return len(c.HasPermit(SwarmMembershipAction{}.ObjectType())) > 0
}
