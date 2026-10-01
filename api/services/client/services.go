package services

import (
	"errors"
	"io"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
)

// ErrStreamEnded reports a discovery stream that closed without a terminal
// error: before the initial boundary, or at any point while following.
var ErrStreamEnded = errors.New("discovery stream ended")

type Client struct {
	astral   *astrald.Client
	targetID *astral.Identity
}

func New(targetID *astral.Identity, astral *astrald.Client) *Client {
	if astral == nil {
		astral = astrald.Default()
	}
	return &Client{astral: astral, targetID: targetID}
}

var defaultClient *Client

func Default() *Client {
	if defaultClient == nil {
		defaultClient = New(nil, astrald.Default())
	}
	return defaultClient
}

// Event is one item of a discovery stream. Exactly one field is set.
type Event struct {
	Update  *services.Update        // one complete offering
	Removed []*services.OfferingKey // shown offerings whose provider was lost
	Initial *InitialOutcome         // the end of the initial attempt, sent once
	Err     error                   // terminal; the channel closes after it
}

// InitialOutcome reports whether the initial attempt completed. Incomplete
// names requested services whose initial work did not resolve.
type InitialOutcome struct {
	Complete   bool
	Incomplete []string
}

// Discover evaluates names for the caller on the target node. Without follow
// the stream ends after the Initial event; with follow it continues until ctx
// ends or the node closes it.
func (client *Client) Discover(ctx *astral.Context, names []string, follow bool) (<-chan Event, error) {
	list := services.JoinNames(names)
	if _, err := services.ParseNames(list); err != nil {
		return nil, err
	}
	ch, err := client.queryCh(ctx, services.MethodDiscover, query.Args{"services": list, "follow": follow})
	if err != nil {
		return nil, err
	}

	out := make(chan Event)
	go readDiscovery(ctx, ch, follow, out)
	return out, nil
}

// Discover runs Discover on the default client.
func Discover(ctx *astral.Context, names []string, follow bool) (<-chan Event, error) {
	return Default().Discover(ctx, names, follow)
}

func readDiscovery(ctx *astral.Context, ch *channel.Channel, follow bool, out chan<- Event) {
	defer close(out)
	defer ch.Close()

	r := discoveryReader{follow: follow}
	err := ch.Collect(func(o astral.Object) error {
		ev, stop, err := r.read(o)
		if err != nil {
			return err
		}
		if ev != nil {
			select {
			case out <- *ev:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if stop {
			return io.EOF
		}
		return nil
	})

	switch {
	case errors.Is(err, io.EOF) && r.boundary && !follow:
		return
	case err == nil:
		err = ErrStreamEnded
	}
	select {
	case out <- Event{Err: err}:
	case <-ctx.Done():
	}
}

type discoveryReader struct {
	follow     bool
	boundary   bool
	incomplete *services.Incomplete
}

// read turns one stream object into an event, and reports whether the stream
// is done.
func (r *discoveryReader) read(o astral.Object) (*Event, bool, error) {
	switch obj := o.(type) {
	case *services.Update:
		return &Event{Update: obj}, false, nil
	case *services.Removed:
		return &Event{Removed: obj.Offerings}, false, nil
	case *services.Incomplete:
		r.incomplete = obj
		return nil, false, nil
	case *astral.EOS:
		r.boundary = true
		return &Event{Initial: outcome(r.incomplete)}, !r.follow, nil
	case *astral.ErrorMessage:
		return nil, false, obj
	default:
		return nil, false, astral.NewErrUnexpectedObject(o)
	}
}

func outcome(inc *services.Incomplete) *InitialOutcome {
	if inc == nil {
		return &InitialOutcome{Complete: true}
	}
	o := &InitialOutcome{}
	for _, s := range inc.Services {
		o.Incomplete = append(o.Incomplete, string(s))
	}
	return o
}

func (client *Client) queryCh(ctx *astral.Context, method string, args any, cfg ...channel.ConfigFunc) (*channel.Channel, error) {
	return client.astral.WithTarget(client.targetID).QueryChannel(ctx, method, args, cfg...)
}
