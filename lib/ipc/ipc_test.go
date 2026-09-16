package ipc

import (
	"path/filepath"
	"testing"
)

// A "~/" address names one socket to both ends: what Listen binds, Dial reaches.
func TestDialContext_expandsHomeLikeListen(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	const address = "unix:~/apphost.sock"

	listener, err := Listen(address)
	if err != nil {
		t.Fatalf("Listen(%v): %v", address, err)
	}
	defer listener.Close()

	if want := filepath.Join(home, "apphost.sock"); listener.Addr().String() != want {
		t.Fatalf("listening on %v, want %v", listener.Addr(), want)
	}

	conn, err := Dial(address)
	if err != nil {
		t.Fatalf("Dial(%v): %v", address, err)
	}
	defer conn.Close()

	if _, err := listener.Accept(); err != nil {
		t.Fatalf("Accept: %v", err)
	}
}

// An address without the prefix reaches the network stack untouched.
func TestExpandHome_leavesOtherPathsAlone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	for _, path := range []string{"/var/run/apphost.sock", "apphost.sock", "~apphost.sock", ""} {
		if got := expandHome(path); got != path {
			t.Errorf("expandHome(%q) = %q, want it unchanged", path, got)
		}
	}
}
