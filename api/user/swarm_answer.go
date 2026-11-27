package user

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// Approval answers the request RequestID with yes. Issuer signs it. Only an
// ExpelRequest is answered by an Approval; a request that yields a contract is
// answered by an offer, a contract its issuer signs whose RequestID names the
// request, and the request's filer co-signs it (auth.Contract.RequestID).
//
// An Approval counts whenever it is given: nodes cannot tell when it was
// signed, so, like the Decline rule (D3), its effect needs no clock. Answering
// nodes do not sign answers to an expired request; MayAnswer says who may.
type Approval struct {
	Issuer *astral.Identity
	// RequestID is the entry ID of the stored signed request it answers
	// (*SignedContractRequest, *SignedJoinRequest or *SignedExpelRequest),
	// not the ID of the request body.
	RequestID *astral.ObjectID
}

var _ crypto.SignableTextObject = &Approval{}

func NewApproval(issuer *astral.Identity, requestID *astral.ObjectID) *Approval {
	return &Approval{Issuer: issuer, RequestID: requestID}
}

func (Approval) ObjectType() string { return "mod.user.approval" }

func (a Approval) WriteTo(w io.Writer) (int64, error)     { return astral.Objectify(&a).WriteTo(w) }
func (a *Approval) ReadFrom(src io.Reader) (int64, error) { return astral.Objectify(a).ReadFrom(src) }

func (a Approval) MarshalJSON() ([]byte, error)  { return astral.Objectify(&a).MarshalJSON() }
func (a *Approval) UnmarshalJSON(b []byte) error { return astral.Objectify(a).UnmarshalJSON(b) }

func (a *Approval) SignableHash() []byte { return signableHash(a) }

// SignableText: <issuer> approves request <request id>
func (a *Approval) SignableText() string {
	return fmt.Sprintf("%s approves request %s", a.Issuer.String(), signableObjectID(a.RequestID))
}

func (a *Approval) Signer() *astral.Identity { return a.Issuer }

// SignedApproval is the SwarmState form of an Approval.
type SignedApproval struct {
	*Approval
	IssuerSig *crypto.Signature
}

func (SignedApproval) ObjectType() string { return "mod.user.signed_approval" }

func (a SignedApproval) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&a).WriteTo(w) }
func (a *SignedApproval) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(a).ReadFrom(src)
}

func (a SignedApproval) MarshalJSON() ([]byte, error)  { return astral.Objectify(&a).MarshalJSON() }
func (a *SignedApproval) UnmarshalJSON(b []byte) error { return astral.Objectify(a).UnmarshalJSON(b) }

func (a *SignedApproval) Signature() *crypto.Signature { return a.IssuerSig }

func (a *SignedApproval) IsNil() bool { return a == nil || a.Approval == nil }

// Decline answers the request RequestID with no. Issuer signs it. A Decline
// closes its request: no offer for it is accepted or co-signed afterwards. A
// contract its issuer and subject both signed before keeps its authority; only
// an expel of a party in its chain or its expiry ends it.
type Decline struct {
	Issuer *astral.Identity
	// RequestID is the entry ID of the stored signed request, as in Approval.
	RequestID *astral.ObjectID
}

var _ crypto.SignableTextObject = &Decline{}

func NewDecline(issuer *astral.Identity, requestID *astral.ObjectID) *Decline {
	return &Decline{Issuer: issuer, RequestID: requestID}
}

func (Decline) ObjectType() string { return "mod.user.decline" }

func (d Decline) WriteTo(w io.Writer) (int64, error)     { return astral.Objectify(&d).WriteTo(w) }
func (d *Decline) ReadFrom(src io.Reader) (int64, error) { return astral.Objectify(d).ReadFrom(src) }

func (d Decline) MarshalJSON() ([]byte, error)  { return astral.Objectify(&d).MarshalJSON() }
func (d *Decline) UnmarshalJSON(b []byte) error { return astral.Objectify(d).UnmarshalJSON(b) }

func (d *Decline) SignableHash() []byte { return signableHash(d) }

// SignableText: <issuer> declines request <request id>
func (d *Decline) SignableText() string {
	return fmt.Sprintf("%s declines request %s", d.Issuer.String(), signableObjectID(d.RequestID))
}

func (d *Decline) Signer() *astral.Identity { return d.Issuer }

// SignedDecline is the SwarmState form of a Decline.
type SignedDecline struct {
	*Decline
	IssuerSig *crypto.Signature
}

func (SignedDecline) ObjectType() string { return "mod.user.signed_decline" }

func (d SignedDecline) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&d).WriteTo(w) }
func (d *SignedDecline) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(d).ReadFrom(src)
}

func (d SignedDecline) MarshalJSON() ([]byte, error)  { return astral.Objectify(&d).MarshalJSON() }
func (d *SignedDecline) UnmarshalJSON(b []byte) error { return astral.Objectify(d).UnmarshalJSON(b) }

func (d *SignedDecline) Signature() *crypto.Signature { return d.IssuerSig }

func (d *SignedDecline) IsNil() bool { return d == nil || d.Decline == nil }

// MayAnswer reports whether a node of tier answerer may sign an Approval or a
// Decline of req, a request body or its signed form. Privileged answers every
// request. Trusted answers only a JoinRequest at TierMember (D5). Member
// answers none. The user key needs no tier and may answer any request.
//
// A Privileged node's Approval of a JoinRequest at TierPrivilegedMember does
// not let its carrier sign the membership: only the user key does (D2).
func MayAnswer(answerer Tier, req astral.Object) bool {
	switch answerer {
	case TierPrivilegedMember:
		switch r := req.(type) {
		case *ContractRequest, *ExpelRequest:
			return true
		case *SignedContractRequest:
			return !r.IsNil()
		case *SignedExpelRequest:
			return !r.IsNil()
		case *JoinRequest:
			return r != nil
		case *SignedJoinRequest:
			return !r.IsNil()
		}
	case TierTrusted:
		switch r := req.(type) {
		case *JoinRequest:
			return r != nil && r.Tier == TierMember
		case *SignedJoinRequest:
			return !r.IsNil() && r.Tier == TierMember
		}
	}
	return false
}

// signableObjectID renders id in at most 69 bytes ("data1" and 64 zbase32
// characters); a nil id renders as "none".
func signableObjectID(id *astral.ObjectID) string {
	if id == nil {
		return "none"
	}
	return id.String()
}

func signableHash(obj astral.Object) []byte {
	id, err := astral.ResolveObjectID(obj)
	if err != nil {
		return nil
	}
	return id.Hash[:]
}

func init() {
	astral.MustAdd(&Approval{}, &SignedApproval{}, &Decline{}, &SignedDecline{})
}
