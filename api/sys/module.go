// Package sys holds the wire types and op names of the sys protocol: the
// node's manager and its appointment.
package sys

import "errors"

const ModuleName = "sys"

const (
	MethodRegisterManager = "sys.register_manager"
)

var ErrProtocolError = errors.New("protocol error")
