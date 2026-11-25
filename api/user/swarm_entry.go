package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// An entry's ID is astral.ResolveObjectID of the signed form exactly as stored,
// signatures included: the key SwarmState stores it under and the ID every
// reference holds (Approval.RequestID, Decline.RequestID,
// auth.Contract.ApprovalID). It differs from the ID of the unsigned body, which
// SignableHash commits to.

// EntryKindOf returns the SwarmState directory an entry belongs in. Only signed
// forms are entries; any other object returns false.
//
//	requests   *SignedContractRequest, *SignedJoinRequest, *SignedExpelRequest
//	approvals  *SignedApproval
//	declines   *SignedDecline
//	contracts  *auth.SignedContract
//	expels     *SignedExpulsion
func EntryKindOf(obj astral.Object) (EntryKind, bool) {
	switch obj.(type) {
	case *SignedContractRequest, *SignedJoinRequest, *SignedExpelRequest:
		return EntryRequests, true
	case *SignedApproval:
		return EntryApprovals, true
	case *SignedDecline:
		return EntryDeclines, true
	case *auth.SignedContract:
		return EntryContracts, true
	case *SignedExpulsion:
		return EntryExpels, true
	}
	return "", false
}
