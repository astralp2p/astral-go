package services

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
)

// OfferingFunc evaluates one service for one caller.
type OfferingFunc func(ctx *astral.Context, caller *astral.Identity) (*services.Update, error)

// Provider is an app's set of services with one evaluator each. Serve keeps
// one binding open for the whole set; lib/apps.WithServices calls it on every
// registration.
type Provider struct {
	offerings map[string]OfferingFunc
	names     []string

	mu      sync.Mutex
	binding *Binding
}

// Advertise retry: the node can still hold the previous binding right after a
// reconnect, until it notices that binding closed.
const (
	retryFirst = 250 * time.Millisecond
	retryMax   = 4 * time.Second
	retryFor   = 30 * time.Second
)

func NewProvider(offerings map[string]OfferingFunc) *Provider {
	p := &Provider{offerings: map[string]OfferingFunc{}}
	for name, f := range offerings {
		p.offerings[name] = f
		p.names = append(p.names, name)
	}
	sort.Strings(p.names)
	return p
}

// Names are the provider's services, sorted.
func (p *Provider) Names() []string { return append([]string(nil), p.names...) }

// Serve opens a binding unless one is open. It retries a refusal naming an
// already advertised service for up to 30 seconds and returns any other error
// at once.
func (p *Provider) Serve(ctx *astral.Context, client *Client) error {
	if p.current() != nil {
		return nil
	}

	delay := retryFirst
	deadline := time.Now().Add(retryFor)
	for {
		b, err := client.Advertise(ctx, p.names, p.evaluate)
		if err == nil {
			p.mu.Lock()
			p.binding = b
			p.mu.Unlock()
			return nil
		}
		if !isAdvertised(err) || time.Now().Add(delay).After(deadline) {
			return err
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
		delay = min(delay*2, retryMax)
	}
}

func isAdvertised(err error) bool {
	return errors.Is(err, services.ErrAdvertised) || err.Error() == services.ErrAdvertised.Error()
}

func (p *Provider) evaluate(ctx *astral.Context, caller *astral.Identity, service string) (*services.Update, error) {
	f := p.offerings[service]
	if f == nil {
		return nil, nil
	}
	return f(ctx, caller)
}

// current returns the open binding, or nil.
func (p *Provider) current() *Binding {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.binding == nil {
		return nil
	}
	select {
	case <-p.binding.Done():
		p.binding = nil
		return nil
	default:
		return p.binding
	}
}

// Change tells the node that callers' offerings may have changed. With no open
// binding, or no callers, it does nothing: the next binding is evaluated from
// scratch.
func (p *Provider) Change(callers ...*astral.Identity) error {
	if b := p.current(); b != nil {
		return b.Change(callers...)
	}
	return nil
}

// ChangeAll tells the node that every following caller's offerings may have
// changed.
func (p *Provider) ChangeAll() error {
	if b := p.current(); b != nil {
		return b.ChangeAll()
	}
	return nil
}

// Close ends the open binding, if any.
func (p *Provider) Close() error {
	if b := p.current(); b != nil {
		return b.Close()
	}
	return nil
}
