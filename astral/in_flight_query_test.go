package astral

import "testing"

func stamped(origin any) *InFlightQuery {
	q := Launch(&Query{})
	if origin != nil {
		q.Extra.Set("origin", origin)
	}
	return q
}

func TestOriginPredicates(t *testing.T) {
	tests := []struct {
		name                      string
		origin                    any
		isLocal, isNetwork, isMCP bool
	}{
		{"unset", nil, true, false, false},
		{"empty", "", true, false, false},
		{"local", OriginLocal, true, false, false},
		{"network", OriginNetwork, false, true, false},
		{"mcp", OriginMCP, false, false, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := stamped(test.origin)

			if got := q.IsLocal(); got != test.isLocal {
				t.Errorf("IsLocal() = %v, want %v", got, test.isLocal)
			}
			if got := q.IsNetwork(); got != test.isNetwork {
				t.Errorf("IsNetwork() = %v, want %v", got, test.isNetwork)
			}
			if got := q.IsMCP(); got != test.isMCP {
				t.Errorf("IsMCP() = %v, want %v", got, test.isMCP)
			}
		})
	}
}

// an origin this build does not know is neither local nor network nor MCP, so a
// guard written as a positive test refuses it and one written against a named
// value does not.
func TestUnknownOriginMatchesNoPredicate(t *testing.T) {
	q := stamped("something-else")

	if q.IsLocal() || q.IsNetwork() || q.IsMCP() {
		t.Fatalf("unknown origin matched a predicate: local=%v network=%v mcp=%v",
			q.IsLocal(), q.IsNetwork(), q.IsMCP())
	}
}
