package ipc

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/akutz/memconn"
	"github.com/astralp2p/astral-go/astral"
)

// local errors
var (
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	ErrInvalidIPCAddress   = errors.New("invalid ipc address")
)

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

// Listen binds to the given "proto:addr" IPC address; for unix sockets it expands "~/" and removes a stale socket file before retrying.
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
		var path = expandHome(address)

		listen, err := net.Listen("unix", path)
		if err != nil {
			// if the socket already exists, remove it if it's stale and listen again
			if strings.Contains(err.Error(), "already in use") {
				rerr := os.Remove(path)
				if rerr != nil {
					return nil, err
				}

				listen, err = net.Listen("unix", path)
			}
		}

		return listen, err

	case "memu", "memb":
		return memconn.Listen(protocol, address)

	default:
		return nil, ErrUnsupportedProtocol
	}
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
