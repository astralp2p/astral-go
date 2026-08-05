package astral

import (
	"bytes"
	"fmt"
	"io"
)

// objectReader / objectWriter wrap io.Reader / io.Writer to carry per-call state across
// nested ReadFrom / WriteTo calls. A frame wraps on entry; nested frames detect the
// wrapper and reuse it. Primitives that only need io.Reader / io.Writer don't notice the
// wrapping. The reader additionally carries the per-call Blueprints registry used to
// resolve type names referenced from a Spec; the writer needs no registry because WriteTo
// holds the concrete Object whose schema is already bound.

const (
	// MaxCodecFrames caps how many frames one call may open. Depth bounds nesting but not
	// width: a slice of a million shallow objects passes any depth cap.
	MaxCodecFrames = 1 << 20

	// MaxCodecBytes caps how many bytes one call may move.
	MaxCodecBytes = 1 << 28
)

// budget is the per-call resource ledger.
//
// why: it is held by pointer and shared by every wrapper derived from the one created at
// the codec entry. A frame that buffers a sub-payload and hands it downstream — Bundle
// does this per member, because the wire format frames each one — used to build a fresh
// reader, which started a fresh counter and reset the guard. Sharing the ledger makes
// that laundering impossible rather than merely fixed at the sites known today.
type budget struct {
	depth  int
	frames int64
	bytes  int64
}

func (b *budget) enter(typ fmt.Stringer) error {
	if b == nil {
		return nil
	}

	b.depth++
	if b.depth > MaxBlueprintDepth {
		return fmt.Errorf("%w: %s", ErrDepthExceeded, typ)
	}

	b.frames++
	if b.frames > MaxCodecFrames {
		return fmt.Errorf("%w: %s (frame budget %d)", ErrDepthExceeded, typ, MaxCodecFrames)
	}

	return nil
}

func (b *budget) exit() {
	if b != nil {
		b.depth--
	}
}

// charge accounts for bytes crossing the wrapper.
//
// why: the depth counter is only honoured by frames that call enter. Charging bytes in
// Read and Write means an unguarded frame — including one written after this — still
// cannot move unbounded data, because bytes reach it only through the wrapper.
func (b *budget) charge(n int) error {
	if b == nil || n <= 0 {
		return nil
	}

	b.bytes += int64(n)
	if b.bytes > MaxCodecBytes {
		return fmt.Errorf("%w: byte budget %d", ErrDepthExceeded, MaxCodecBytes)
	}
	return nil
}

type objectReader struct {
	io.Reader
	b *budget
	// bps is the registry that nested field reads consult to resolve type names referenced
	// from a Spec (PrimitiveSpec, RefSpec, PtrSpec). Nil → defaultBlueprints, matching the
	// pre-WithBlueprints behaviour. Decode threads cfg.Blueprints in here so a per-call
	// registry (set via WithBlueprints) flows down into every nested ReadFrom frame.
	bps *Blueprints
}

type objectWriter struct {
	io.Writer
	b *budget
}

// attachReader is the only constructor of a fresh ledger on the read side. An existing
// wrapper is reused, so a nested frame inherits both the ledger and the registry.
func attachReader(r io.Reader, bps *Blueprints) *objectReader {
	if or, ok := r.(*objectReader); ok {
		if or.bps == nil {
			or.bps = bps
		}
		if or.b == nil {
			or.b = &budget{}
		}
		return or
	}
	return &objectReader{Reader: r, b: &budget{}, bps: bps}
}

func attachWriter(w io.Writer) *objectWriter {
	if ow, ok := w.(*objectWriter); ok {
		if ow.b == nil {
			ow.b = &budget{}
		}
		return ow
	}
	return &objectWriter{Writer: w, b: &budget{}}
}

// sub derives a reader over an in-memory sub-payload that shares this reader's ledger.
// It replaces every bytes.NewReader a codec path would otherwise hand downstream.
func (or *objectReader) sub(p []byte) *objectReader {
	return &objectReader{Reader: bytes.NewReader(p), b: or.b, bps: or.bps}
}

// subWriter derives a writer over a staging buffer that shares this writer's ledger.
// Length-prefixed framing needs the payload length before the payload can be emitted, so
// the buffer is required by the wire format; sharing the ledger is what stops it
// laundering the guard.
func (ow *objectWriter) subWriter(w io.Writer) *objectWriter {
	return &objectWriter{Writer: w, b: ow.b}
}

func (or *objectReader) Read(p []byte) (int, error) {
	n, err := or.Reader.Read(p)
	if cerr := or.b.charge(n); cerr != nil {
		return n, cerr
	}
	return n, err
}

func (ow *objectWriter) Write(p []byte) (int, error) {
	n, err := ow.Writer.Write(p)
	if cerr := ow.b.charge(n); cerr != nil {
		return n, cerr
	}
	return n, err
}

// resolve returns the registry the reader should consult for nested type lookups.
// Falls back to defaultBlueprints so paths that hand-wrap a reader without setting bps
// keep their historical behaviour.
func (or *objectReader) resolve() *Blueprints {
	if or.bps != nil {
		return or.bps
	}
	return defaultBlueprints
}

// enterReader is the single guard every recursive read frame calls. It attaches a ledger
// when the caller supplied a naked reader — the true codec entry — and charges one frame.
// Pair with a deferred exit so the counter unwinds on every return path, including errors.
func enterReader(r io.Reader, typ fmt.Stringer) (*objectReader, error) {
	or := attachReader(r, nil)
	return or, or.b.enter(typ)
}

func enterWriter(w io.Writer, typ fmt.Stringer) (*objectWriter, error) {
	ow := attachWriter(w)
	return ow, ow.b.enter(typ)
}

func (or *objectReader) enter(typ fmt.Stringer) error { return or.b.enter(typ) }
func (or *objectReader) exit()                        { or.b.exit() }
func (ow *objectWriter) enter(typ fmt.Stringer) error { return ow.b.enter(typ) }
func (ow *objectWriter) exit()                        { ow.b.exit() }

// frameName labels a codec frame in a depth error.
type frameName string

func (f frameName) String() string { return string(f) }
