package crypto

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestSignerTypesRoundTrip sends every signer type through the binary form and
// the JSON form an app may speak, including an answer with no signature, whose
// nil must not come back as an empty signature.
func TestSignerTypesRoundTrip(t *testing.T) {
	key := &PublicKey{Type: "secp256k1", Key: bytes.Repeat([]byte{2}, 33)}
	id := astral.NewNonce()

	for _, obj := range []astral.Object{
		&SignerKey{Key: key, Scheme: SchemeBIP137},
		&SignerKeys{Keys: []*SignerKey{{Key: key, Scheme: SchemeBIP137}, {Key: key, Scheme: SchemeASN1}}},
		&SignTextRequest{RequestID: id, Key: key, Scheme: SchemeBIP137, Text: "hello world"},
		&SignHashRequest{RequestID: id, Key: key, Scheme: SchemeASN1, Hash: bytes.Repeat([]byte{7}, 32)},
		&SignAnswer{RequestID: id, Signature: &Signature{Scheme: SchemeBIP137, Data: []byte{1, 2, 3}}},
		&SignAnswer{RequestID: id, Error: "declined on the device"},
		&SignCancel{RequestID: id},
	} {
		t.Run(obj.ObjectType(), func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := obj.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}
			got := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(astral.Object)
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}
			if !reflect.DeepEqual(got, obj) {
				t.Fatalf("binary: got %+v, want %+v", got, obj)
			}

			text, err := json.Marshal(obj)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			got = reflect.New(reflect.TypeOf(obj).Elem()).Interface().(astral.Object)
			if err := json.Unmarshal(text, got); err != nil {
				t.Fatalf("unmarshal %s: %v", text, err)
			}
			if !reflect.DeepEqual(got, obj) {
				t.Fatalf("json: got %+v, want %+v (%s)", got, obj, text)
			}
		})
	}
}
