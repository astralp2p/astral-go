package inference

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/routing"
)

// provider answers inference.complete: it reads the request, hands it to
// answer, and sends what answer returns.
func provider(t *testing.T, got chan<- *inference.Request, answer ...astral.Object) *Client {
	t.Helper()
	c, _ := clientFor(t, map[string]any{inference.OpComplete: func(_ *astral.Context, q *routing.IncomingQuery) error {
		ch := q.Accept()
		defer ch.Close()
		var req *inference.Request
		if err := ch.Switch(channel.Expect(&req), channel.FailOnEOF()); err != nil {
			return err
		}
		if got != nil {
			got <- req
		}
		for _, o := range answer {
			if err := ch.Send(o); err != nil {
				return err
			}
		}
		return nil
	}})
	return c
}

// collect ranges over an answer and returns its events and its error.
func collect(seq func(func(Event, error) bool)) (events []Event, err error) {
	for ev, e := range seq {
		if e != nil {
			return events, e
		}
		events = append(events, ev)
	}
	return events, nil
}

func TestCompleteStreamsChunksAndUsage(t *testing.T) {
	got := make(chan *inference.Request, 1)
	c := provider(t, got,
		&inference.Chunk{Index: 0, Reasoning: "think", Model: "m"},
		&inference.Chunk{Index: 1, Text: "calling", ToolCalls: []*inference.ToolCallDelta{{Index: 0, ID: "c0", Name: "ls", Arguments: `{"pa`}}},
		&inference.Chunk{Index: 2, ToolCalls: []*inference.ToolCallDelta{{Index: 0, Arguments: `th":"/"}`}}, FinishReason: inference.FinishToolCalls},
		&inference.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, CachedPromptTokens: 8, ReasoningTokens: 2},
		&astral.EOS{},
	)
	req := &inference.Request{Model: "m", Stream: true, ToolChoice: inference.ToolChoiceRequired,
		Messages: []*inference.Message{{Role: inference.RoleUser, Parts: []*inference.ContentPart{{Kind: inference.PartText, Text: "list /"}}}},
		Tools:    []*inference.Tool{{Name: "ls", Parameters: `{"type":"object"}`}}}

	events, err := collect(c.Complete(astral.NewContext(nil), req))
	if err != nil {
		t.Fatal(err)
	}
	if r := <-got; r.Model != "m" || len(r.Tools) != 1 || r.Tools[0].Name != "ls" || r.ToolChoice != inference.ToolChoiceRequired {
		t.Fatalf("provider read %+v", r)
	}
	if len(events) != 4 || events[3].Usage == nil || events[3].Usage.CachedPromptTokens != 8 {
		t.Fatalf("events %+v", events)
	}
	var args string
	for _, ev := range events[:3] {
		if ev.Chunk == nil {
			t.Fatalf("events %+v", events)
		}
		for _, d := range ev.Chunk.ToolCalls {
			args += string(d.Arguments)
		}
	}
	if args != `{"path":"/"}` || events[2].Chunk.FinishReason != inference.FinishToolCalls {
		t.Fatalf("arguments %q, finish %q", args, events[2].Chunk.FinishReason)
	}
}

func TestCompleteErrorBeforeChunks(t *testing.T) {
	c := provider(t, nil,
		&inference.Error{Kind: inference.ErrorRateLimited, Status: 429, Message: "slow down", RetryAfter: 7},
		&astral.EOS{},
	)

	events, err := collect(c.Complete(astral.NewContext(nil), &inference.Request{Model: "m"}))
	var ierr *inference.Error
	if !errors.As(err, &ierr) || ierr.Kind != inference.ErrorRateLimited || ierr.Status != 429 || ierr.RetryAfter != 7 {
		t.Fatalf("err = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events %+v before the error", events)
	}
}

func TestCompleteErrorAfterChunks(t *testing.T) {
	c := provider(t, nil,
		&inference.Chunk{Index: 0, Text: "Hel"},
		&inference.Chunk{Index: 1, Text: "lo"},
		&inference.Error{Kind: inference.ErrorFarSide, Status: 502, Message: "upstream reset"},
		&astral.EOS{},
	)

	events, err := collect(c.Complete(astral.NewContext(nil), &inference.Request{Model: "m"}))
	var ierr *inference.Error
	if !errors.As(err, &ierr) || ierr.Kind != inference.ErrorFarSide {
		t.Fatalf("err = %v", err)
	}
	if len(events) != 2 || events[1].Chunk.Text != "lo" {
		t.Fatalf("the chunks before the error do not stand: %+v", events)
	}
}

// An answer that stops without its finishing chunk is a failure the caller
// sees, whether the channel closed or eos came.
func TestCompleteUnfinished(t *testing.T) {
	for name, answer := range map[string][]astral.Object{
		"closed": {&inference.Chunk{Index: 0, Text: "Hel"}},
		"eos":    {&inference.Chunk{Index: 0, Text: "Hel"}, &astral.EOS{}},
		"empty":  {&astral.EOS{}},
	} {
		t.Run(name, func(t *testing.T) {
			c := provider(t, nil, answer...)
			if _, err := collect(c.Complete(astral.NewContext(nil), &inference.Request{})); !errors.Is(err, ErrUnfinished) {
				t.Fatalf("err = %v; want ErrUnfinished", err)
			}
		})
	}
}

// A finished answer whose channel closes without eos is whole.
func TestCompleteFinishedWithoutEOS(t *testing.T) {
	c := provider(t, nil, &inference.Chunk{Index: 0, Text: "ok", FinishReason: inference.FinishStop})
	events, err := collect(c.Complete(astral.NewContext(nil), &inference.Request{}))
	if err != nil || len(events) != 1 {
		t.Fatalf("events %+v, err %v", events, err)
	}
}

func TestCompleteRejectsBrokenOrder(t *testing.T) {
	for name, answer := range map[string][]astral.Object{
		"gap":          {&inference.Chunk{Index: 0}, &inference.Chunk{Index: 2}},
		"late start":   {&inference.Chunk{Index: 1}},
		"after finish": {&inference.Chunk{Index: 0, FinishReason: inference.FinishStop}, &inference.Chunk{Index: 1}},
		"usage first":  {&inference.Usage{TotalTokens: 1}, &inference.Chunk{Index: 0, FinishReason: inference.FinishStop}},
		"usage before finish": {&inference.Chunk{Index: 0}, &inference.Usage{TotalTokens: 1},
			&inference.Chunk{Index: 1, FinishReason: inference.FinishStop}},
		"usage twice": {&inference.Chunk{Index: 0, FinishReason: inference.FinishStop},
			&inference.Usage{TotalTokens: 1}, &inference.Usage{TotalTokens: 1}},
	} {
		t.Run(name, func(t *testing.T) {
			c := provider(t, nil, append(answer, &astral.EOS{})...)
			if _, err := collect(c.Complete(astral.NewContext(nil), &inference.Request{})); err == nil {
				t.Fatal("a broken order passed")
			}
		})
	}
}

// waiting answers inference.complete with first, then waits for the caller to
// close the channel and reports it on released.
func waiting(t *testing.T, first ...astral.Object) (c *Client, started, released chan struct{}) {
	t.Helper()
	started, released = make(chan struct{}), make(chan struct{})
	c, _ = clientFor(t, map[string]any{inference.OpComplete: func(_ *astral.Context, q *routing.IncomingQuery) error {
		ch := q.Accept()
		defer ch.Close()
		var req *inference.Request
		if err := ch.Switch(channel.Expect(&req), channel.FailOnEOF()); err != nil {
			return err
		}
		for _, o := range first {
			if err := ch.Send(o); err != nil {
				return err
			}
		}
		close(started)
		// like a provider: the caller sends nothing after the request, so a
		// read returns only when the caller closes its end
		for {
			if _, err := ch.Receive(); err != nil {
				close(released)
				return nil
			}
		}
	}})
	return c, started, released
}

func waitReleased(t *testing.T, released <-chan struct{}) {
	t.Helper()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("the provider still holds the request 1 s later")
	}
}

// Ending ctx while the provider is still before its first byte, as in a long
// prefill, closes the channel and ends the sequence with ctx's error.
func TestCompleteCancelBeforeFirstChunk(t *testing.T) {
	c, started, released := waiting(t)
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	go func() {
		<-started
		cancel()
	}()
	events, err := collect(c.Complete(ctx, &inference.Request{}))
	if !errors.Is(err, context.Canceled) || len(events) != 0 {
		t.Fatalf("events %+v, err %v; want context.Canceled", events, err)
	}
	waitReleased(t, released)
}

func TestCompleteCancelMidStream(t *testing.T) {
	c, _, released := waiting(t, &inference.Chunk{Index: 0, Text: "Hel"})
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	var chunks int
	var err error
	for ev, e := range c.Complete(ctx, &inference.Request{Stream: true}) {
		if e != nil {
			err = e
			break
		}
		if ev.Chunk != nil {
			chunks++
			cancel()
		}
	}
	if !errors.Is(err, context.Canceled) || chunks != 1 {
		t.Fatalf("chunks %d, err %v; want 1 chunk, then context.Canceled", chunks, err)
	}
	waitReleased(t, released)
}

// A caller that stops ranging releases the request without ending ctx.
func TestCompleteStopRangingClosesTheChannel(t *testing.T) {
	c, _, released := waiting(t, &inference.Chunk{Index: 0, Text: "Hel"})

	for range c.Complete(astral.NewContext(nil), &inference.Request{Stream: true}) {
		break
	}
	waitReleased(t, released)
}
