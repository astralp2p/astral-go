package vfs

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

func bundleOf(t *testing.T, objects ...astral.Object) *astral.Bundle {
	t.Helper()
	b := astral.NewBundle()
	if err := b.Append(objects...); err != nil {
		t.Fatalf("append: %v", err)
	}
	return b
}

// TestScopedActionConstraints covers the read and write actions: both follow
// the same rules.
func TestScopedActionConstraints(t *testing.T) {
	vfsID := astral.GenerateIdentity()
	other := astral.GenerateIdentity()

	for _, kind := range []struct {
		name string
		make func(*astral.Identity) auth.ActionObject
	}{
		{"read", func(id *astral.Identity) auth.ActionObject { return &ReadAction{VFSID: id} }},
		{"write", func(id *astral.Identity) auth.ActionObject { return &WriteAction{VFSID: id} }},
	} {
		action := kind.make(vfsID)
		c := action.(auth.Constrainable)

		for _, tc := range []struct {
			name string
			cs   *astral.Bundle
			want bool
		}{
			{"nil", nil, false},
			{"empty", astral.NewBundle(), false},
			{"listed", bundleOf(t, vfsID), true},
			{"listed among others", bundleOf(t, other, vfsID), true},
			{"not listed", bundleOf(t, other), false},
			{"zero identity member", bundleOf(t, &astral.Identity{}), false},
			{"zero identity member beside listed", bundleOf(t, vfsID, &astral.Identity{}), false},
			{"non-identity member", bundleOf(t, astral.NewError("x")), false},
			{"non-identity member beside listed", bundleOf(t, vfsID, astral.NewError("x")), false},
		} {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				if got := c.ApplyConstraints(tc.cs); got != tc.want {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			})
		}

		t.Run(kind.name+"/zero VFSID matches nothing", func(t *testing.T) {
			zero := kind.make(&astral.Identity{}).(auth.Constrainable)
			if zero.ApplyConstraints(bundleOf(t, vfsID)) || zero.ApplyConstraints(bundleOf(t, &astral.Identity{})) {
				t.Fatal("zero VFSID matched a bundle")
			}
			nilID := kind.make(nil).(auth.Constrainable)
			if nilID.ApplyConstraints(bundleOf(t, vfsID)) {
				t.Fatal("nil VFSID matched a bundle")
			}
		})
	}
}

// TestPermitWithoutConstraintsIsRefused covers the grant path: ValidatePermit
// refuses a read or write permit that carries no constraints.
func TestPermitWithoutConstraintsIsRefused(t *testing.T) {
	for _, action := range []astral.Object{&ReadAction{}, &WriteAction{}} {
		err := auth.ValidatePermit(&auth.Permit{Action: astral.String8(action.ObjectType())})
		if err == nil {
			t.Fatalf("%s: unconstrained permit accepted", action.ObjectType())
		}
	}

	manage := &ManageAction{}
	if err := auth.ValidatePermit(&auth.Permit{Action: astral.String8(manage.ObjectType())}); err != nil {
		t.Fatalf("manage: unconstrained permit refused: %v", err)
	}
}

func TestManageActionRefusesAnyConstraints(t *testing.T) {
	action := &ManageAction{}

	if !action.ApplyConstraints(nil) || !action.ApplyConstraints(astral.NewBundle()) {
		t.Fatal("no constraints must pass")
	}

	vfsID := astral.GenerateIdentity()
	for name, cs := range map[string]*astral.Bundle{
		"identity":  bundleOf(t, vfsID),
		"zero":      bundleOf(t, &astral.Identity{}),
		"non-ident": bundleOf(t, astral.NewError("x")),
	} {
		if action.ApplyConstraints(cs) {
			t.Fatalf("%s: constrained manage permit allowed", name)
		}
		permit := &auth.Permit{Action: astral.String8(action.ObjectType()), Constraints: cs}
		if permit.Allows(action) {
			t.Fatalf("%s: Permit.Allows granted a constrained manage permit", name)
		}
	}
}

func TestPermitAllowsListedVFS(t *testing.T) {
	vfsID := astral.GenerateIdentity()
	action := &ReadAction{VFSID: vfsID}
	permit := &auth.Permit{Action: astral.String8(action.ObjectType()), Constraints: bundleOf(t, vfsID)}

	if !permit.Allows(action) {
		t.Fatal("permit listing the VFS refused")
	}
	if permit.Allows(&ReadAction{VFSID: astral.GenerateIdentity()}) {
		t.Fatal("permit allowed an unlisted VFS")
	}
	if permit.Allows(&WriteAction{VFSID: vfsID}) {
		t.Fatal("read permit allowed write")
	}
}

func TestActionsRoundTrip(t *testing.T) {
	vfsID := astral.GenerateIdentity()
	actor := astral.GenerateIdentity()

	var buf bytes.Buffer
	in := &ReadAction{Action: auth.NewAction(actor), VFSID: vfsID}
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var out ReadAction
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if !out.VFSID.IsEqual(vfsID) || !out.ActorID.IsEqual(actor) || out.Nonce != in.Nonce {
		t.Fatalf("round trip changed the action: %+v", out)
	}
}

func TestErrorsText(t *testing.T) {
	want := map[string]error{
		"not permitted":    ErrNotPermitted,
		"no such vfs":      ErrNoSuchVFS,
		"invalid path":     ErrInvalidPath,
		"not found":        ErrNotFound,
		"not a folder":     ErrNotAFolder,
		"is a folder":      ErrIsAFolder,
		"already exists":   ErrAlreadyExists,
		"folder not empty": ErrFolderNotEmpty,
		"quota exceeded":   ErrQuotaExceeded,
		"no inodes left":   ErrNoInodesLeft,
		"invalid time":     ErrInvalidTime,
		"no storage":       ErrNoStorage,
	}
	if len(want) != 12 {
		t.Fatalf("%d error values, want 12", len(want))
	}
	for text, err := range want {
		if err.Error() != text {
			t.Errorf("got %q, want %q", err.Error(), text)
		}
	}
}
