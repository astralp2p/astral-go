package auth

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// Contract is the unsigned body of an authorization grant from Issuer to Subject.
// Wrap it in SignedContract before indexing or verifying.
type Contract struct {
	Issuer  *astral.Identity
	Subject *astral.Identity
	Permits []*Permit

	ExpiresAt astral.Time

	// ApprovalID names the Approval (api/user.SignedApproval) this contract
	// carries out. Nil when the contract carries out no Approval, e.g. a
	// membership the user key signs in a ceremony.
	//
	// why last: fields encode in order with no tags, so a new field changes
	// every contract's bytes and ObjectID either way; appending keeps the
	// earlier fields at their offsets. Contracts encoded before this field
	// existed no longer decode (new swarms only, D11).
	ApprovalID *astral.ObjectID
}

type Permit struct {
	Action      astral.String8 // object type of an action
	Constraints *astral.Bundle // list of constraints
	Delegation  astral.Uint8   // hops allowed below a link carrying this permit; 0 = non-delegable
}

var _ astral.Object = &Permit{}
var _ crypto.SignableTextObject = &Contract{}

func (Permit) ObjectType() string { return "mod.auth.permit" }

func (p Permit) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&p).WriteTo(w) }
func (p *Permit) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(p).ReadFrom(r) }

func (p Permit) MarshalJSON() ([]byte, error)  { return astral.Objectify(&p).MarshalJSON() }
func (p *Permit) UnmarshalJSON(b []byte) error { return astral.Objectify(p).UnmarshalJSON(b) }

// Allows reports whether p permits the action: the action type must match and,
// for Constrainable actions, the permit's constraints must pass.
func (p *Permit) Allows(action ActionObject) bool {
	if string(p.Action) != action.ObjectType() {
		return false
	}
	if ca, ok := action.(Constrainable); ok {
		return ca.ApplyConstraints(p.Constraints)
	}
	return true
}

// HasPermit returns all permits in this contract that match the given action type.
// Empty result means the contract grants no such permission.
func (c *Contract) HasPermit(action string) []*Permit {
	if c.Permits == nil {
		return nil
	}
	var result []*Permit
	for _, p := range c.Permits {
		if string(p.Action) == action {
			result = append(result, p)
		}
	}
	return result
}

func (Contract) ObjectType() string { return "mod.auth.contract" }

func (c Contract) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&c).WriteTo(w) }
func (c *Contract) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(c).ReadFrom(r) }

func (c Contract) MarshalJSON() ([]byte, error)  { return astral.Objectify(&c).MarshalJSON() }
func (c *Contract) UnmarshalJSON(b []byte) error { return astral.Objectify(c).UnmarshalJSON(b) }

func (c *Contract) SignableHash() []byte {
	id, err := astral.ResolveObjectID(c)
	if err != nil {
		return nil
	}
	return id.Hash[:]
}

// SignableText is one fixed-shape ASCII line:
//
//	<issuer> grants <subject> [<word> ]permits (<n>) until <YYYY-MM-DD hh:mm:ss>
//
// <word> is the PermitWord of the first permit whose action declares one, e.g.
// the tier of a membership; it is absent otherwise. The commitment the signer
// prefixes binds everything else, ApprovalID included.
func (c *Contract) SignableText() string {
	return contractText(c, c.permitWord(), len(c.Permits))
}

// PermitWorder is implemented by an action type whose permit names itself in a
// contract's signable text, so a signer such as a Coldcard shows it.
//
// A word longer than MaxPermitWordLen or outside [a-z_] is ignored, which keeps
// the line within the signer's limit.
type PermitWorder interface {
	PermitWord() string
}

// MaxPermitWordLen is the longest PermitWord a contract's signable text carries
// (len("privileged_member")).
const MaxPermitWordLen = 17

func (c *Contract) permitWord() string {
	for _, p := range c.Permits {
		if p == nil {
			continue
		}
		w, ok := astral.New(string(p.Action)).(PermitWorder)
		if !ok {
			continue
		}
		if word := w.PermitWord(); validPermitWord(word) {
			return word
		}
	}
	return ""
}

func validPermitWord(w string) bool {
	if w == "" || len(w) > MaxPermitWordLen {
		return false
	}
	for i := 0; i < len(w); i++ {
		if (w[i] < 'a' || w[i] > 'z') && w[i] != '_' {
			return false
		}
	}
	return true
}

// contractText takes the permit count apart from c so a test can render the
// worst case, a count of math.MaxUint32, without building that many permits.
func contractText(c *Contract, word string, permits int) string {
	if word != "" {
		word += " "
	}
	return fmt.Sprintf(
		"%s grants %s %spermits (%d) until %s",
		c.Issuer.String(),
		c.Subject.String(),
		word,
		permits,
		c.ExpiresAt.Time().Format("2006-01-02 15:04:05"),
	)
}

func init() {
	astral.MustAdd(&Contract{})
	astral.MustAdd(&Permit{})
}
