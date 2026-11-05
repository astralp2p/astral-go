package nearby

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"

	"github.com/astralp2p/astral-go/api/events"
	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/astral"
)

func samplePeersSeen() map[string]*PeerSeen {
	return map[string]*PeerSeen{
		"via hint": {
			NodeID: testIdentity(1), UserID: testIdentity(2), Via: ViaHint,
			SourceIP: ip.IP(net.IPv4(192, 168, 1, 42).To4()),
		},
		"via profile, no user": {
			NodeID: testIdentity(3), Via: ViaProfile,
			SourceIP: ip.IP(net.IPv4(10, 0, 0, 7).To4()),
		},
	}
}

func assertPeerSeenEqual(t *testing.T, got, want *PeerSeen) {
	t.Helper()
	if !got.NodeID.IsEqual(want.NodeID) || !got.UserID.IsEqual(want.UserID) || got.Via != want.Via ||
		!net.IP(got.SourceIP).Equal(net.IP(want.SourceIP)) {
		t.Errorf("round trip = %+v; want %+v", got, want)
	}
}

func TestPeerSeenBinaryRoundTrip(t *testing.T) {
	for name, orig := range samplePeersSeen() {
		t.Run(name, func(t *testing.T) {
			data, err := astral.EncodeBytes(orig)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			got, err := astral.DecodeAs[*PeerSeen](data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			assertPeerSeenEqual(t, got, orig)
		})
	}
}

func TestPeerSeenJSONRoundTrip(t *testing.T) {
	for name, orig := range samplePeersSeen() {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got PeerSeen
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", data, err)
			}
			assertPeerSeenEqual(t, &got, orig)
		})
	}
}

// The fields sit on the wire in the order NodeID, UserID, Via, SourceIP.
func TestPeerSeenWireLayout(t *testing.T) {
	node := testIdentity(1)
	e := PeerSeen{NodeID: node, Via: ViaHint, SourceIP: ip.IP(net.IPv4(192, 168, 1, 42).To4())}

	var buf bytes.Buffer
	if _, err := e.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	want := bytes.Join([][]byte{
		{1}, node.PublicKey().SerializeCompressed(),
		{0},
		{4, 'h', 'i', 'n', 't'},
		{4, 192, 168, 1, 42},
	}, nil)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("wire = %x; want %x", buf.Bytes(), want)
	}
}

func TestPeerSeenViaValues(t *testing.T) {
	if ViaHint != "hint" || ViaProfile != "profile" {
		t.Errorf("Via values = %q, %q; want \"hint\", \"profile\"", ViaHint, ViaProfile)
	}
}

// PeerSeen travels as events.Event data and decodes back to its own type.
func TestPeerSeenAsEventData(t *testing.T) {
	orig := samplePeersSeen()["via hint"]
	data, err := astral.EncodeBytes(&events.Event{ID: astral.NewNonce(), SourceID: testIdentity(9), Timestamp: astral.Now(), Data: orig})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := astral.DecodeAs[*events.Event](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	seen, ok := got.Data.(*PeerSeen)
	if !ok {
		t.Fatalf("Data = %T; want *PeerSeen", got.Data)
	}
	assertPeerSeenEqual(t, seen, orig)
}
