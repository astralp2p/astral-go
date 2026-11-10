package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// PrivateListPath is the tree path of the node's private list.
const PrivateListPath = "/mod/services/private"

// PrivateEntry covers the bindings of ProviderID that advertise Service. A nil
// or zero ProviderID covers every provider of Service on the node.
type PrivateEntry struct {
	Service    astral.String8
	ProviderID *astral.Identity
}

func (PrivateEntry) ObjectType() string { return "mod.services.private_entry" }

func (e PrivateEntry) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *PrivateEntry) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e PrivateEntry) MarshalJSON() ([]byte, error)  { return astral.Objectify(&e).MarshalJSON() }
func (e *PrivateEntry) UnmarshalJSON(b []byte) error { return astral.Objectify(e).UnmarshalJSON(b) }

// Covers reports whether the entry covers provider's binding of service.
func (e *PrivateEntry) Covers(provider *astral.Identity, service string) bool {
	if string(e.Service) != service {
		return false
	}
	return e.ProviderID == nil || e.ProviderID.IsZero() || e.ProviderID.IsEqual(provider)
}

// PrivateList is the node's list of providers the user made private. An
// entry only makes a binding private; nothing in the list makes one open.
type PrivateList struct {
	Entries []*PrivateEntry
}

func (PrivateList) ObjectType() string { return "mod.services.private_list" }

func (l PrivateList) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&l).WriteTo(w)
}

func (l *PrivateList) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(l).ReadFrom(r)
}

func (l PrivateList) MarshalJSON() ([]byte, error)  { return astral.Objectify(&l).MarshalJSON() }
func (l *PrivateList) UnmarshalJSON(b []byte) error { return astral.Objectify(l).UnmarshalJSON(b) }

// Covers reports whether any entry covers provider's binding of service. A nil
// list covers nothing.
func (l *PrivateList) Covers(provider *astral.Identity, service string) bool {
	if l == nil {
		return false
	}
	for _, e := range l.Entries {
		if e != nil && e.Covers(provider, service) {
			return true
		}
	}
	return false
}

func init() {
	astral.MustAdd(&PrivateEntry{})
	astral.MustAdd(&PrivateList{})
}
