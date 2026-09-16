package nodes

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/tor"
	"github.com/astralp2p/astral-go/astral"
)

// The record that broke nodes.new_link: an inbound Tor link has no local onion to
// report, so the LinkInfo carried a zero tor.Endpoint. The digest wrote nothing
// where the reader wanted DigestSize, and every field after it read off by that
// much -- astral-py reported `ShortRead: wanted 105 bytes at offset 131, 32
// available` and the whole answer was unreadable, not just the endpoint.
//
// The error was the fortunate outcome. Decoding this same shape off the unfixed
// encoder returns a LinkInfo that raises nothing and is wrong throughout: the
// remote endpoint reads as absent, Network as "", HighPressure as false and
// BytesThroughput as 0. Asserting the trailing fields, not just that ReadFrom
// succeeds, is what holds that down.
//
// The endpoint fields are interface-typed, so this also covers the type tag: the
// zero endpoint travels as a real mod.tor.endpoint, not as an absent one.
func TestLinkInfoRoundTrip_ZeroTorEndpoint(t *testing.T) {
	local, remote := astral.GenerateIdentity(), astral.GenerateIdentity()

	digest, err := tor.DigestFromString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad.onion")
	if err != nil {
		t.Fatalf("DigestFromString: %v", err)
	}

	want := LinkInfo{
		ID:              astral.NewNonce(),
		LocalIdentity:   local,
		RemoteIdentity:  remote,
		LocalEndpoint:   &tor.Endpoint{},
		RemoteEndpoint:  &tor.Endpoint{Digest: digest, Port: 1791},
		Outbound:        false,
		Network:         "tor",
		HighPressure:    true,
		BytesThroughput: 0x0102030405060708,
	}

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got LinkInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	// the fields after the endpoints are what a shifted frame destroys
	if got.Network != want.Network {
		t.Errorf("Network = %q, want %q", got.Network, want.Network)
	}
	if got.HighPressure != want.HighPressure {
		t.Errorf("HighPressure = %v, want %v", got.HighPressure, want.HighPressure)
	}
	if got.BytesThroughput != want.BytesThroughput {
		t.Errorf("BytesThroughput = %x, want %x", got.BytesThroughput, want.BytesThroughput)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	localEndpoint, ok := got.LocalEndpoint.(*tor.Endpoint)
	if !ok {
		t.Fatalf("LocalEndpoint = %T, want *tor.Endpoint", got.LocalEndpoint)
	}
	if !localEndpoint.IsZero() {
		t.Errorf("LocalEndpoint = %v, want the zero endpoint", localEndpoint.Address())
	}

	remoteEndpoint, ok := got.RemoteEndpoint.(*tor.Endpoint)
	if !ok {
		t.Fatalf("RemoteEndpoint = %T, want *tor.Endpoint", got.RemoteEndpoint)
	}
	if got, want := remoteEndpoint.Address(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad.onion:1791"; got != want {
		t.Errorf("RemoteEndpoint = %q, want %q", got, want)
	}
}
