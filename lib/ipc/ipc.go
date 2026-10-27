package ipc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/akutz/memconn"
	"github.com/astralp2p/astral-go/astral"
)

// local errors
var (
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	ErrInvalidIPCAddress   = errors.New("invalid ipc address")
	ErrNotSocket           = errors.New("file exists and is not a unix socket")
	ErrSocketInUse         = errors.New("unix socket is in use by another process")
)

// staleProbeTimeout bounds the dial that tells a stale socket file from a live one.
const staleProbeTimeout = time.Second

func Dial(target string) (conn *Conn, err error) {
	return DialContext(context.Background(), target)
}

// DialContext connects to target using the format "proto:addr", where proto is one of tcp, unix, memu, or memb.
// For unix sockets it expands "~/", as Listen does, so both ends name the socket the same way.
func DialContext(ctx context.Context, target string) (conn *Conn, err error) {
	parts := strings.SplitN(target, ":", 2)
	if len(parts) < 2 {
		return nil, ErrInvalidIPCAddress
	}
	proto, addr := parts[0], parts[1]

	var c net.Conn
	var dialer net.Dialer

	switch proto {
	case "tcp":
		c, err = dialer.DialContext(ctx, "tcp", addr)

	case "unix":
		addr = expandHome(addr)
		c, err = dialer.DialContext(ctx, "unix", addr)

	case "memu", "memb":
		c, err = memconn.DialContext(ctx, proto, addr)

	default:
		err = ErrUnsupportedProtocol
	}
	if err != nil {
		return nil, err
	}

	conn = &Conn{Conn: c, protocol: proto, addr: addr}

	return
}

// Listen binds to the given "proto:addr" IPC address. For unix sockets it expands "~/" and
// replaces a stale socket file, one that no process accepts on. Any other file at the path, and a
// socket a process still serves, make Listen fail.
func Listen(ipcAddress string) (net.Listener, error) {
	var protocol, address string

	if parts := strings.SplitN(ipcAddress, ":", 2); len(parts) < 2 {
		return nil, ErrInvalidIPCAddress
	} else {
		protocol, address = parts[0], parts[1]
	}

	switch protocol {
	case "tcp":
		return net.Listen("tcp", address)

	case "unix":
		return listenUnix(expandHome(address))

	case "memu", "memb":
		return memconn.Listen(protocol, address)

	default:
		return nil, ErrUnsupportedProtocol
	}
}

// listenUnix listens on the unix socket at path, replacing a stale socket file.
func listenUnix(path string) (net.Listener, error) {
	listen, err := net.Listen("unix", path)
	if err == nil || !errors.Is(err, syscall.EADDRINUSE) {
		return listen, err
	}

	if err = removeStaleSocket(path); err != nil {
		return nil, err
	}

	return net.Listen("unix", path)
}

// removeStaleSocket removes the socket file at path when no process accepts on it.
//
// why: a bind fails with EADDRINUSE for any existing file, so the error alone does not tell a
// crashed node's socket from a live one or from a file that is not a socket.
func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSocket == 0 {
		return fmt.Errorf("%v: %w", path, ErrNotSocket)
	}

	conn, err := net.DialTimeout("unix", path, staleProbeTimeout)
	switch {
	case err == nil:
		conn.Close()
		return fmt.Errorf("%v: %w", path, ErrSocketInUse)
	case !errors.Is(err, syscall.ECONNREFUSED):
		// note: a socket that cannot be probed, e.g. without permission, is kept
		return fmt.Errorf("%v: probe: %w", path, err)
	}

	return os.Remove(path)
}

// ListenAny opens a listener on a system-assigned ephemeral address for the given protocol, useful when the caller does not care which address is used.
func ListenAny(protocol string) (net.Listener, error) {
	switch protocol {
	case "tcp":
		return net.Listen("tcp", "127.0.0.1:0")

	case "unix":
		return net.Listen(
			"unix",
			filepath.Join(
				os.TempDir(),
				"apphostclient."+tempName(16),
			),
		)

	case "memu", "memb":
		return memconn.Listen(protocol, astral.NewNonce().String())

	default:
		return nil, ErrUnsupportedProtocol
	}
}

// expandHome replaces a leading "~/" with the current user's home directory.
// It returns path unchanged when the prefix is absent or the home directory is
// unknown, so a caller always has an address to pass to the network stack.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}

func tempName(length int) (s string) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789_"
	var name = make([]byte, length)
	for i := 0; i < len(name); i++ {
		name[i] = charset[rand.IntN(len(charset))]
	}
	return string(name[:])
}
