package nearby

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func samplePublicProfiles() map[string]*PublicProfile {
	return map[string]*PublicProfile{
		"owned":    {NodeID: astral.GenerateIdentity(), UserID: astral.GenerateIdentity(), NodeAlias: "alice-laptop"},
		"unowned":  {NodeID: astral.GenerateIdentity(), NodeAlias: "new-node"},
		"no alias": {NodeID: astral.GenerateIdentity(), UserID: astral.GenerateIdentity()},
		"zero":     {},
	}
}

func assertProfileEqual(t *testing.T, got, want *PublicProfile) {
	t.Helper()
	if !got.NodeID.IsEqual(want.NodeID) {
		t.Errorf("NodeID = %v; want %v", got.NodeID, want.NodeID)
	}
	if !got.UserID.IsEqual(want.UserID) {
		t.Errorf("UserID = %v; want %v", got.UserID, want.UserID)
	}
	if got.NodeAlias != want.NodeAlias {
		t.Errorf("NodeAlias = %q; want %q", got.NodeAlias, want.NodeAlias)
	}
}

func TestPublicProfileBinaryRoundTrip(t *testing.T) {
	for name, orig := range samplePublicProfiles() {
		t.Run(name, func(t *testing.T) {
			data, err := astral.EncodeBytes(orig)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			got, err := astral.DecodeAs[*PublicProfile](data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			assertProfileEqual(t, got, orig)
		})
	}
}

func TestPublicProfileJSONRoundTrip(t *testing.T) {
	for name, orig := range samplePublicProfiles() {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got PublicProfile
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", data, err)
			}
			assertProfileEqual(t, &got, orig)
		})
	}
}

// The fields sit on the wire in the order NodeID, UserID, NodeAlias, and an absent
// UserID is the single presence byte 0.
func TestPublicProfileWireLayout(t *testing.T) {
	node, user := astral.GenerateIdentity(), astral.GenerateIdentity()
	nodeKey := node.PublicKey().SerializeCompressed()
	userKey := user.PublicKey().SerializeCompressed()

	tests := []struct {
		name    string
		profile PublicProfile
		want    []byte
	}{
		{
			"owned",
			PublicProfile{NodeID: node, UserID: user, NodeAlias: "ab"},
			bytes.Join([][]byte{{1}, nodeKey, {1}, userKey, {2, 'a', 'b'}}, nil),
		},
		{
			"unowned",
			PublicProfile{NodeID: node, NodeAlias: "ab"},
			bytes.Join([][]byte{{1}, nodeKey, {0}, {2, 'a', 'b'}}, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tt.profile.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("wire = %x; want %x", buf.Bytes(), tt.want)
			}
		})
	}
}

// A present identity of 33 zero bytes decodes as a non-nil zero identity, and an
// absent one as nil. IsZero reports both.
func TestPublicProfileDecodesAZeroIdentity(t *testing.T) {
	zero := append([]byte{1}, make([]byte, 33)...)
	tests := []struct {
		name             string
		wire             []byte
		nodeNil, userNil bool
	}{
		{"zero node, no user", bytes.Join([][]byte{zero, {0}, {0}}, nil), false, true},
		{"no node, zero user", bytes.Join([][]byte{{0}, zero, {0}}, nil), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got PublicProfile
			n, err := got.ReadFrom(bytes.NewReader(tt.wire))
			if err != nil {
				t.Fatalf("ReadFrom(%x): %v", tt.wire, err)
			}
			if n != int64(len(tt.wire)) {
				t.Errorf("ReadFrom reported %d bytes; wire holds %d", n, len(tt.wire))
			}
			if (got.NodeID == nil) != tt.nodeNil || (got.UserID == nil) != tt.userNil {
				t.Errorf("NodeID nil = %v, UserID nil = %v; want %v, %v",
					got.NodeID == nil, got.UserID == nil, tt.nodeNil, tt.userNil)
			}
			if !got.NodeID.IsZero() || !got.UserID.IsZero() {
				t.Errorf("NodeID.IsZero = %v, UserID.IsZero = %v; want true, true",
					got.NodeID.IsZero(), got.UserID.IsZero())
			}

			var again bytes.Buffer
			if _, err := got.WriteTo(&again); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			if !bytes.Equal(again.Bytes(), tt.wire) {
				t.Errorf("re-encoded = %x; want %x", again.Bytes(), tt.wire)
			}
		})
	}
}
