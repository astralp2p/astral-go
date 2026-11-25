package nodes

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/tcp"
	"github.com/astralp2p/astral-go/astral"
)

func newTestLinkInfo() LinkInfo {
	return LinkInfo{
		ID:             astral.NewNonce(),
		LocalIdentity:  astral.GenerateIdentity(),
		RemoteIdentity: astral.GenerateIdentity(),
		LocalEndpoint:  &tcp.Endpoint{},
		RemoteEndpoint: &tcp.Endpoint{},
		Network:        "tcp",
		Age:            astral.Duration(2*time.Hour + 14*time.Minute),
	}
}

func duration(d time.Duration) *astral.Duration {
	v := astral.Duration(d)
	return &v
}

// A link with no pong yet carries no RTT: the fields stay absent across the binary codec.
func TestLinkInfoRoundTrip_NoRTT(t *testing.T) {
	want := newTestLinkInfo()

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	var got LinkInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.Age != want.Age {
		t.Errorf("Age = %v, want %v", got.Age, want.Age)
	}
	if got.RTT != nil || got.RTTAge != nil {
		t.Errorf("RTT, RTTAge = %v, %v, want nil, nil", got.RTT, got.RTTAge)
	}
}

func TestLinkInfoRoundTrip_RTT(t *testing.T) {
	want := newTestLinkInfo()
	want.RTT = duration(18 * time.Millisecond)
	want.RTTAge = duration(2 * time.Minute)

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	var got LinkInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.RTT == nil || *got.RTT != *want.RTT {
		t.Errorf("RTT = %v, want %v", got.RTT, *want.RTT)
	}
	if got.RTTAge == nil || *got.RTTAge != *want.RTTAge {
		t.Errorf("RTTAge = %v, want %v", got.RTTAge, *want.RTTAge)
	}
	if buf.Len() != 0 {
		t.Errorf("%d bytes left after ReadFrom", buf.Len())
	}
}

// JSON carries the durations in nanoseconds, and an absent RTT as null.
func TestLinkInfoJSON_RTT(t *testing.T) {
	for _, tc := range []struct {
		name string
		rtt  *astral.Duration
		want string
	}{
		{"absent", nil, `"RTT":null`},
		{"present", duration(18 * time.Millisecond), `"RTT":18000000`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := newTestLinkInfo()
			want.RTT, want.RTTAge = tc.rtt, tc.rtt

			b, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if !strings.Contains(string(b), tc.want) || !strings.Contains(string(b), `"Age":8040000000000`) {
				t.Fatalf("JSON = %s, want %s and Age in nanoseconds", b, tc.want)
			}

			var got LinkInfo
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if (got.RTT == nil) != (tc.rtt == nil) || (got.RTT != nil && *got.RTT != *tc.rtt) {
				t.Errorf("RTT = %v, want %v", got.RTT, tc.rtt)
			}
		})
	}
}
