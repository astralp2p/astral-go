package log

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/fmt"
)

var (
	reentrantArmed  atomic.Bool
	reentrantLogger atomic.Pointer[Logger]
)

// reentrantEntryView renders an entry's objects, and once armed logs from
// inside Render. astrald's mod/log renders an identity through a database
// read, and a database logger wired back into this logger re-enters exactly
// here.
type reentrantEntryView struct{ e *Entry }

func (v reentrantEntryView) Render() string {
	if reentrantArmed.CompareAndSwap(true, false) {
		if l := reentrantLogger.Load(); l != nil {
			l.Log("nested from render")
		}
	}

	var out string
	for _, o := range v.e.Objects {
		out += astral.Stringify(o)
	}

	return out
}

func init() {
	fmt.SetView(func(e *Entry) fmt.View { return reentrantEntryView{e} })
}

func newTestLogger(w io.Writer) *Logger {
	var l = &Logger{
		id:    astral.Anyone,
		w:     w,
		queue: make(chan string, rootQueueCap),
	}

	go l.pump()

	return l
}

// blockingWriter never returns from Write, which is a full pipe to a console
// nobody reads.
type blockingWriter struct{ release chan struct{} }

func (w *blockingWriter) Write(p []byte) (int, error) {
	<-w.release
	return len(p), nil
}

// TestRenderMayReEnterLogger fails by deadlock if the render runs under the
// root mutex.
func TestRenderMayReEnterLogger(t *testing.T) {
	var l = newTestLogger(io.Discard)

	reentrantLogger.Store(l)
	reentrantArmed.Store(true)
	defer reentrantArmed.Store(false)

	var done = make(chan struct{})

	go func() {
		defer close(done)
		l.Log("%v", astral.NewString32("outer"))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("logging deadlocked while a view re-entered the logger")
	}
}

// TestBlockedWriterDoesNotBlockLogging fails by timeout if the root writer is
// written under the root mutex.
func TestBlockedWriterDoesNotBlockLogging(t *testing.T) {
	var release = make(chan struct{})
	defer close(release)

	var l = newTestLogger(&blockingWriter{release: release})

	var done = make(chan struct{})

	go func() {
		defer close(done)
		for range rootQueueCap * 4 {
			l.Log("%v", astral.NewString32("entry"))
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a blocked root writer stalled the goroutine that logs")
	}

	if l.dropped.Load() == 0 {
		t.Error("a blocked writer dropped no entries, want drops counted")
	}
}

// TestEntryQueuesOneLine records that an entry reaches the writer as a single
// rendered line. Printer.Printf writes once per formatted token, so rendering
// through it wrote twice per entry.
func TestEntryQueuesOneLine(t *testing.T) {
	var l = &Logger{id: astral.Anyone, queue: make(chan string, 4)}

	l.logEntry(NewEntry(astral.Anyone, 0, astral.NewString32("hello")))

	if len(l.queue) != 1 {
		t.Fatalf("queued %v lines, want 1", len(l.queue))
	}

	var line = <-l.queue

	if !strings.Contains(line, "hello") {
		t.Errorf("queued line %q does not carry the entry", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Errorf("queued line %q does not end with a newline", line)
	}
}

// countingLogger records the entries a subscriber receives.
type countingLogger struct{ n atomic.Int64 }

func (c *countingLogger) LogEntry(*Entry) { c.n.Add(1) }

// TestFilterGatesTheWriterOnly records that the filter keeps an entry off the
// console but still delivers it to subscribers. astrald's log.listen shows
// every entry regardless of the node's console verbosity, so filtering the
// fan-out silently breaks its watchdog.
func TestFilterGatesTheWriterOnly(t *testing.T) {
	var l = &Logger{id: astral.Anyone, queue: make(chan string, 4)}
	var sub countingLogger

	l.AddLogger(&sub)
	l.SetFilter(func(*Entry) bool { return false })
	l.logEntry(NewEntry(astral.Anyone, 0, astral.NewString32("hidden")))

	if len(l.queue) != 0 {
		t.Errorf("queued %v lines for a filtered entry, want 0", len(l.queue))
	}
	if got := sub.n.Load(); got != 1 {
		t.Errorf("subscriber received %v entries, want 1", got)
	}
}
