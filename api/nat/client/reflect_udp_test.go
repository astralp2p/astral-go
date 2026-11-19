package nat

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
)

func TestReflectUDP_ReadsTheOfferThenTheSeenEndpoint(t *testing.T) {
	token := bytes.Repeat([]byte{0x7e}, 16)
	var answer bytes.Buffer
	s := channel.NewSender(&answer)
	for _, o := range []astral.Object{
		&nat.ReflectUDPOffer{Port: 47001, Token: token},
		&nat.Endpoint{IP: mustIP(t, "203.0.113.9"), Port: 6635},
	} {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}

	router := &scriptedRouter{answer: &answer}
	client := New(astral.GenerateIdentity(), astrald.New(router))
	ctx := astral.NewContext(nil)

	sess, err := client.ReflectUDP(ctx)
	if err != nil {
		t.Fatalf("ReflectUDP: %v", err)
	}
	defer sess.Close()

	if sess.Port != 47001 || !bytes.Equal(sess.Token, token) {
		t.Errorf("offer = port %d token % x; want 47001 and % x", sess.Port, sess.Token, token)
	}
	if router.queryString != nat.MethodReflectUDP {
		t.Errorf("query %q; want %q", router.queryString, nat.MethodReflectUDP)
	}

	seen, err := sess.Seen(ctx)
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen.String() != "203.0.113.9:6635" {
		t.Errorf("seen %v; want 203.0.113.9:6635", seen)
	}
}

func TestReflectUDP_ReturnsTheReflectorsError(t *testing.T) {
	var answer bytes.Buffer
	if err := channel.NewSender(&answer).Send(astral.Err(nat.ErrNoReflection)); err != nil {
		t.Fatalf("encode: %v", err)
	}
	client := New(astral.GenerateIdentity(), astrald.New(&scriptedRouter{answer: &answer}))

	if _, err := client.ReflectUDP(astral.NewContext(nil)); err == nil || err.Error() != nat.ErrNoReflection.Error() {
		t.Fatalf("ReflectUDP error = %v; want %q", err, nat.ErrNoReflection)
	}
}
