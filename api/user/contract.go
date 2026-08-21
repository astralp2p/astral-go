package user

import (
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// IsNodeContract reports whether a contract grants swarm membership.
func IsNodeContract(c *auth.Contract) bool {
	return len(c.HasPermit(SwarmMembershipAction{}.ObjectType())) > 0
}

// NewNodeContract creates a node contract granting swarm membership from issuer to subject.
// A management node also receives the two swarm permits, delegable one hop so the
// node can contract them out to apps it hosts.
func NewNodeContract(issuer, subject *astral.Identity, managementNode bool, duration time.Duration) (*auth.Contract, error) {
	permits := []*auth.Permit{
		{Action: astral.String8(SwarmMembershipAction{}.ObjectType())},
	}
	if managementNode {
		permits = append(permits,
			&auth.Permit{Action: astral.String8(AdminSwarmAction{}.ObjectType()), Delegation: 1},
			&auth.Permit{Action: astral.String8(SeeSwarmAction{}.ObjectType()), Delegation: 1},
		)
	}

	return &auth.Contract{
		Issuer:    issuer,
		Subject:   subject,
		Permits:   permits,
		ExpiresAt: astral.Time(time.Now().Add(duration)),
	}, nil
}
