package messaging

import "time"

// WaitRequest is a park on the caller's inbox until it holds a message the
// caller has not archived. It is carried as MethodWait arguments, not as an
// object.
//
// From narrows to one sender by hex identity or alias. Since is a previous
// answer's NextSince; 0 waits on the whole inbox. A zero Timeout takes the
// module's default window, and a Timeout over the module's ceiling is granted
// the ceiling.
type WaitRequest struct {
	From    string
	Since   uint64
	Timeout time.Duration
}
