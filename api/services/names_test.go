package services

import (
	"strings"
	"testing"
)

func TestParseNames(t *testing.T) {
	for _, tc := range []struct {
		list string
		want []string
		ok   bool
	}{
		{"player", []string{"player"}, true},
		{"player,bitcoin-wallet", []string{"player", "bitcoin-wallet"}, true},
		{"contacts-backend", []string{"contacts-backend"}, true},
		{"github.com/project-xyz", []string{"github.com/project-xyz"}, true},

		{"", nil, false},
		{",", nil, false},
		{"player,", nil, false},
		{"player,,wallet", nil, false},
		{" player", nil, false},
		{"player ", nil, false},
		{"player,player", nil, false},
		{strings.Repeat("a", 256), nil, false},
		{strings.Repeat("a,", MaxNames) + "a", nil, false},
	} {
		got, err := ParseNames(tc.list)
		if (err == nil) != tc.ok {
			t.Fatalf("ParseNames(%q): err %v, want ok=%v", tc.list, err, tc.ok)
		}
		if !tc.ok {
			continue
		}
		if JoinNames(got) != JoinNames(tc.want) {
			t.Fatalf("ParseNames(%q) = %q, want %q", tc.list, got, tc.want)
		}
	}
}
