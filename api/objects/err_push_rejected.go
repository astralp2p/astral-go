package objects

import "errors"

// ErrPushRejected - the target node received a pushed object and did not accept it
var ErrPushRejected = errors.New("push rejected")
