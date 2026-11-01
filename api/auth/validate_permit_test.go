package auth

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// needsConstraintsAction is a test-local action that requires constraints and
// validates them: it accepts exactly one String8 "good".
type needsConstraintsAction struct {
	Action
}

func (needsConstraintsAction) ObjectType() string { return "test.auth.needs_constraints_action" }

func (a needsConstraintsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *needsConstraintsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

func (a needsConstraintsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return a.ValidateConstraints(cs) == nil
}

func (needsConstraintsAction) ValidateConstraints(cs *astral.Bundle) error {
	if cs == nil || len(cs.Objects()) != 1 {
		return errors.New("want one constraint")
	}
	if s, ok := cs.Objects()[0].(*astral.String8); !ok || *s != "good" {
		return errors.New("want good")
	}
	return nil
}

func init() { astral.MustAdd(&needsConstraintsAction{}) }

func bundleOf(objects ...astral.Object) *astral.Bundle {
	b := astral.NewBundle()
	if err := b.Append(objects...); err != nil {
		panic(err)
	}
	return b
}

func TestValidatePermit(t *testing.T) {
	const needs = "test.auth.needs_constraints_action"

	for _, tc := range []struct {
		name    string
		permit  *Permit
		refused string // empty: passes; otherwise a substring of the refusal
	}{
		{"nil permit", nil, "missing action"},
		{"empty action", &Permit{}, "missing action"},
		{"unknown action", &Permit{Action: "mod.nope.action"}, "mod.nope.action: unknown action"},
		{"unconstrained node action", &Permit{Action: "mod.auth.see_node_state_action"}, ""},
		{"node action with constraints is written as sent", &Permit{Action: "mod.auth.see_node_state_action", Constraints: bundleOf(astral.NewString8("x"))}, ""},
		{"serve objects with constraints", &Permit{Action: "mod.auth.serve_objects_action", Constraints: bundleOf(astral.NewString8("describer"))}, ""},
		{"non-constrainable action with constraints", &Permit{Action: "mod.auth.sudo_action", Constraints: bundleOf(astral.NewString8("x"))}, ""},
		{"required constraints missing", &Permit{Action: needs}, needs + ": requires constraints"},
		{"required constraints empty", &Permit{Action: needs, Constraints: astral.NewBundle()}, needs + ": requires constraints"},
		{"validator refuses", &Permit{Action: needs, Constraints: bundleOf(astral.NewString8("bad"))}, needs + ": want good"},
		{"validator accepts", &Permit{Action: needs, Constraints: bundleOf(astral.NewString8("good"))}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePermit(tc.permit)
			if tc.refused == "" {
				if err != nil {
					t.Fatalf("ValidatePermit: %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidatePermit: nil, want refusal %q", tc.refused)
			}
			if !errors.Is(err, ErrInvalidPermit) {
				t.Errorf("errors.Is(%v, ErrInvalidPermit) = false", err)
			}
			if want := "invalid permit: " + tc.refused; err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
			if !strings.HasPrefix(err.Error(), ErrInvalidPermit.Error()) {
				t.Errorf("error %q lacks prefix %q", err, ErrInvalidPermit)
			}
		})
	}
}
