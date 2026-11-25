package user

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// Approval answers the request RequestID with yes. Issuer signs it. The node
// the Approval touches carries it out: it signs a contract whose ApprovalID is
// this Approval's ObjectID.
type Approval struct {
	Issuer    *astral.Identity
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
// beats every Approval of the same request, even one already carried out.
type Decline struct {
	Issuer    *astral.Identity
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
