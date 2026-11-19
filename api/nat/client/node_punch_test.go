package nat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// scriptedRouter answers one query with pre-encoded objects and records what
// the client sent.
type scriptedRouter struct {
	queryString string
	sent        bytes.Buffer
	answer      io.Reader
}

func (r *scriptedRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	r.queryString = q.QueryString.String()
	return query.NewConn(nil, nil, nopWriteCloser{&r.sent}, r.answer, true), nil
}

func (*scriptedRouter) GuestID() *astral.Identity { return astral.GenerateIdentity() }
func (*scriptedRouter) HostID() *astral.Identity  { return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// fakePuncher opens local port 38826 and reports the peer at 45.59.119.13:55001.
type fakePuncher struct{ session []byte }

func (p *fakePuncher) Open() (int, error) { return 38826, nil }
func (p *fakePuncher) Close() error       { return nil }
func (p *fakePuncher) Session() []byte    { return p.session }

func (p *fakePuncher) HolePunch(context.Context, ip.IP, int) (*nat.PunchResult, error) {
	peerIP, _ := ip.ParseIP("45.59.119.13")
	return &nat.PunchResult{LocalPort: 38826, RemoteIP: peerIP, RemotePort: 55001}, nil
}

type fakeReflector struct {
	seen *nat.Endpoint
	err  error
}

func (r fakeReflector) Reflect(*astral.Context) (*nat.Endpoint, error) { return r.seen, r.err }

func mustIP(t *testing.T, s string) ip.IP {
	t.Helper()
	addr, err := ip.ParseIP(s)
	if err != nil {
		t.Fatalf("parse ip %q: %v", s, err)
	}
	return addr
}

// offerOf runs NodePunch against a participant that answers every step, and
// returns the offer the client sent.
func offerOf(t *testing.T, reflector nat.Reflector) *nat.PunchSignal {
	t.Helper()

	session := bytes.Repeat([]byte{0x42}, 16)
	var answer bytes.Buffer
	s := channel.NewSender(&answer)
	for _, sig := range []*nat.PunchSignal{
		{Signal: nat.PunchSignalTypeAnswer, Session: session, IP: mustIP(t, "45.59.119.13"), Port: 55001, BindPort: 55001},
		{Signal: nat.PunchSignalTypeGo, Session: session},
		{Signal: nat.PunchSignalTypeResult, Session: session, IP: mustIP(t, "212.105.131.237"), Port: 6635, BindPort: 55001},
	} {
		if err := s.Send(sig); err != nil {
			t.Fatalf("encode %s: %v", sig.Signal, err)
		}
	}

	router := &scriptedRouter{answer: &answer}
	client := New(astral.GenerateIdentity(), astrald.New(router))

	hole, err := client.NodePunch(astral.NewContext(nil), astral.GenerateIdentity(),
		mustIP(t, "192.0.2.1"), &fakePuncher{session: session}, reflector)
	if err != nil {
		t.Fatalf("NodePunch: %v", err)
	}
	if hole.ActiveBindPort != 38826 {
		t.Errorf("hole active bind port = %d; want 38826", hole.ActiveBindPort)
	}

	sent, err := channel.NewReceiver(bytes.NewReader(router.sent.Bytes())).Receive()
	if err != nil {
		t.Fatalf("decode the first sent object: %v", err)
	}
	offer, ok := sent.(*nat.PunchSignal)
	if !ok || offer.Signal != nat.PunchSignalTypeOffer {
		t.Fatalf("first sent object = %#v; want an offer", sent)
	}
	return offer
}

func TestNodePunch_AnnouncesTheReflectedEndpoint(t *testing.T) {
	offer := offerOf(t, fakeReflector{seen: &nat.Endpoint{IP: mustIP(t, "212.105.131.237"), Port: 6635}})

	if offer.IP.String() != "212.105.131.237" || offer.Port != 6635 {
		t.Errorf("offer advertises %v:%d; want the reflected 212.105.131.237:6635", offer.IP, offer.Port)
	}
	if offer.BindPort != 38826 {
		t.Errorf("offer bind port = %d; want the local port 38826", offer.BindPort)
	}
}

func TestNodePunch_AnnouncesTheLocalPortWithoutAReflection(t *testing.T) {
	tests := []struct {
		name      string
		reflector nat.Reflector
	}{
		{"no reflector", nil},
		{"failing reflector", fakeReflector{err: errors.New("no reflected datagram")}},
		{"IPv6 reflection", fakeReflector{seen: &nat.Endpoint{IP: mustIP(t, "2001:db8::1"), Port: 6635}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offer := offerOf(t, tt.reflector)
			if offer.IP.String() != "192.0.2.1" || offer.Port != 38826 || offer.BindPort != 38826 {
				t.Errorf("offer advertises %v:%d bind %d; want 192.0.2.1:38826 bind 38826",
					offer.IP, offer.Port, offer.BindPort)
			}
		})
	}
}
