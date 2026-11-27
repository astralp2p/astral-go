package user

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// A membership is an auth.Contract that carries exactly one tier permit. There
// is no separate membership wire type: SwarmState stores it as an
// auth.SignedContract under EntryContracts.

// MemberAction, TrustedAction and PrivilegedMemberAction are the tier permits.
// Each names its tier in a contract's signable text (auth.PermitWorder), so a
// Coldcard shows e.g. "privileged_member".
type MemberAction struct{ auth.Action }
type TrustedAction struct{ auth.Action }
type PrivilegedMemberAction struct{ auth.Action }

func (MemberAction) ObjectType() string           { return "mod.user.member_action" }
func (TrustedAction) ObjectType() string          { return "mod.user.trusted_action" }
func (PrivilegedMemberAction) ObjectType() string { return "mod.user.privileged_member_action" }

func (MemberAction) PermitWord() string           { return string(TierMember) }
func (TrustedAction) PermitWord() string          { return string(TierTrusted) }
func (PrivilegedMemberAction) PermitWord() string { return string(TierPrivilegedMember) }

// The tier actions carry no fields beyond auth.Action and evaluate no
// constraints, so a tier permit with constraints is refused (see SeeSwarmAction).

func (a MemberAction) WriteTo(w io.Writer) (int64, error)      { return astral.Objectify(&a).WriteTo(w) }
func (a *MemberAction) ReadFrom(r io.Reader) (int64, error)    { return astral.Objectify(a).ReadFrom(r) }
func (a MemberAction) ApplyConstraints(cs *astral.Bundle) bool { return noConstraints(cs) }

func (a TrustedAction) WriteTo(w io.Writer) (int64, error)      { return astral.Objectify(&a).WriteTo(w) }
func (a *TrustedAction) ReadFrom(r io.Reader) (int64, error)    { return astral.Objectify(a).ReadFrom(r) }
func (a TrustedAction) ApplyConstraints(cs *astral.Bundle) bool { return noConstraints(cs) }

func (a PrivilegedMemberAction) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&a).WriteTo(w)
}
func (a *PrivilegedMemberAction) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(a).ReadFrom(r)
}
func (a PrivilegedMemberAction) ApplyConstraints(cs *astral.Bundle) bool { return noConstraints(cs) }

func noConstraints(cs *astral.Bundle) bool { return cs == nil || len(cs.Objects()) == 0 }

// Action returns the permit action of tier t, or "" for an invalid tier.
func (t Tier) Action() string {
	switch t {
	case TierMember:
		return MemberAction{}.ObjectType()
	case TierTrusted:
		return TrustedAction{}.ObjectType()
	case TierPrivilegedMember:
		return PrivilegedMemberAction{}.ObjectType()
	}
	return ""
}

// NewMembership builds the membership contract by which issuer admits subject
// at tier. approvalID is the Approval the membership carries out; it is nil
// when the user key signs the membership in a ceremony (a Privileged node).
// The contract carries one non-delegable tier permit and expires after duration.
func NewMembership(issuer, subject *astral.Identity, tier Tier, approvalID *astral.ObjectID, duration time.Duration) (*auth.Contract, error) {
	switch {
	case issuer.IsZero() || subject.IsZero():
		return nil, errors.New("membership: issuer and subject are required")
	case !tier.IsValid():
		return nil, fmt.Errorf("membership: invalid tier %q", string(tier))
	case duration <= 0:
		return nil, errors.New("membership: duration must be positive")
	}

	return &auth.Contract{
		Issuer:     issuer,
		Subject:    subject,
		Permits:    []*auth.Permit{{Action: astral.String8(tier.Action())}},
		ExpiresAt:  astral.Time(time.Now().Add(duration)),
		ApprovalID: approvalID,
	}, nil
}

// ContractTier returns the tier a contract grants. It returns false unless the
// contract carries exactly one tier permit.
func ContractTier(c *auth.Contract) (Tier, bool) {
	if c == nil {
		return "", false
	}
	var found Tier
	for _, p := range c.Permits {
		if p == nil {
			continue
		}
		for _, t := range Tiers {
			if string(p.Action) != t.Action() {
				continue
			}
			if found != "" {
				return "", false
			}
			found = t
		}
	}
	return found, found != ""
}

// TierOnly reports whether c holds tier's permit and nothing else: one
// non-delegable, unconstrained permit, as NewMembership builds it.
//
// why: ContractTier ignores permits that name no tier, so a membership could
// carry more than its tier; a check that a membership grants only its tier
// reads this one rule.
func TierOnly(c *auth.Contract, tier Tier) bool {
	if c == nil || len(c.Permits) != 1 || c.Permits[0] == nil {
		return false
	}
	p := c.Permits[0]
	return string(p.Action) == tier.Action() && p.Delegation == 0 &&
		(p.Constraints == nil || len(p.Constraints.Objects()) == 0)
}

// Membership is a view over a signed membership contract and its tier. It is
// not a wire type: its codec methods are the embedded SignedContract's, so it
// encodes as that contract alone.
type Membership struct {
	*auth.SignedContract
	Tier Tier
}

// MembershipOf views c as a membership. It returns false when c is nil or does
// not carry exactly one tier permit. It checks neither signatures, expiry,
// approval links nor expels.
func MembershipOf(c *auth.SignedContract) (*Membership, bool) {
	if c.IsNil() {
		return nil, false
	}
	tier, ok := ContractTier(c.Contract)
	if !ok {
		return nil, false
	}
	return &Membership{SignedContract: c, Tier: tier}, true
}

func init() {
	astral.MustAdd(&MemberAction{}, &TrustedAction{}, &PrivilegedMemberAction{})
}
