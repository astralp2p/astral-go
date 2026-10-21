package nodes

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func signalChannel(t *testing.T, signals ...*MigrateSignal) *channel.Channel {
	t.Helper()
	var in bytes.Buffer
	s := channel.NewBinarySender(&in)
	for _, sig := range signals {
		if err := s.Send(sig); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	return channel.New(channel.Join(&in, &bytes.Buffer{}))
}

// TestExpectMigrateSignal_FailOnEOF_ClosedBeforeSignal: a channel that ends
// before the signal fails the Switch and leaves buf untouched.
func TestExpectMigrateSignal_FailOnEOF_ClosedBeforeSignal(t *testing.T) {
	buf := astral.Uint32(7)
	err := signalChannel(t).Switch(ExpectMigrateSignal(MigrateSignalReady, &buf), channel.FailOnEOF())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
	if buf != 7 {
		t.Fatalf("buf = %d, want 7", buf)
	}
}

// TestExpectMigrateSignal_FailOnEOF_ReceivesSignal: the expected signal ends
// the Switch with nil and writes the peer's buffer size.
func TestExpectMigrateSignal_FailOnEOF_ReceivesSignal(t *testing.T) {
	var buf astral.Uint32
	ch := signalChannel(t, &MigrateSignal{Signal: MigrateSignalReady, Buffer: 4096})
	err := ch.Switch(ExpectMigrateSignal(MigrateSignalReady, &buf), channel.FailOnEOF())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if buf != 4096 {
		t.Fatalf("buf = %d, want 4096", buf)
	}
}
