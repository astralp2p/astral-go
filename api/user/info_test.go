package user

import (
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestInfo_BinaryRoundTrip(t *testing.T) {
	cases := []Info{
		{State: StateFresh},
		{State: StateUnclaimed, Manager: astral.GenerateIdentity()},
		{State: StateClaimed, NodeAlias: "node", UserAlias: "user"},
	}

	for _, orig := range cases {
		data, err := astral.EncodeBytes(&orig)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		got, err := astral.DecodeAs[*Info](data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if got.State != orig.State || !got.Manager.IsEqual(orig.Manager) ||
			got.NodeAlias != orig.NodeAlias || got.UserAlias != orig.UserAlias {
			t.Fatalf("round trip = %+v; want %+v", got, orig)
		}
	}
}

func TestInfo_JSONNamesStateAndManager(t *testing.T) {
	manager := astral.GenerateIdentity()
	data, err := json.Marshal(astral.Objectify(&Info{State: StateUnclaimed, Manager: manager}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]any
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if fields["State"] != StateUnclaimed || fields["Manager"] != manager.String() {
		t.Fatalf("json = %s; want State %q and Manager %v", data, StateUnclaimed, manager)
	}
}
