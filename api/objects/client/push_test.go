package objects

import (
	"errors"
	"testing"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

// A node that receives the object and answers false refused it: Push says so
// with ErrPushRejected, which a caller tells apart from a push that never got
// through.
func TestPushAnswersTheNodesVerdict(t *testing.T) {
	for _, accept := range []bool{true, false} {
		c := clientFor(t, objects.MethodPush, func(_ *astral.Context, q *routing.IncomingQuery) error {
			ch := q.Accept()
			defer ch.Close()
			if _, err := ch.Receive(); err != nil {
				return err
			}
			return ch.Send(astral.NewBool(accept))
		})
		err := c.Push(astral.NewContext(nil), astral.NewString8("object"))
		switch {
		case accept && err != nil:
			t.Fatalf("accepted push: err = %v, want nil", err)
		case !accept && !errors.Is(err, objects.ErrPushRejected):
			t.Fatalf("refused push: err = %v, want ErrPushRejected", err)
		}
	}
}
