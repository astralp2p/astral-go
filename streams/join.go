package streams

import (
	"io"
	"sync"
)

// Join copies from each stream to the other until one of them reaches EOF or an error. Returns bytes written
// both ways, and a nil error if EOF was reached, or the first non-EOF error.
func Join(left, right io.ReadWriteCloser) (writtenL int64, writtenR int64, err error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(2)

	// setErr keeps the first non-nil error in completion order.
	setErr := func(e error) {
		mu.Lock()
		defer mu.Unlock()
		if err == nil {
			err = e
		}
	}

	go func() {
		defer wg.Done()
		defer left.Close()

		var errL error
		writtenL, errL = io.Copy(left, right)
		setErr(errL)
	}()

	go func() {
		defer wg.Done()
		defer right.Close()

		var errR error
		writtenR, errR = io.Copy(right, left)
		setErr(errR)
	}()

	wg.Wait()
	return
}
