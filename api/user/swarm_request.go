package user

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// RequestTimeout is both the default and the maximum time a request stays
// answerable: Timeout is at most FiledAt + RequestTimeout.
const RequestTimeout = time.Hour

// RequestDeadline returns the default Timeout of a request filed at filedAt.
func RequestDeadline(filedAt astral.Time) astral.Time {
	return astral.Time(filedAt.Time().Add(RequestTimeout))
}

// ValidRequestTimeout reports whether timeout is after filedAt and at most
// RequestTimeout after it.
func ValidRequestTimeout(filedAt, timeout astral.Time) bool {
	d := timeout.Time().Sub(filedAt.Time())
	return d > 0 && d <= RequestTimeout
}

// signableTime is the fixed 19-byte time every signable line ends with.
func signableTime(t astral.Time) string { return t.Time().Format("2006-01-02 15:04:05") }

// ContractRequest asks for permits for Identity, e.g. an app FiledBy hosts.
// FiledBy signs it. A Privileged node answers it with an offer or a Decline.
type ContractRequest struct {
	FiledBy  *astral.Identity
	Identity *astral.Identity
	Permits  []*auth.Permit
	FiledAt  astral.Time
	Timeout  astral.Time // at most FiledAt + RequestTimeout
}

var _ crypto.SignableTextObject = &ContractRequest{}

// NewContractRequest returns a request filed now, with the default Timeout.
func NewContractRequest(filedBy, identity *astral.Identity, permits []*auth.Permit) *ContractRequest {
	now := astral.Now()
	return &ContractRequest{
		FiledBy:  filedBy,
		Identity: identity,
		Permits:  permits,
		FiledAt:  now,
		Timeout:  RequestDeadline(now),
	}
}

func (ContractRequest) ObjectType() string { return "mod.user.contract_request" }

func (r ContractRequest) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&r).WriteTo(w) }
func (r *ContractRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r ContractRequest) MarshalJSON() ([]byte, error)  { return astral.Objectify(&r).MarshalJSON() }
func (r *ContractRequest) UnmarshalJSON(b []byte) error { return astral.Objectify(r).UnmarshalJSON(b) }

func (r *ContractRequest) SignableHash() []byte { return signableHash(r) }

// SignableText: <filed by> requests permits (<n>) for <identity> until <timeout>
func (r *ContractRequest) SignableText() string {
	return contractRequestText(r, len(r.Permits))
}

func contractRequestText(r *ContractRequest, permits int) string {
	return fmt.Sprintf("%s requests permits (%d) for %s until %s",
		r.FiledBy.String(), permits, r.Identity.String(), signableTime(r.Timeout))
}

// Signer returns the identity whose signature the request carries.
func (r *ContractRequest) Signer() *astral.Identity { return r.FiledBy }

// ValidTimeout reports whether Timeout is within RequestTimeout of FiledAt.
func (r *ContractRequest) ValidTimeout() bool { return ValidRequestTimeout(r.FiledAt, r.Timeout) }

// Expired reports whether now is past Timeout.
func (r *ContractRequest) Expired(now time.Time) bool { return now.After(r.Timeout.Time()) }

// Covers reports whether c's permits are among those r requests: c has at
// least one permit, and each encodes byte-equal (Action, Constraints and
// Delegation) to one of r.Permits. It checks permits only; who issued c and
// whom it names are the caller's to check. A contract that copies r.Permits
// always covers.
//
// Covers is the rule for the contract that carries a request out, the host's
// contract to the app r names. An approver's offer to the filer carries each
// permit one delegation hop deeper, so Covers refuses every valid offer; an
// offer is checked with CoversOffer.
func (r *ContractRequest) Covers(c *auth.Contract) bool {
	return r.covers(c, 0)
}

// CoversOffer reports whether offer is no wider than r: offer has at least one
// permit, and each has the Action and Constraints of one of r.Permits and a
// Delegation exactly 1 more than that permit's. A permit at Delegation 0, one
// that matches no requested permit, or one at any other depth fails the
// check. Like Covers it checks permits only; that offer names r as its
// RequestID and r's filer as its Subject, and who issued it, are the
// caller's to check.
//
// why: the approver's contract to the host must let the host pass the
// requested permits on to the app in one more hop, and no further; a depth
// equal to the request's would leave the host unable to issue the app's
// contract, and a larger one would let the app delegate beyond what was asked.
func (r *ContractRequest) CoversOffer(offer *auth.Contract) bool {
	return r.covers(offer, 1)
}

// covers checks each of c's permits, with its Delegation lowered by extra,
// byte-equal against r.Permits.
func (r *ContractRequest) covers(c *auth.Contract, extra astral.Uint8) bool {
	if r == nil || c == nil || len(c.Permits) == 0 {
		return false
	}
	requested := make(map[string]bool, len(r.Permits))
	for _, p := range r.Permits {
		if b, ok := permitBytes(p); ok {
			requested[b] = true
		}
	}
	for _, p := range c.Permits {
		if p == nil || p.Delegation < extra {
			return false
		}
		lowered := *p
		lowered.Delegation -= extra
		if b, ok := permitBytes(&lowered); !ok || !requested[b] {
			return false
		}
	}
	return true
}

func permitBytes(p *auth.Permit) (string, bool) {
	if p == nil {
		return "", false
	}
	var buf bytes.Buffer
	if _, err := p.WriteTo(&buf); err != nil {
		return "", false
	}
	return buf.String(), true
}

// SignedContractRequest is the SwarmState form of a ContractRequest.
type SignedContractRequest struct {
	*ContractRequest
	FiledBySig *crypto.Signature
}

func (SignedContractRequest) ObjectType() string { return "mod.user.signed_contract_request" }

func (r SignedContractRequest) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&r).WriteTo(w)
}
func (r *SignedContractRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r SignedContractRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}
func (r *SignedContractRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func (r *SignedContractRequest) Signature() *crypto.Signature { return r.FiledBySig }

// IsNil guards against both a nil receiver and an embedded nil body.
func (r *SignedContractRequest) IsNil() bool { return r == nil || r.ContractRequest == nil }

// JoinRequest asks for Joiner to join the swarm at Tier, carried in by Carrier.
// Joiner signs it with Carrier already named, and resends the same signed object
// to resume a join.
type JoinRequest struct {
	Joiner  *astral.Identity
	Carrier *astral.Identity // a Trusted or Privileged member
	Tier    Tier
	FiledAt astral.Time
	Timeout astral.Time // at most FiledAt + RequestTimeout
}

var _ crypto.SignableTextObject = &JoinRequest{}

// NewJoinRequest returns a request filed now, with the default Timeout.
func NewJoinRequest(joiner, carrier *astral.Identity, tier Tier) *JoinRequest {
	now := astral.Now()
	return &JoinRequest{
		Joiner:  joiner,
		Carrier: carrier,
		Tier:    tier,
		FiledAt: now,
		Timeout: RequestDeadline(now),
	}
}

func (JoinRequest) ObjectType() string { return "mod.user.join_request" }

func (r JoinRequest) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&r).WriteTo(w) }
func (r *JoinRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r JoinRequest) MarshalJSON() ([]byte, error)  { return astral.Objectify(&r).MarshalJSON() }
func (r *JoinRequest) UnmarshalJSON(b []byte) error { return astral.Objectify(r).UnmarshalJSON(b) }

func (r *JoinRequest) SignableHash() []byte { return signableHash(r) }

// SignableText: <joiner> asks <carrier> to join as <tier> until <timeout>.
// A Tier outside Tiers shows as "invalid".
func (r *JoinRequest) SignableText() string {
	return fmt.Sprintf("%s asks %s to join as %s until %s",
		r.Joiner.String(), r.Carrier.String(), signableTier(r.Tier), signableTime(r.Timeout))
}

func signableTier(t Tier) string {
	if !t.IsValid() {
		return "invalid"
	}
	return string(t)
}

func (r *JoinRequest) Signer() *astral.Identity { return r.Joiner }

func (r *JoinRequest) ValidTimeout() bool { return ValidRequestTimeout(r.FiledAt, r.Timeout) }

func (r *JoinRequest) Expired(now time.Time) bool { return now.After(r.Timeout.Time()) }

// SignedJoinRequest is the SwarmState form of a JoinRequest.
type SignedJoinRequest struct {
	*JoinRequest
	JoinerSig *crypto.Signature
}

func (SignedJoinRequest) ObjectType() string { return "mod.user.signed_join_request" }

func (r SignedJoinRequest) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&r).WriteTo(w)
}
func (r *SignedJoinRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r SignedJoinRequest) MarshalJSON() ([]byte, error) { return astral.Objectify(&r).MarshalJSON() }
func (r *SignedJoinRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func (r *SignedJoinRequest) Signature() *crypto.Signature { return r.JoinerSig }

func (r *SignedJoinRequest) IsNil() bool { return r == nil || r.JoinRequest == nil }

// ExpelRequest asks for Subject to be expelled. FiledBy, a Trusted or
// Privileged member, signs it. The user key carries it out with an Expulsion.
type ExpelRequest struct {
	FiledBy *astral.Identity
	Subject *astral.Identity
	FiledAt astral.Time
	Timeout astral.Time // at most FiledAt + RequestTimeout
}

var _ crypto.SignableTextObject = &ExpelRequest{}

// NewExpelRequest returns a request filed now, with the default Timeout.
func NewExpelRequest(filedBy, subject *astral.Identity) *ExpelRequest {
	now := astral.Now()
	return &ExpelRequest{FiledBy: filedBy, Subject: subject, FiledAt: now, Timeout: RequestDeadline(now)}
}

func (ExpelRequest) ObjectType() string { return "mod.user.expel_request" }

func (r ExpelRequest) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&r).WriteTo(w) }
func (r *ExpelRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r ExpelRequest) MarshalJSON() ([]byte, error)  { return astral.Objectify(&r).MarshalJSON() }
func (r *ExpelRequest) UnmarshalJSON(b []byte) error { return astral.Objectify(r).UnmarshalJSON(b) }

func (r *ExpelRequest) SignableHash() []byte { return signableHash(r) }

// SignableText: <filed by> asks to expel <subject> until <timeout>
func (r *ExpelRequest) SignableText() string {
	return fmt.Sprintf("%s asks to expel %s until %s",
		r.FiledBy.String(), r.Subject.String(), signableTime(r.Timeout))
}

func (r *ExpelRequest) Signer() *astral.Identity { return r.FiledBy }

func (r *ExpelRequest) ValidTimeout() bool { return ValidRequestTimeout(r.FiledAt, r.Timeout) }

func (r *ExpelRequest) Expired(now time.Time) bool { return now.After(r.Timeout.Time()) }

// SignedExpelRequest is the SwarmState form of an ExpelRequest.
type SignedExpelRequest struct {
	*ExpelRequest
	FiledBySig *crypto.Signature
}

func (SignedExpelRequest) ObjectType() string { return "mod.user.signed_expel_request" }

func (r SignedExpelRequest) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&r).WriteTo(w)
}
func (r *SignedExpelRequest) ReadFrom(src io.Reader) (int64, error) {
	return astral.Objectify(r).ReadFrom(src)
}

func (r SignedExpelRequest) MarshalJSON() ([]byte, error) { return astral.Objectify(&r).MarshalJSON() }
func (r *SignedExpelRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func (r *SignedExpelRequest) Signature() *crypto.Signature { return r.FiledBySig }

func (r *SignedExpelRequest) IsNil() bool { return r == nil || r.ExpelRequest == nil }

func init() {
	astral.MustAdd(
		&ContractRequest{}, &SignedContractRequest{},
		&JoinRequest{}, &SignedJoinRequest{},
		&ExpelRequest{}, &SignedExpelRequest{},
	)
}
