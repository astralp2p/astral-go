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
// registration. Between registrations a keeper opens a new binding whenever
// the node ends the current one.
//
// Spec: .ai/system/protocols/services/ops/services.advertise.md#liveness
type Provider struct {
	offerings map[string]OfferingFunc
	names     []string
	private   bool
	opening   chan struct{} // holds the one open in progress

	mu      sync.Mutex
	binding *Binding
	keeper  *keeper // the running keeper; nil when none runs
}

// keeper re-advertises after the node ends a binding, until cancel.
type keeper struct {
	cancel func()
}

// Advertise retry: the node can still hold the previous binding right after a
// reconnect, until it notices that binding closed.
const (
	retryFirst = 250 * time.Millisecond
	retryMax   = 4 * time.Second
	retryFor   = 30 * time.Second
)

// Keeper backoff after an advertise that failed for another reason, such as a
// node that is down.
var (
	keepFirst = time.Second
	keepMax   = 30 * time.Second
)

func NewProvider(offerings map[string]OfferingFunc) *Provider {
	p := &Provider{offerings: map[string]OfferingFunc{}, opening: make(chan struct{}, 1)}
	for name, f := range offerings {
		p.offerings[name] = f
		p.names = append(p.names, name)
	}
	sort.Strings(p.names)
	return p
}

// SetOpen sets the visibility of the bindings Serve opens next. A provider is
// open unless set otherwise; a private one is found only by callers holding a
// discovery permit for it.
func (p *Provider) SetOpen(open bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.private = !open
}

// Names are the provider's services, sorted.
func (p *Provider) Names() []string { return append([]string(nil), p.names...) }

// Serve opens a binding unless one is open. It retries a refusal naming an
// already advertised service for up to 30 seconds and returns any other error
// at once. After the first binding opens, one keeper per Provider opens a new
// binding whenever the node ends the current one, until ctx ends or Close.
func (p *Provider) Serve(ctx *astral.Context, client *Client) error {
	if err := p.ensure(ctx, client); err != nil {
		return err
	}
	p.startKeeper(ctx, client)
	return nil
}

// ensure opens a binding unless one is open.
// why: the registrar hook and the keeper may both find no binding after a
// reconnect; one open at a time keeps the second from a refusal loop.
func (p *Provider) ensure(ctx *astral.Context, client *Client) error {
	select {
	case p.opening <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.opening }()
	if p.current() != nil {
		return nil
	}
	return p.open(ctx, client)
}

// open advertises the provider's names, retrying a refusal naming an already
// advertised service for up to 30 seconds.
func (p *Provider) open(ctx *astral.Context, client *Client) error {
	delay := retryFirst
	deadline := time.Now().Add(retryFor)
	for {
		b, err := client.AdvertiseOpen(ctx, p.names, p.isOpen(), p.evaluate)
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

// startKeeper starts the keeper unless one runs.
func (p *Provider) startKeeper(ctx *astral.Context, client *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.keeper != nil {
		return
	}
	kctx, cancel := ctx.WithCancel()
	k := &keeper{cancel: cancel}
	p.keeper = k
	go p.keep(kctx, client, k)
}

// keep waits for the open binding to end and opens a new one, backing off
// 1 s doubling to 30 s while advertising fails, until ctx ends.
// note: a provider that hangs never runs its keeper, so a binding the node
// retired for unanswered asks stays retired.
func (p *Provider) keep(ctx *astral.Context, client *Client, k *keeper) {
	defer p.stopKeeper(k)
	delay := keepFirst
	for {
		if b := p.current(); b != nil {
			select {
			case <-b.Done():
			case <-ctx.Done():
				return
			}
			continue
		}
		if ctx.Err() != nil {
			return
		}
		if err := p.ensure(ctx, client); err == nil {
			delay = keepFirst
			continue
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
		delay = min(delay*2, keepMax)
	}
}

// stopKeeper cancels k and forgets it, unless another keeper replaced it.
func (p *Provider) stopKeeper(k *keeper) {
	k.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.keeper == k {
		p.keeper = nil
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

func (p *Provider) isOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.private
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

// Close stops the keeper and ends the open binding, if any. A later Serve
// opens a binding and starts a keeper again.
func (p *Provider) Close() error {
	p.mu.Lock()
	k := p.keeper
	p.mu.Unlock()
	if k != nil {
		p.stopKeeper(k)
	}
	if b := p.current(); b != nil {
		return b.Close()
	}
	return nil
}
