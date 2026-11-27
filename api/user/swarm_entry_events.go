package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// The events below are the events.Event data the user module emits locally
// when a checked verdict of SwarmState changes. Each is a signal to re-read
// the SwarmView: a handler reads the current status before it acts, and copes
// with duplicates. Nothing is emitted for what is already stored when the
// module starts. Every ID is an entry ID.

// SwarmRequestFiled is emitted once when a request first appears in
// SwarmState. Kind is the object type of the signed request, e.g.
// "mod.user.signed_contract_request".
type SwarmRequestFiled struct {
	RequestID *astral.ObjectID
	Kind      astral.String8
}

var _ astral.Object = &SwarmRequestFiled{}

func (SwarmRequestFiled) ObjectType() string { return "mod.user.swarm_request_filed" }

func (e SwarmRequestFiled) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmRequestFiled) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmRequestFiled) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmRequestFiled) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmRequestApproved is emitted when a request's status becomes
// RequestApproved. ApprovalID is the lowest entry ID among the request's
// counting Approvals; for an ExpelRequest carried out by the root key's
// Expulsion, it is that Expulsion's entry ID.
type SwarmRequestApproved struct {
	RequestID  *astral.ObjectID
	ApprovalID *astral.ObjectID
}

var _ astral.Object = &SwarmRequestApproved{}

func (SwarmRequestApproved) ObjectType() string { return "mod.user.swarm_request_approved" }

func (e SwarmRequestApproved) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmRequestApproved) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmRequestApproved) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmRequestApproved) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmRequestDeclined is emitted when a request's status becomes
// RequestDeclined.
type SwarmRequestDeclined struct {
	RequestID *astral.ObjectID
}

var _ astral.Object = &SwarmRequestDeclined{}

func (SwarmRequestDeclined) ObjectType() string { return "mod.user.swarm_request_declined" }

func (e SwarmRequestDeclined) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmRequestDeclined) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmRequestDeclined) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmRequestDeclined) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmRequestExpired is emitted when a request's status becomes
// RequestExpired.
type SwarmRequestExpired struct {
	RequestID *astral.ObjectID
}

var _ astral.Object = &SwarmRequestExpired{}

func (SwarmRequestExpired) ObjectType() string { return "mod.user.swarm_request_expired" }

func (e SwarmRequestExpired) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmRequestExpired) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmRequestExpired) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmRequestExpired) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmContractAdded is emitted once when a stored contract first counts: it
// has not expired, neither its issuer nor its subject is expelled, and the
// Approval it carries out (ApprovalID, when set) counts for a request that is
// approved. A membership counts when its link holds.
type SwarmContractAdded struct {
	ContractID *astral.ObjectID
	Issuer     *astral.Identity
	Subject    *astral.Identity
	ApprovalID *astral.ObjectID
}

var _ astral.Object = &SwarmContractAdded{}

func (SwarmContractAdded) ObjectType() string { return "mod.user.swarm_contract_added" }

func (e SwarmContractAdded) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmContractAdded) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmContractAdded) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmContractAdded) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

func init() {
	_ = astral.Add(&SwarmRequestFiled{})
	_ = astral.Add(&SwarmRequestApproved{})
	_ = astral.Add(&SwarmRequestDeclined{})
	_ = astral.Add(&SwarmRequestExpired{})
	_ = astral.Add(&SwarmContractAdded{})
}
