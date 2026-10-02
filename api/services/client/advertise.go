package services

import (
	"log"
	"sync"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Handler evaluates one service for one caller. A nil update or an error is
// answered as unavailable; an ask is never left unanswered.
type Handler func(ctx *astral.Context, caller *astral.Identity, service string) (*services.Update, error)

// Binding is an open advertisement: the node asks, the handler answers, and
// the provider announces changes. The service set is fixed for its lifetime.
type Binding struct {
	ch      *channel.Channel
	handler Handler
	done    chan struct{}
	once    sync.Once
}

// Advertise opens a binding for names on the target node. It returns once the
// node acknowledges the binding, or with the node's refusal.
func (client *Client) Advertise(ctx *astral.Context, names []string, h Handler) (*Binding, error) {
	list := services.JoinNames(names)
	if _, err := services.ParseNames(list); err != nil {
		return nil, err
	}

	// why: answers run on per-ask goroutines while Change runs on the
	// caller's, so writes must not interleave.
	ch, err := client.queryCh(ctx, services.MethodAdvertise, query.Args{"services": list}, channel.WithLockedWrites())
	if err != nil {
		return nil, err
	}
	return startBinding(ctx, ch, h)
}

// Advertise opens a binding on the default node.
func Advertise(ctx *astral.Context, names []string, h Handler) (*Binding, error) {
	return Default().Advertise(ctx, names, h)
}

func startBinding(ctx *astral.Context, ch *channel.Channel, h Handler) (*Binding, error) {
	if err := ch.Switch(channel.ExpectAck, channel.PassErrors); err != nil {
		ch.Close()
		return nil, err
	}

	b := &Binding{ch: ch, handler: h, done: make(chan struct{})}
	go b.serve(ctx)
	return b, nil
}

func (b *Binding) serve(ctx *astral.Context) {
	defer close(b.done)
	defer b.closeChannel()

	_ = b.ch.Handle(ctx, func(o astral.Object) {
		ask, ok := o.(*services.Ask)
		if !ok {
			b.ch.Close()
			return
		}
		// why: one goroutine per ask. The node already sends one ask per
		// caller at a time, so a slow caller never delays another.
		go b.answer(ctx, ask)
	})
}

func (b *Binding) answer(ctx *astral.Context, ask *services.Ask) {
	u, err := b.handler(ctx, ask.CallerID, string(ask.Service))
	if err != nil {
		log.Printf("services: evaluating %v for %v: %v", ask.Service, ask.CallerID, err)
	}
	reply := services.Update{}
	if err == nil && u != nil {
		reply = *u
	}
	reply.Name = ask.Service
	reply.ProviderID = nil

	_ = b.ch.Send(&services.Answer{RequestID: ask.RequestID, Update: &reply})
}

// Change tells the node that the callers' offerings may have changed. An empty
// list sends nothing: the node fails a binding on a change selecting no caller.
func (b *Binding) Change(callers ...*astral.Identity) error {
	if len(callers) == 0 {
		return nil
	}
	return b.ch.Send(&services.Change{Callers: callers})
}

// ChangeAll tells the node that every following caller's offerings may have
// changed.
func (b *Binding) ChangeAll() error {
	return b.ch.Send(&services.Change{All: true})
}

// Close ends the binding; the node releases its services.
func (b *Binding) Close() error {
	b.closeChannel()
	<-b.done
	return nil
}

// Done is closed when the binding ends, by Close or by the node.
func (b *Binding) Done() <-chan struct{} { return b.done }

func (b *Binding) closeChannel() {
	b.once.Do(func() { b.ch.Close() })
}
