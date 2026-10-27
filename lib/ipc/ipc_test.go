package ipc

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// A regular file at the socket path is never removed.
func TestListen_regularFileIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apphost.sock")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}

	listener, err := Listen("unix:" + path)
	if err == nil {
		listener.Close()
		t.Fatal("Listen succeeded over a regular file")
	}
	if !errors.Is(err, ErrNotSocket) || !strings.Contains(err.Error(), path) {
		t.Fatalf("Listen error = %v, want ErrNotSocket naming %v", err, path)
	}

	if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
		t.Fatalf("file after Listen = (%q, %v), want (%q, nil)", data, err, "keep")
	}
}

// A socket that a process still serves is never taken over.
func TestListen_liveSocketIsKept(t *testing.T) {
	address := "unix:" + filepath.Join(t.TempDir(), "apphost.sock")

	first, err := Listen(address)
	if err != nil {
		t.Fatalf("first Listen: %v", err)
	}
	defer first.Close()

	accepted := make(chan error, 2)
	go func() {
		for {
			conn, err := first.Accept()
			if err != nil {
				return
			}
			conn.Close()
			accepted <- nil
		}
	}()

	second, err := Listen(address)
	if err == nil {
		second.Close()
		t.Fatal("second Listen took over a live socket")
	}
	if !errors.Is(err, ErrSocketInUse) {
		t.Fatalf("second Listen error = %v, want ErrSocketInUse", err)
	}

	conn, err := Dial(address)
	if err != nil {
		t.Fatalf("Dial after the refused Listen: %v", err)
	}
	conn.Close()

	select {
	case <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("the first listener no longer accepts")
	}
}

// A socket file that no process accepts on is replaced.
func TestListen_staleSocketIsReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apphost.sock")

	crashed, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	// note: a closed listener that keeps its file is what a crashed process leaves
	crashed.(*net.UnixListener).SetUnlinkOnClose(false)
	crashed.Close()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stale socket file missing: %v", err)
	}

	listener, err := Listen("unix:" + path)
	if err != nil {
		t.Fatalf("Listen over a stale socket: %v", err)
	}
	defer listener.Close()

	conn, err := Dial("unix:" + path)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	conn.Close()
}
