package channel

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func inputChannel(t *testing.T, objects ...astral.Object) *Channel {
	t.Helper()
	return New(Join(encodeObjects(t, objects...), &bytes.Buffer{}))
}

// TestSwitch_FailOnEOF_ExpectClosedBeforeObject: with FailOnEOF, a channel that
// ends before the expected object makes Switch return io.ErrUnexpectedEOF.
func TestSwitch_FailOnEOF_ExpectClosedBeforeObject(t *testing.T) {
	var got *astral.String8
	err := inputChannel(t).Switch(Expect(&got), PassErrors, FailOnEOF())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

// TestSwitch_FailOnEOF_PeerClosesBeforeObject: a peer that closes its end of
// a live connection before sending is reported as io.ErrUnexpectedEOF.
func TestSwitch_FailOnEOF_PeerClosesBeforeObject(t *testing.T) {
	local, peer := net.Pipe()
	defer local.Close()
	peer.Close()

	var got *astral.String8
	err := New(local).Switch(Expect(&got), PassErrors, FailOnEOF())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

// TestSwitch_FailOnEOF_ExpectAckClosedBeforeAck: FailOnEOF covers any handler
// that stops the Switch, not only Expect.
func TestSwitch_FailOnEOF_ExpectAckClosedBeforeAck(t *testing.T) {
	err := inputChannel(t).Switch(ExpectAck, PassErrors, FailOnEOF())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

// TestSwitch_FailOnEOF_ExpectReceivesObject: an object that arrives before EOF
// is captured and Switch returns nil.
func TestSwitch_FailOnEOF_ExpectReceivesObject(t *testing.T) {
	want := astral.String8("hello")
	var got *astral.String8
	err := inputChannel(t, &want).Switch(Expect(&got), PassErrors, FailOnEOF())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got == nil || *got != want {
		t.Fatalf("got = %v, want %q", got, want)
	}
}

// TestSwitch_FailOnEOF_ErrorObjectWins: a received error object is returned
// as before; FailOnEOF does not replace it.
func TestSwitch_FailOnEOF_ErrorObjectWins(t *testing.T) {
	var got *astral.String8
	err := inputChannel(t, astral.NewError("refused")).Switch(Expect(&got), PassErrors, FailOnEOF())
	if err == nil || errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want the received error object", err)
	}
	if err.Error() != "refused" {
		t.Fatalf("err = %q, want %q", err.Error(), "refused")
	}
}

// TestSwitch_DefaultEOF_ReturnsNil: without FailOnEOF, Switch keeps returning
// nil on EOF and leaves the Expect target unset.
func TestSwitch_DefaultEOF_ReturnsNil(t *testing.T) {
	var got *astral.String8
	err := inputChannel(t).Switch(Expect(&got), PassErrors)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("got = %v, want nil", got)
	}
}

// TestSwitch_DefaultEOF_CollectEndsCleanly: a Switch that runs to EOF without
// FailOnEOF collects every object and returns nil.
func TestSwitch_DefaultEOF_CollectEndsCleanly(t *testing.T) {
	a, b := astral.String8("a"), astral.String8("b")
	var got []*astral.String8
	err := inputChannel(t, &a, &b).Switch(Collect(&got))
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("collected %d objects, want 2", len(got))
	}
}

// TestSwitch_FailOnEOF_CollectWithoutEOS: with FailOnEOF, a collecting Switch
// must end on a handler, such as BreakOnEOS; EOF before EOS is an error, and
// the objects received before it stay collected.
func TestSwitch_FailOnEOF_CollectWithoutEOS(t *testing.T) {
	a := astral.String8("a")
	var got []*astral.String8
	err := inputChannel(t, &a).Switch(Collect(&got), BreakOnEOS, FailOnEOF())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
	if len(got) != 1 {
		t.Fatalf("collected %d objects, want 1", len(got))
	}

	got = nil
	err = inputChannel(t, &a, &astral.EOS{}).Switch(Collect(&got), BreakOnEOS, FailOnEOF())
	if err != nil {
		t.Fatalf("with EOS: err = %v, want nil", err)
	}
}
