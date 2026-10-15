package nodes

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/exonet"
	"github.com/astralp2p/astral-go/api/gateway"
	"github.com/astralp2p/astral-go/api/kcp"
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/api/tcp"
	"github.com/astralp2p/astral-go/api/tor"
	"github.com/astralp2p/astral-go/astral"
)

// A zero NodeInfo is what objects.new hands to the encoder, so it must encode
// rather than panic. The nil Identity travels as an absent presence flag.
func TestNodeInfoWriteTo_NilIdentity(t *testing.T) {
	var buf bytes.Buffer

	n, err := NodeInfo{}.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	// string8 alias (1) + identity presence flag (1) + uint32 endpoint count (4).
	if want := int64(6); n != want {
		t.Errorf("wrote %d bytes, want %d", n, want)
	}
	if got, want := buf.Len(), 6; got != want {
		t.Errorf("buffer holds %d bytes, want %d", got, want)
	}
}

// A nil Identity round-trips as nil, distinct from the zero identity.
func TestNodeInfoRoundTrip_NilIdentity(t *testing.T) {
	var buf bytes.Buffer

	if _, err := (NodeInfo{}).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got NodeInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if got.Identity != nil {
		t.Errorf("Identity = %v, want nil", got.Identity)
	}
	if len(got.Endpoints) != 0 {
		t.Errorf("Endpoints = %v, want none", got.Endpoints)
	}
}

// An identity that is present encodes as its own 33 bytes behind the flag.
func TestNodeInfoWriteTo_KeepsRealIdentity(t *testing.T) {
	id := astral.GenerateIdentity()

	var buf bytes.Buffer
	if _, err := (NodeInfo{Identity: id}).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got NodeInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !got.Identity.IsEqual(id) {
		t.Errorf("Identity = %v, want %v", got.Identity, id)
	}
}

// ReadFrom must not swallow a failure from the identity read. The frame keeps
// bytes after the invalid key, so a swallowed error would let the endpoint
// count read succeed.
func TestNodeInfoReadFrom_InvalidIdentityErrors(t *testing.T) {
	frame := []byte{0x00, 0x01}                              // empty alias, identity present
	frame = append(frame, bytes.Repeat([]byte{0xff}, 33)...) // 33 bytes, not a valid compressed key
	frame = append(frame, 0x00, 0x00, 0x00, 0x00)            // endpoint count, reads cleanly

	var got NodeInfo
	if _, err := got.ReadFrom(bytes.NewReader(frame)); err == nil {
		t.Fatal("ReadFrom accepted an unparseable identity")
	}
}

// Every endpoint type travels under its own object type, including kcp and
// nat, and through the base62 text form as well as the binary one.
func TestNodeInfoRoundTrip_Endpoints(t *testing.T) {
	digest, err := tor.DigestFromString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad.onion")
	if err != nil {
		t.Fatalf("DigestFromString: %v", err)
	}

	want := NodeInfo{
		Alias:    "void0",
		Identity: astral.GenerateIdentity(),
		Endpoints: []exonet.Endpoint{
			mustParse(t, tcp.ParseEndpoint, "1.2.3.4:1791"),
			&tor.Endpoint{Digest: digest, Port: 1791},
			&gateway.Endpoint{GatewayID: astral.GenerateIdentity(), TargetID: astral.GenerateIdentity()},
			mustParse(t, kcp.ParseEndpoint, "5.6.7.8:1792"),
			mustParse(t, nat.ParseEndpoint, "9.10.11.12:1793"),
		},
	}

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	var fromBinary NodeInfo
	if _, err := fromBinary.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("ReadFrom left %d bytes unread", buf.Len())
	}
	assertNodeInfo(t, "binary", fromBinary, want)

	text, err := want.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	var fromText NodeInfo
	if err := fromText.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	assertNodeInfo(t, "text", fromText, want)
}

func mustParse[E exonet.Endpoint](t *testing.T, parse func(string) (E, error), s string) exonet.Endpoint {
	t.Helper()
	e, err := parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return e
}

func assertNodeInfo(t *testing.T, form string, got, want NodeInfo) {
	t.Helper()
	if got.Alias != want.Alias {
		t.Errorf("%s: Alias = %q, want %q", form, got.Alias, want.Alias)
	}
	if !got.Identity.IsEqual(want.Identity) {
		t.Errorf("%s: Identity = %v, want %v", form, got.Identity, want.Identity)
	}
	if len(got.Endpoints) != len(want.Endpoints) {
		t.Fatalf("%s: %d endpoints, want %d", form, len(got.Endpoints), len(want.Endpoints))
	}
	for i, e := range want.Endpoints {
		g := got.Endpoints[i]
		if g.ObjectType() != e.ObjectType() || g.Address() != e.Address() {
			t.Errorf("%s: endpoint %d = %v %v, want %v %v", form, i, g.ObjectType(), g.Address(), e.ObjectType(), e.Address())
		}
	}
}
