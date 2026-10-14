package auth

import (
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// AuthorizerAssignment names the external authorizer the node asks about one
// action type, after its handlers and the contract chain have not allowed it.
//
// A nil Actor makes the assignment action-wide. An assignment with an Actor
// answers for that actor alone and takes precedence over the action-wide one.
//
// Endpoint selects the transport by scheme: http://, https:// or astral://.
// An http endpoint authenticates with at most one of TokenFile (a file holding
// the token), TokenEnv (an environment variable holding it) or TokenValue (the
// token itself). Timeout bounds one question; zero takes the node's default.
type AuthorizerAssignment struct {
	Action     astral.String8
	Actor      *astral.Identity
	Endpoint   astral.String8
	Timeout    astral.Duration
	TokenFile  astral.String8
	TokenEnv   astral.String8
	TokenValue astral.String8
}

func (AuthorizerAssignment) ObjectType() string { return "mod.auth.authorizer_assignment" }

func (a AuthorizerAssignment) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AuthorizerAssignment) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// Validate reports an assignment that names no action, no endpoint, or more
// than one token source.
func (a *AuthorizerAssignment) Validate() error {
	switch {
	case a.Action == "":
		return errors.New("authorizer assignment: no action named")
	case a.Endpoint == "":
		return errors.New("authorizer assignment: no endpoint named")
	}

	sources := 0
	for _, s := range []astral.String8{a.TokenFile, a.TokenEnv, a.TokenValue} {
		if s != "" {
			sources++
		}
	}
	if sources > 1 {
		return errors.New("authorizer assignment: more than one token source")
	}

	return nil
}

func init() {
	astral.MustAdd(&AuthorizerAssignment{})
}
