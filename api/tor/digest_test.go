package tor

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func realDigest() Digest {
	d := make(Digest, DigestSize)
	for i := range d {
		d[i] = byte(i + 1)
	}
	return d
}

// ReadFrom demands DigestSize, so WriteTo owes DigestSize whatever it holds.
// The zero value used to write nothing at all, which no reader could consume.
func TestDigestWriteTo_ZeroValueIsFullWidth(t *testing.T) {
	var buf bytes.Buffer

	n, err := Digest(nil).WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if want := int64(DigestSize); n != want {
		t.Errorf("reported %d bytes, want %d", n, want)
	}
	if got, want := buf.Len(), DigestSize; got != want {
		t.Errorf("wrote %d bytes, want %d", got, want)
	}
	if !bytes.Equal(buf.Bytes(), make([]byte, DigestSize)) {
		t.Errorf("wrote %x, want all null", buf.Bytes())
	}
}

// The zero value must survive the wire as the zero value: Endpoint.IsZero and
// Address both read len(Digest) == 0, so a digest that came back as DigestSize
// null bytes would claim to name an onion service that does not exist.
func TestDigestRoundTrip_ZeroValue(t *testing.T) {
	var buf bytes.Buffer

	if _, err := Digest(nil).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got Digest
	n, err := got.ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if want := int64(DigestSize); n != want {
		t.Errorf("read %d bytes, want %d", n, want)
	}
	if len(got) != 0 {
		t.Errorf("Digest = %x, want the zero value", got)
	}
}

// Mapping all-null back to the zero value must not swallow a real digest.
func TestDigestRoundTrip_RealDigest(t *testing.T) {
	want := realDigest()

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if got := buf.Len(); got != DigestSize {
		t.Fatalf("wrote %d bytes, want %d", got, DigestSize)
	}

	var got Digest
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Digest = %x, want %x", got, want)
	}
}

// A digest of some other length is not a digest. Padding or truncating it to fit
// would put a different address on the wire; erroring says so.
func TestDigestWriteTo_WrongLengthRefused(t *testing.T) {
	var buf bytes.Buffer

	n, err := Digest(bytes.Repeat([]byte{0xab}, DigestSize-1)).WriteTo(&buf)
	if !errors.Is(err, ErrInvalidDigestLength) {
		t.Fatalf("WriteTo error = %v, want ErrInvalidDigestLength", err)
	}
	if n != 0 {
		t.Errorf("reported %d bytes, want 0", n)
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %d bytes, want none", buf.Len())
	}
}

// The defect that mattered: a short digest does not merely lose itself, it moves
// every following field of the enclosing object by the bytes it did not write.
// This is the failure astral-py reported as a ShortRead against nodes.new_link.
func TestDigestRoundTrip_ZeroValueDoesNotShiftFollowingFields(t *testing.T) {
	type record struct {
		Digest Digest
		Tail   astral.Uint64
	}

	want := record{Tail: 0x0102030405060708}

	var buf bytes.Buffer
	if _, err := astral.Objectify(&want).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if got, expect := buf.Len(), DigestSize+8; got != expect {
		t.Fatalf("wrote %d bytes, want %d", got, expect)
	}

	var got record
	if _, err := astral.Objectify(&got).ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.Tail != want.Tail {
		t.Errorf("Tail = %x, want %x -- the digest shifted the fields after it", got.Tail, want.Tail)
	}
}

// A zero Endpoint is what a link with no address to report carries, and it must
// come back reading as unknown rather than as a null onion service.
func TestEndpointRoundTrip_ZeroValue(t *testing.T) {
	var buf bytes.Buffer

	if _, err := (Endpoint{}).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if got, want := buf.Len(), DigestSize+2; got != want {
		t.Fatalf("wrote %d bytes, want %d", got, want)
	}

	var got Endpoint
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("Endpoint = %v, want the zero endpoint", got.Address())
	}
	if want := "unknown"; got.Address() != want {
		t.Errorf("Address = %q, want %q", got.Address(), want)
	}
}

// UnmarshalText accepts exactly one text form for the zero digest, so
// MarshalText owes that form. Emitting a bare .onion instead made the type's own
// encoder produce text its own parser refuses.
func TestDigestText_ZeroValueRoundTrips(t *testing.T) {
	text, err := Digest(nil).MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if want := "unknown"; string(text) != want {
		t.Errorf("MarshalText = %q, want %q", text, want)
	}

	var got Digest
	if err := got.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", text, err)
	}
	if len(got) != 0 {
		t.Errorf("Digest = %x, want the zero value", got)
	}
}

// The zero value is the only case that changed; a real digest keeps the
// lowercase base32 .onion form the spec states.
func TestDigestText_RealValueRoundTrips(t *testing.T) {
	want := realDigest()

	text, err := want.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if !strings.HasSuffix(string(text), ".onion") {
		t.Errorf("MarshalText = %q, want a .onion hostname", text)
	}

	var got Digest
	if err := got.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", text, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Digest = %x, want %x", got, want)
	}
}

// MarshalJSON delegates to MarshalText, so text and JSON agree on every value
// rather than differing on the zero one alone.
func TestDigestText_AgreesWithJSON(t *testing.T) {
	for _, d := range []Digest{nil, realDigest()} {
		text, err := d.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText: %v", err)
		}

		raw, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
		var asJSON string
		if err := json.Unmarshal(raw, &asJSON); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if string(text) != asJSON {
			t.Errorf("MarshalText = %q, JSON = %q", text, asJSON)
		}
	}
}

// The zero digest survives a JSON round trip as the zero value: UnmarshalJSON
// routes through UnmarshalText, which used to refuse what MarshalJSON emitted.
func TestDigestJSON_ZeroValueRoundTrips(t *testing.T) {
	raw, err := json.Marshal(Digest(nil))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var got Digest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("UnmarshalJSON(%s): %v", raw, err)
	}
	if len(got) != 0 {
		t.Errorf("Digest = %x, want the zero value", got)
	}
}

// A zero digest inside an endpoint keeps the endpoint's own unknown form: the
// digest's text form is not spliced into <digest>:<port>, because Address
// short-circuits on IsZero before formatting.
func TestDigestText_ZeroValueDoesNotLeakIntoEndpointAddress(t *testing.T) {
	if got, want := (&Endpoint{}).Address(), "unknown"; got != want {
		t.Errorf("Address = %q, want %q", got, want)
	}
}
