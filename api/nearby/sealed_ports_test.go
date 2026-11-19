package nearby

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// The example of protocols/nearby/types/mod.nearby.sealed_ports.md: the node and the
// hint nonce of the mod.nearby.stealth_hint example, tcp 1800 and kcp 1801.
const (
	vectorNode  = "035d6bfc7818844824354bd5ba1b7c08510139de14aa574e3f53a490814e3eab0e"
	vectorNonce = astral.Nonce(0x7c1a93b50f2e4d18)
	vectorData  = "7773db1b"
)

func vectorPorts() *ListenPorts { return &ListenPorts{TCP: 1800, KCP: 1801} }

func TestListenPortsPayload(t *testing.T) {
	var buf bytes.Buffer
	if _, err := vectorPorts().WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(buf.Bytes()), "07080709"; got != want {
		t.Errorf("payload = %v; want %v (uint16 TCP, uint16 KCP, big-endian)", got, want)
	}
}

func TestSealPortsMatchesTheVector(t *testing.T) {
	node, err := astral.ParseIdentity(vectorNode)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := SealPorts(node, vectorNonce, vectorPorts())
	if err != nil {
		t.Fatalf("SealPorts: %v", err)
	}
	if got := hex.EncodeToString(sealed.Data); got != vectorData {
		t.Errorf("Data = %v; want %v", got, vectorData)
	}

	data, _ := hex.DecodeString(vectorData)
	p, ok := (&SealedPorts{Data: data}).Open(node, vectorNonce)
	if !ok || *p != *vectorPorts() {
		t.Errorf("Open = %+v, %v; want %+v, true", p, ok, vectorPorts())
	}
}

func TestSealPortsRoundTrips(t *testing.T) {
	for _, c := range hintCases() {
		for _, ports := range []ListenPorts{{1800, 1801}, {0, 1792}, {1791, 0}, {65535, 1}} {
			sealed, err := SealPorts(c.node, c.nonce, &ports)
			if err != nil {
				t.Fatalf("%v: SealPorts: %v", c.name, err)
			}
			got, ok := sealed.Open(c.node, c.nonce)
			if !ok || *got != ports {
				t.Errorf("%v: Open = %+v, %v; want %+v, true", c.name, got, ok, ports)
			}
		}
	}
}

// A node other than the sender reads other ports.
func TestSealedPortsOpenWithAnotherNode(t *testing.T) {
	sealed, err := SealPorts(testIdentity(1), vectorNonce, vectorPorts())
	if err != nil {
		t.Fatal(err)
	}

	got, ok := sealed.Open(testIdentity(2), vectorNonce)
	if ok && *got == *vectorPorts() {
		t.Errorf("Open with another node = %+v; want other ports", got)
	}
	got, ok = sealed.Open(testIdentity(1), vectorNonce+1)
	if ok && *got == *vectorPorts() {
		t.Errorf("Open under another nonce = %+v; want other ports", got)
	}
}

// One node sealing the same ports under two nonces writes two ciphertexts.
func TestSealPortsFreshPerNonce(t *testing.T) {
	node := testIdentity(1)
	a, err := SealPorts(node, 1, vectorPorts())
	if err != nil {
		t.Fatal(err)
	}
	b, err := SealPorts(node, 2, vectorPorts())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Data, b.Data) {
		t.Errorf("Data under nonces 1 and 2 = %x; want different", a.Data)
	}
}

// The ports key has its own label, so the keystream differs from the proof wrap's under
// the same node and nonce.
func TestSealPortsKeystreamDiffersFromProofWrap(t *testing.T) {
	node := testIdentity(1)
	sealed, err := SealPorts(node, vectorNonce, &ListenPorts{})
	if err != nil {
		t.Fatal(err)
	}
	wrapped := WrapProofSig(node, vectorNonce, make([]byte, sealedPortsLen))
	if bytes.Equal(sealed.Data, wrapped) {
		t.Errorf("ports keystream = proof keystream = %x", wrapped)
	}
}

func TestSealPortsRefusesAbsentNodes(t *testing.T) {
	for _, node := range []*astral.Identity{nil, {}} {
		if _, err := SealPorts(node, 1, vectorPorts()); err == nil {
			t.Errorf("SealPorts(%v) = nil error; want ErrZeroIdentity", node)
		}
	}
}

func TestSealedPortsOpenRefusesMalformed(t *testing.T) {
	node := testIdentity(1)
	cases := map[string]*SealedPorts{
		"nil":   nil,
		"empty": {},
		"short": {Data: []byte{1, 2, 3}},
		"long":  {Data: []byte{1, 2, 3, 4, 5}},
	}
	for name, s := range cases {
		if p, ok := s.Open(node, 1); ok {
			t.Errorf("%v: Open = %+v, true; want false", name, p)
		}
	}
	if p, ok := (&SealedPorts{Data: []byte{1, 2, 3, 4}}).Open(nil, 1); ok {
		t.Errorf("Open(nil node) = %+v, true; want false", p)
	}
}

func TestPortObjectsRoundTripTheWire(t *testing.T) {
	sealed, err := SealPorts(testIdentity(1), 7, vectorPorts())
	if err != nil {
		t.Fatal(err)
	}
	for _, obj := range []astral.Object{vectorPorts(), sealed} {
		var buf bytes.Buffer
		if _, err := astral.Encode(&buf, obj); err != nil {
			t.Fatalf("Encode %v: %v", obj.ObjectType(), err)
		}
		got, _, err := astral.Decode(&buf)
		if err != nil {
			t.Fatalf("Decode %v: %v", obj.ObjectType(), err)
		}
		if got.ObjectType() != obj.ObjectType() {
			t.Errorf("decoded %v; want %v", got.ObjectType(), obj.ObjectType())
		}
	}
}
