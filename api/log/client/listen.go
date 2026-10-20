package log

import (
	"sync"

	"github.com/astralp2p/astral-go/api/log"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	alog "github.com/astralp2p/astral-go/astral/log"

	// why: an Entry carries objects of any type, and decoding one needs its
	// blueprint.
	_ "github.com/astralp2p/astral-go"
)

// Listener is an open log.listen stream.
type Listener struct {
	ch      *channel.Channel
	closeMu sync.Once
}

// Listen subscribes to the target node's live log stream. The stream has no
// server-side terminator; it runs until Close, until ctx is done, or until the
// node disconnects a stalled caller.
//
// The target rejects a caller that does not hold mod.auth.see_node_state_action.
func (c *Client) Listen(ctx *astral.Context) (*Listener, error) {
	// why: an Entry whose objects include a type without a blueprint in this
	// process decodes as an UnparsedObject instead of ending the stream.
	ch, err := c.queryCh(ctx, log.MethodListen, nil, channel.AllowUnparsed(true))
	if err != nil {
		return nil, err
	}

	l := &Listener{ch: ch}

	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	return l, nil
}

// Listen calls the operation on the default client.
func Listen(ctx *astral.Context) (*Listener, error) {
	return Default().Listen(ctx)
}

// Next blocks until the next log entry arrives or the stream ends. It returns
// an *alog.Entry, or an *astral.UnparsedObject of type astrald.log.entry when
// the entry carries an object type without a blueprint in this process.
func (l *Listener) Next() (astral.Object, error) {
	obj, err := l.ch.Receive()
	if err != nil {
		return nil, err
	}

	switch o := obj.(type) {
	case *alog.Entry:
		return o, nil
	case *astral.UnparsedObject:
		if o.ObjectType() == (alog.Entry{}).ObjectType() {
			return o, nil
		}
	case astral.Error:
		return nil, o
	}

	return nil, astral.NewErrUnexpectedObject(obj)
}

// Close ends the subscription.
func (l *Listener) Close() error {
	var err error
	l.closeMu.Do(func() {
		err = l.ch.Close()
	})
	return err
}
