package services

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestRemovedCarriesCause(t *testing.T) {
	provider := astral.GenerateIdentity()
	in := &Removed{Offerings: []*OfferingKey{{ProviderID: provider, Name: "player"}}, Cause: astral.Uint8(CauseBindingClosed)}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	var out Removed
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if RemovalCause(out.Cause) != CauseBindingClosed || len(out.Offerings) != 1 {
		t.Fatalf("decoded %+v, want one offering with cause binding_closed", out)
	}

	j, err := in.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(j, []byte(`"Cause":4`)) {
		t.Fatalf("json %s lacks Cause 4", j)
	}
}

func TestRemovalCauseNames(t *testing.T) {
	for c, want := range map[RemovalCause]string{
		CauseMemberLost:    "member_lost",
		CauseMemberLeft:    "member_left",
		CauseMemberRemoved: "member_removed",
		CauseBindingClosed: "binding_closed",
		CauseHidden:        "hidden",
		0:                  "unknown(0)",
		9:                  "unknown(9)",
	} {
		if got := c.String(); got != want {
			t.Errorf("RemovalCause(%d) = %q, want %q", uint8(c), got, want)
		}
	}
	if CauseMemberLost != 1 || CauseMemberLeft != 2 || CauseMemberRemoved != 3 || CauseBindingClosed != 4 || CauseHidden != 5 {
		t.Fatal("cause values differ from services.removed.md")
	}
}
