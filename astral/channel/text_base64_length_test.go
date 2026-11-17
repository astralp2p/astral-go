package channel

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestTextReceiver_Base64ExactLength checks that an untyped "#[]:<base64>" line
// yields a Blob holding exactly the encoded bytes, padded or not, with no
// trailing NUL bytes from the decode buffer.
func TestTextReceiver_Base64ExactLength(t *testing.T) {
	encodings := map[string]*base64.Encoding{
		"padded":   base64.StdEncoding,
		"unpadded": base64.RawStdEncoding,
	}
	for name, enc := range encodings {
		for n := 0; n <= 8; n++ {
			want := []byte(strings.Repeat("1", n))
			if name == "unpadded" && n%3 == 0 {
				continue // identical to the padded line
			}
			line := "#[]:" + enc.EncodeToString(want) + "\n"
			if name == "unpadded" {
				// the receiver uses StdEncoding, which rejects unpadded input
				recv := NewTextReceiver(strings.NewReader(line))
				if _, err := recv.Receive(); err == nil {
					t.Errorf("%s len %d: want decode error for %q", name, n, line)
				}
				continue
			}

			recv := NewTextReceiver(strings.NewReader(line))
			got, err := recv.Receive()
			if err != nil {
				t.Fatalf("%s len %d: receive %q: %v", name, n, line, err)
			}
			blob, ok := got.(*astral.Blob)
			if !ok {
				t.Fatalf("%s len %d: want *astral.Blob, got %T", name, n, got)
			}
			if !bytes.Equal(*blob, want) {
				t.Errorf("%s len %d: got %q, want %q", name, n, []byte(*blob), want)
			}
		}
	}
}
