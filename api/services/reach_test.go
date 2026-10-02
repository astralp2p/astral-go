package services

import "testing"

func TestParseReach(t *testing.T) {
	for in, want := range map[string]Reach{"": ReachLocal, "local": ReachLocal, "swarm": ReachSwarm} {
		if got, err := ParseReach(in); err != nil || got != want {
			t.Fatalf("ParseReach(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"network", "Swarm", " local"} {
		if _, err := ParseReach(in); err == nil {
			t.Fatalf("ParseReach(%q) accepted an unknown reach", in)
		}
	}
}
