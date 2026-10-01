package services

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func encode(t *testing.T, objects ...astral.Object) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	s := channel.NewSender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}
	return &buf
}

func decode(t *testing.T, b []byte) []astral.Object {
	t.Helper()
	r := channel.NewReceiver(bytes.NewReader(b))
	var list []astral.Object
	for {
		o, err := r.Receive()
		if errors.Is(err, io.EOF) {
			return list
		}
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		list = append(list, o)
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) bytes() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]byte(nil), l.b.Bytes()...)
}

// The binding answers every ask: the handler's offering with the asked name and
// no provider ID, or unavailable when the handler fails.
func TestBindingAnswersEveryAsk(t *testing.T) {
	caller := astral.GenerateIdentity()
	in := encode(t, &astral.Ack{},
		&services.Ask{RequestID: 1, CallerID: caller, Service: "player"},
		&services.Ask{RequestID: 2, CallerID: caller, Service: "broken"},
	)
	out := &lockedBuffer{}
	h := func(_ *astral.Context, _ *astral.Identity, service string) (*services.Update, error) {
		if service == "broken" {
			return nil, errors.New("boom")
		}
		return &services.Update{Available: true, Name: "wrong", ProviderID: astral.GenerateIdentity()}, nil
	}

	b, err := startBinding(astral.NewContext(nil), channel.New(channel.Join(in, out), channel.WithLockedWrites()), h)
	if err != nil {
		t.Fatal(err)
	}
	<-b.Done()

	answers := map[astral.Nonce]*services.Update{}
	deadline := time.Now().Add(2 * time.Second)
	for len(answers) < 2 && time.Now().Before(deadline) {
		answers = map[astral.Nonce]*services.Update{}
		for _, o := range decode(t, out.bytes()) {
			a := o.(*services.Answer)
			answers[a.RequestID] = a.Update
		}
		time.Sleep(10 * time.Millisecond)
	}
	if u := answers[1]; u == nil || !bool(u.Available) || u.Name != "player" || u.ProviderID != nil {
		t.Fatalf("answer 1: %+v", u)
	}
	if u := answers[2]; u == nil || bool(u.Available) || u.Name != "broken" {
		t.Fatalf("answer 2: %+v", u)
	}
}

func TestBindingRefusalIsReturned(t *testing.T) {
	in := encode(t, astral.Err(services.ErrAdvertised))
	_, err := startBinding(astral.NewContext(nil), channel.New(channel.Join(in, &bytes.Buffer{})), nil)
	if err == nil || !isAdvertised(err) {
		t.Fatalf("got %v; want the advertised refusal", err)
	}
}

func TestEmptyChangeSendsNothing(t *testing.T) {
	out := &lockedBuffer{}
	b := &Binding{ch: channel.New(channel.Join(&bytes.Buffer{}, out)), done: make(chan struct{})}
	if err := b.Change(); err != nil {
		t.Fatal(err)
	}
	if len(out.bytes()) != 0 {
		t.Fatal("an empty Change was sent")
	}
	if err := b.ChangeAll(); err != nil {
		t.Fatal(err)
	}
	if got := decode(t, out.bytes()); len(got) != 1 || !bool(got[0].(*services.Change).All) {
		t.Fatalf("sent %v", got)
	}
}

func readAll(t *testing.T, follow bool, objects ...astral.Object) []Event {
	t.Helper()
	ch := channel.New(channel.Join(encode(t, objects...), &bytes.Buffer{}))
	out := make(chan Event)
	go readDiscovery(astral.NewContext(nil), ch, follow, out)
	var list []Event
	for ev := range out {
		list = append(list, ev)
	}
	return list
}

func TestDiscoveryEvents(t *testing.T) {
	provider := astral.GenerateIdentity()
	u := &services.Update{Available: true, Name: "player", ProviderID: provider}

	complete := readAll(t, false, u, &astral.EOS{})
	if len(complete) != 2 || complete[0].Update == nil || complete[1].Initial == nil || !complete[1].Initial.Complete {
		t.Fatalf("complete one-shot: %+v", complete)
	}

	inc := readAll(t, false, u, &services.Incomplete{Services: []astral.String8{"player"}}, &astral.EOS{})
	if len(inc) != 2 || inc[1].Initial.Complete || inc[1].Initial.Incomplete[0] != "player" {
		t.Fatalf("incomplete one-shot: %+v", inc)
	}

	cut := readAll(t, false, u)
	if len(cut) != 2 || !errors.Is(cut[1].Err, ErrStreamEnded) {
		t.Fatalf("one-shot closed before eos: %+v", cut)
	}

	refused := readAll(t, false, astral.NewError("not permitted"))
	if len(refused) != 1 || refused[0].Err == nil || refused[0].Err.Error() != "not permitted" {
		t.Fatalf("refusal: %+v", refused)
	}

	key := &services.OfferingKey{ProviderID: provider, Name: "player"}
	follow := readAll(t, true, &astral.EOS{}, u, &services.Removed{Offerings: []*services.OfferingKey{key}})
	if len(follow) != 4 || !follow[0].Initial.Complete || follow[1].Update == nil || len(follow[2].Removed) != 1 || !errors.Is(follow[3].Err, ErrStreamEnded) {
		t.Fatalf("follow: %+v", follow)
	}
}

func TestWatcherKeepsTheCurrentSet(t *testing.T) {
	a, b := astral.GenerateIdentity(), astral.GenerateIdentity()
	w := &Watcher{offerings: map[watchKey]*services.Update{}, initial: make(chan InitialOutcome, 1), changed: make(chan struct{}, 1), done: make(chan struct{})}

	w.apply(Event{Update: &services.Update{Available: true, Name: "player", ProviderID: a}})
	w.apply(Event{Update: &services.Update{Available: true, Name: "player", ProviderID: b}})
	w.apply(Event{Initial: &InitialOutcome{Complete: true}})
	if len(w.Offerings()) != 2 {
		t.Fatalf("offerings %d; want 2", len(w.Offerings()))
	}
	if o, ok := <-w.Initial(); !ok || !o.Complete {
		t.Fatal("initial outcome missing")
	}

	w.apply(Event{Update: &services.Update{Available: false, Name: "player", ProviderID: a}})
	w.apply(Event{Removed: []*services.OfferingKey{{ProviderID: b, Name: "player"}}})
	if len(w.Offerings()) != 0 {
		t.Fatalf("offerings %d; want 0", len(w.Offerings()))
	}
	w.apply(Event{Err: ErrStreamEnded})
	if !errors.Is(w.Err(), ErrStreamEnded) {
		t.Fatal("terminal error lost")
	}
}

func TestProviderNamesAndEvaluation(t *testing.T) {
	p := NewProvider(map[string]OfferingFunc{
		"player": func(*astral.Context, *astral.Identity) (*services.Update, error) {
			return &services.Update{Available: true}, nil
		},
		"bitcoin-wallet": nil,
	})
	if got := services.JoinNames(p.Names()); got != "bitcoin-wallet,player" {
		t.Fatalf("names %q", got)
	}
	if u, _ := p.evaluate(nil, nil, "unknown"); u != nil {
		t.Fatal("an unknown service must evaluate to nothing")
	}
	if err := p.Change(astral.GenerateIdentity()); err != nil {
		t.Fatal("Change without a binding must do nothing")
	}
}
