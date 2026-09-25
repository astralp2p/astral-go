package messaging

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func envelopesAt(cursors ...astral.Uint64) (list []*Envelope) {
	for _, c := range cursors {
		list = append(list, &Envelope{Cursor: c})
	}
	return
}

func TestNextSince(t *testing.T) {
	for name, tc := range map[string]struct {
		list  []*Envelope
		since uint64
		want  uint64
	}{
		"nothing, from the start":  {nil, 0, 0},
		"nothing, keeps since":     {nil, 9, 9},
		"empty list keeps since":   {[]*Envelope{}, 9, 9},
		"the furthest, any order":  {envelopesAt(4, 11, 7), 0, 11},
		"newer than since":         {envelopesAt(12, 15), 9, 15},
		"none newer keeps since":   {envelopesAt(3, 5), 9, 9},
		"equal to since":           {envelopesAt(9), 9, 9},
		"a nil row is skipped":     {[]*Envelope{nil, {Cursor: 5}, nil}, 2, 5},
		"only nil rows keep since": {[]*Envelope{nil}, 2, 2},
	} {
		t.Run(name, func(t *testing.T) {
			if got := NextSince(tc.list, tc.since); got != tc.want {
				t.Fatalf("NextSince: want %v, got %v", tc.want, got)
			}
		})
	}
}

// A wait's answer carries its own NextSince; what a caller computes from the
// rows it came back with must agree once it crosses the wire.
func TestWaitResult_NextSinceSurvives(t *testing.T) {
	src := &WaitResult{Messages: envelopesAt(20, 21), Granted: astral.Duration(2e9)}
	src.NextSince = astral.Uint64(NextSince(src.Messages, 3))

	dst := jsonRoundTrip(t, src).(*WaitResult)
	if dst.NextSince != 21 || NextSince(dst.Messages, 3) != 21 {
		t.Fatalf("next since: want 21, got %v (rows %v)", dst.NextSince, NextSince(dst.Messages, 3))
	}
	if dst.Granted != src.Granted {
		t.Fatalf("granted: want %v, got %v", src.Granted, dst.Granted)
	}
}
