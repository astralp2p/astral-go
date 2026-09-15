package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Action is the base struct embedded by all typed action objects: mod.auth.action.
//
// why: an embedded field is a Blueprint field, and a Blueprint field names a registered
// type. As a plain struct, Action made every embedding action undescribable, so the
// reflection codec refuses it. Registered, it encodes to the same bytes it did inline.
//
// Action declares no MarshalJSON or UnmarshalJSON. Most embedding actions declare neither,
// so a JSON method here would be promoted into them and encode the embedded fields alone.
type Action struct {
	Nonce   astral.Nonce
	ActorID *astral.Identity
}

func (Action) ObjectType() string { return "mod.auth.action" }

func (a Action) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *Action) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// NewAction returns an Action with a fresh nonce and the given actor.
func NewAction(actor *astral.Identity) Action {
	return Action{Nonce: astral.NewNonce(), ActorID: actor}
}

func (a Action) Id() astral.Nonce              { return a.Nonce }
func (a Action) Actor() *astral.Identity       { return a.ActorID }
func (a *Action) SetActor(id *astral.Identity) { a.ActorID = id }

// ActionObject is the interface satisfied by all action types.
type ActionObject interface {
	astral.Object
	Id() astral.Nonce
	Actor() *astral.Identity
	SetActor(*astral.Identity)
}

// Constrainable is implemented by actions that know how to evaluate permit constraints.
// Actions that do NOT implement this interface are always permitted regardless of constraints.
type Constrainable interface {
	ApplyConstraints(*astral.Bundle) bool
}

func init() { astral.MustAdd(&Action{}) }
