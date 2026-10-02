package services

import "errors"

// ErrAdvertised rejects an advertisement naming a service the provider already
// advertises on a live binding. A node sends its message as an error_message;
// a client compares messages, because error_message carries only text.
var ErrAdvertised = errors.New("service already advertised by this provider")
