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

// SwarmRequestApproved is emitted when an ExpelRequest's status becomes
// RequestApproved; no other request kind is approved. An ExpelRequest is
// approved by a counting Approval, or by the user key's Expulsion of the
// request's Subject. ApprovalID is the lowest entry ID among the request's
// counting Approvals, or, when none exists, the lowest entry ID among the user
// key's Expulsions whose Subject is the request's Subject.
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

// SwarmOfferAdded is emitted once when a valid offer first appears: a
// contract only its issuer has signed, accepted by the merge check, whose
// request RequestID is live and not declined, and whose issuer is not
// expelled. OfferID is the offer's entry ID. The offer and the contract its
// subject completes from it share the ID of the unsigned contract body. Every
// node emits it for every valid offer; a handler filters by Subject.
type SwarmOfferAdded struct {
	OfferID   *astral.ObjectID
	RequestID *astral.ObjectID
	Issuer    *astral.Identity
	Subject   *astral.Identity
}

var _ astral.Object = &SwarmOfferAdded{}

func (SwarmOfferAdded) ObjectType() string { return "mod.user.swarm_offer_added" }

func (e SwarmOfferAdded) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmOfferAdded) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmOfferAdded) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmOfferAdded) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmContractAdded is emitted once when a stored contract first counts: its
// issuer and its subject both signed it, it has not expired, and neither
// party is expelled. RequestID is the contract's auth.Contract.RequestID, nil
// when it answers no stored request. An offer never fires it.
type SwarmContractAdded struct {
	ContractID *astral.ObjectID
	Issuer     *astral.Identity
	Subject    *astral.Identity
	RequestID  *astral.ObjectID
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
	_ = astral.Add(&SwarmOfferAdded{})
	_ = astral.Add(&SwarmContractAdded{})
}
