package apps

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestNewHandlerOnBindsTheNamedAddress covers the reason the constructor exists: a
// node reaches the handler at an address both sides name, so the listener must be
// at the address given and not at one the system assigned.
func TestNewHandlerOnBindsTheNamedAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handler.sock")
	token := astral.NewNonce()

	h, err := NewHandlerOn("unix:"+path, token)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	defer h.Close()

	if got := h.Endpoint(); !strings.HasSuffix(got, path) {
		t.Fatalf("endpoint: got %v, want it to end in %v", got, path)
	}
	if h.Token() != token {
		t.Fatalf("token: got %v, want %v", h.Token(), token)
	}
}

// TestNewHandlerOnRefusesAnAddressWithNoProtocol: "proto:addr" is the whole of the
// address form, and a path alone names no transport.
func TestNewHandlerOnRefusesAnAddressWithNoProtocol(t *testing.T) {
	if _, err := NewHandlerOn("/tmp/handler.sock", astral.NewNonce()); err == nil {
		t.Fatal("want an error for an address carrying no protocol")
	}
}

// TestWithHandlerAddressRefusesAnEmptyAddress: an empty address is the absent
// option, and an option that names nothing is a caller's mistake rather than a
// silent fallback to a system-assigned address.
func TestWithHandlerAddressRefusesAnEmptyAddress(t *testing.T) {
	if _, err := newServeConfig(WithHandlerAddress("")); err == nil {
		t.Fatal("want an error for an empty handler address")
	}
}

// TestNewServeHandlerHonoursTheNamedAddress: the option reaches the listener the
// serve loop reads from, which is the whole path the option exists to travel.
func TestNewServeHandlerHonoursTheNamedAddress(t *testing.T) {
	cfg, err := newServeConfig(WithHandlerAddress("unix:" + filepath.Join(t.TempDir(), "handler.sock")))
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	h, err := newServeHandler(cfg.address)
	if err != nil {
		t.Fatalf("new serve handler: %v", err)
	}
	defer h.Close()

	if !strings.HasPrefix(h.Endpoint(), "unix:") {
		t.Fatalf("endpoint: got %v, want the named unix address", h.Endpoint())
	}
}
