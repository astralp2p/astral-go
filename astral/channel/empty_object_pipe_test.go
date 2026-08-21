package channel

import (
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// The framing layer is where the empty payload actually bites. BinarySender buffers an
// object and emits the buffer as a bytes32, so an object that serialises to nothing
// produces a zero-length payload write — and Ack, EOS and Nil all embed EmptyObject and
// serialise to nothing. Those three are the stream terminators, so a channel over an
// io.Pipe could not send the objects that end a stream.
//
// io.Pipe is not a contrivance here: in-process routing uses it in both directions.

func TestBinaryChannel_EmptyObjectsCrossAPipe(t *testing.T) {
	for _, c := range []struct {
		name string
		obj  astral.Object
	}{
		{"ack", &astral.Ack{}},
		{"eos", &astral.EOS{}},
		{"nil", &astral.Nil{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pr, pw := io.Pipe()
			t.Cleanup(func() { pr.Close(); pw.Close() })

			sent := make(chan error, 1)
			go func() { sent <- NewBinarySender(pw).Send(c.obj) }()

			got := make(chan astral.Object, 1)
			failed := make(chan error, 1)
			go func() {
				o, err := NewBinaryReceiver(pr).Receive()
				if err != nil {
					failed <- err
					return
				}
				got <- o
			}()

			// Only the receive is waited on. Send returning first is normal and says
			// nothing: the defect is the send parking forever, which shows as the
			// receive never arriving.
			select {
			case o := <-got:
				if o.ObjectType() != c.obj.ObjectType() {
					t.Errorf("want %s, got %s", c.obj.ObjectType(), o.ObjectType())
				}
			case err := <-failed:
				t.Fatalf("receive: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatalf("a %s did not cross an io.Pipe in 2s: its zero-length payload "+
					"write parks with no matching read to release it", c.name)
			}

			select {
			case err := <-sent:
				if err != nil {
					t.Errorf("send: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("the %s was received but its send is still parked", c.name)
			}
		})
	}
}
