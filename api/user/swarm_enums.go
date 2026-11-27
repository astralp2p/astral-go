package user

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// The SwarmState enums below are string8 aliases: the wire carries the word
// itself, so JSON and a signer's display show it as written here. The zero
// value "" is never valid; IsValid reports whether a value is one of the
// declared constants.

// Tier is a swarm member's tier, fixed at join (D8). A membership contract
// carries it as one permit (see NewMembership).
type Tier astral.String8

const (
	TierMember           Tier = "member"            // connectivity and SwarmState sync; adds nobody
	TierTrusted          Tier = "trusted"           // adds Members and approves Member-tier JoinRequests (D5)
	TierPrivilegedMember Tier = "privileged_member" // acts for the user except expel; signed by the user key (D2)
)

// Tiers lists every valid Tier, lowest first.
var Tiers = []Tier{TierMember, TierTrusted, TierPrivilegedMember}

func (t Tier) IsValid() bool {
	return t == TierMember || t == TierTrusted || t == TierPrivilegedMember
}

func (t Tier) String() string { return string(t) }

// RequestStatus is a request's status, derived from SwarmState and never stored.
// Precedence: fulfilled, declined, approved, expired, offered, pending.
//
// why fulfilled first: a Decline closes only its request; a contract both
// parties signed keeps its authority, so the status shows it.
type RequestStatus astral.String8

const (
	RequestPending   RequestStatus = "pending"   // no answer yet and the timeout has not passed
	RequestOffered   RequestStatus = "offered"   // a valid offer exists and the timeout has not passed
	RequestFulfilled RequestStatus = "fulfilled" // a contract with both signatures names the request
	RequestApproved  RequestStatus = "approved"  // an ExpelRequest has a counting Approval, or the user key expelled its Subject, and no Decline
	RequestDeclined  RequestStatus = "declined"  // a Decline exists and no contract fulfils the request
	RequestExpired   RequestStatus = "expired"   // the timeout has passed with no other status
)

// RequestStatuses lists every valid RequestStatus.
var RequestStatuses = []RequestStatus{
	RequestPending, RequestOffered, RequestFulfilled, RequestApproved, RequestDeclined, RequestExpired,
}

func (s RequestStatus) IsValid() bool {
	switch s {
	case RequestPending, RequestOffered, RequestFulfilled, RequestApproved, RequestDeclined, RequestExpired:
		return true
	}
	return false
}

func (s RequestStatus) String() string { return string(s) }

// EntryVerdict is what the merge check decides about an entry offered to
// SwarmState.
type EntryVerdict astral.String8

const (
	EntryAccept EntryVerdict = "accept" // store it and pass it on
	EntryDrop   EntryVerdict = "drop"   // never store it and never pass it on
	EntryWait   EntryVerdict = "wait"   // hold it; check it again after the next merge
)

func (v EntryVerdict) IsValid() bool {
	return v == EntryAccept || v == EntryDrop || v == EntryWait
}

func (v EntryVerdict) String() string { return string(v) }

// EntryKind names a SwarmState directory: swarmstate/<kind>/<ObjectID>.
type EntryKind astral.String8

const (
	EntryRequests  EntryKind = "requests"  // SignedContractRequest, SignedJoinRequest, SignedExpelRequest
	EntryApprovals EntryKind = "approvals" // SignedApproval
	EntryDeclines  EntryKind = "declines"  // SignedDecline
	EntryContracts EntryKind = "contracts" // auth.SignedContract, memberships included
	EntryExpels    EntryKind = "expels"    // SignedExpulsion
)

// EntryKinds lists every valid EntryKind.
var EntryKinds = []EntryKind{EntryRequests, EntryApprovals, EntryDeclines, EntryContracts, EntryExpels}

func (k EntryKind) IsValid() bool {
	switch k {
	case EntryRequests, EntryApprovals, EntryDeclines, EntryContracts, EntryExpels:
		return true
	}
	return false
}

func (k EntryKind) String() string { return string(k) }

// --- codecs: each enum encodes as its string8 word ---

func (*Tier) ObjectType() string                    { return "mod.user.tier" }
func (*Tier) UnderlyingPrimitive() string           { return "string8" }
func (t Tier) WriteTo(w io.Writer) (int64, error)   { return astral.String8(t).WriteTo(w) }
func (t *Tier) ReadFrom(r io.Reader) (int64, error) { return (*astral.String8)(t).ReadFrom(r) }
func (t Tier) MarshalJSON() ([]byte, error)         { return json.Marshal(string(t)) }
func (t *Tier) UnmarshalJSON(b []byte) error        { return json.Unmarshal(b, (*string)(t)) }

func (*RequestStatus) ObjectType() string                  { return "mod.user.request_status" }
func (*RequestStatus) UnderlyingPrimitive() string         { return "string8" }
func (s RequestStatus) WriteTo(w io.Writer) (int64, error) { return astral.String8(s).WriteTo(w) }
func (s *RequestStatus) ReadFrom(r io.Reader) (int64, error) {
	return (*astral.String8)(s).ReadFrom(r)
}
func (s RequestStatus) MarshalJSON() ([]byte, error)  { return json.Marshal(string(s)) }
func (s *RequestStatus) UnmarshalJSON(b []byte) error { return json.Unmarshal(b, (*string)(s)) }

func (*EntryVerdict) ObjectType() string                  { return "mod.user.entry_verdict" }
func (*EntryVerdict) UnderlyingPrimitive() string         { return "string8" }
func (v EntryVerdict) WriteTo(w io.Writer) (int64, error) { return astral.String8(v).WriteTo(w) }
func (v *EntryVerdict) ReadFrom(r io.Reader) (int64, error) {
	return (*astral.String8)(v).ReadFrom(r)
}
func (v EntryVerdict) MarshalJSON() ([]byte, error)  { return json.Marshal(string(v)) }
func (v *EntryVerdict) UnmarshalJSON(b []byte) error { return json.Unmarshal(b, (*string)(v)) }

func (*EntryKind) ObjectType() string                    { return "mod.user.entry_kind" }
func (*EntryKind) UnderlyingPrimitive() string           { return "string8" }
func (k EntryKind) WriteTo(w io.Writer) (int64, error)   { return astral.String8(k).WriteTo(w) }
func (k *EntryKind) ReadFrom(r io.Reader) (int64, error) { return (*astral.String8)(k).ReadFrom(r) }
func (k EntryKind) MarshalJSON() ([]byte, error)         { return json.Marshal(string(k)) }
func (k *EntryKind) UnmarshalJSON(b []byte) error        { return json.Unmarshal(b, (*string)(k)) }

func init() {
	var (
		tier    Tier
		status  RequestStatus
		verdict EntryVerdict
		kind    EntryKind
	)
	astral.MustAdd(&tier, &status, &verdict, &kind)
}
