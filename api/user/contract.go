package user

import (
	"time"

	"github.com/cryptopunkscc/astral-go/api/auth"
	"github.com/cryptopunkscc/astral-go/astral"
)

// IsNodeContract reports whether a contract grants swarm membership.
func IsNodeContract(c *auth.Contract) bool {
	return len(c.HasPermit(SwarmMembershipAction{}.ObjectType())) > 0
}

// NewNodeContract creates a node contract granting swarm membership from issuer to subject.
func NewNodeContract(issuer, subject *astral.Identity, duration time.Duration) (*auth.Contract, error) {
	permits := []*auth.Permit{
		{Action: astral.String8(SwarmMembershipAction{}.ObjectType())},
	}

	return &auth.Contract{
		Issuer:    issuer,
		Subject:   subject,
		Permits:   permits,
		ExpiresAt: astral.Time(time.Now().Add(duration)),
	}, nil
}
