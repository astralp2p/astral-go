package routing

import "fmt"

// ErrPanic reports an op that panicked. Value is the value passed to panic.
// Stack is the stack trace captured at the point of recovery.
type ErrPanic struct {
	Value any
	Stack []byte
}

var _ error = &ErrPanic{}

func (e *ErrPanic) Error() string {
	return fmt.Sprintf("op panicked: %v", e.Value)
}

// Unwrap exposes the panic value when it is itself an error, so errors.Is and
// errors.As reach it. It returns nil for any other panic value.
func (e *ErrPanic) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}
