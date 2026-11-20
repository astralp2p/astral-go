package log

import (
	"io"
	"strconv"
	"strings"
	"sync"
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
		id:     astral.Anyone,
		w:      w,
		queue:  make(chan string, rootQueueCap),
		flushc: make(chan chan struct{}),
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

// lockedBuffer is a writer the pump and a test read without racing.
type lockedBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// lines returns the written lines, waiting for want of them.
func (b *lockedBuffer) lines(t *testing.T, want int) []string {
	t.Helper()

	var deadline = time.Now().Add(5 * time.Second)

	for {
		var out = strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
		if b.String() == "" {
			out = nil
		}
		if len(out) >= want || time.Now().After(deadline) {
			return out
		}
		time.Sleep(time.Millisecond)
	}
}

// setRenderer stands in for a module registering the entry view.
func setRenderer(t *testing.T, set bool) *atomic.Bool {
	t.Helper()

	var ready atomic.Bool
	ready.Store(set)

	var old = rendererSet
	rendererSet = ready.Load
	t.Cleanup(func() { rendererSet = old })

	return &ready
}

func logN(l *Logger, from, to int) {
	for i := from; i < to; i++ {
		l.Log("%v", astral.NewString32("line "+strconv.Itoa(i)))
	}
}

func wantLines(t *testing.T, got []string, from, to int) {
	t.Helper()

	if len(got) != to-from {
		t.Fatalf("wrote %v lines, want %v: %q", len(got), to-from, got)
	}

	for i, line := range got {
		if !strings.HasSuffix(line, "line "+strconv.Itoa(from+i)) {
			t.Errorf("line %v is %q, want line %v", i, line, from+i)
		}
	}
}

// TestEntriesBeforeRendererWaitInOrder records that entries written before a
// renderer is set print once it is, in order and ahead of later entries, and
// that nothing prints meanwhile.
func TestEntriesBeforeRendererWaitInOrder(t *testing.T) {
	var ready = setRenderer(t, false)
	var out lockedBuffer
	var l = newTestLogger(&out)

	logN(l, 0, 3)

	time.Sleep(20 * time.Millisecond)
	if got := out.String(); got != "" {
		t.Fatalf("wrote %q before a renderer was set, want nothing", got)
	}

	ready.Store(true)
	logN(l, 3, 5)

	wantLines(t, out.lines(t, 5), 0, 5)
}

// TestEntriesAfterRendererSkipTheHold records that with a renderer set an
// entry goes straight to the queue.
func TestEntriesAfterRendererSkipTheHold(t *testing.T) {
	setRenderer(t, true)

	var l = &Logger{id: astral.Anyone, queue: make(chan string, 4)}

	logN(l, 0, 1)

	if len(l.queue) != 1 || len(l.held) != 0 {
		t.Errorf("queued %v and held %v, want 1 and 0", len(l.queue), len(l.held))
	}
}

// TestCloseWritesEntriesHeldForARenderer records the exit path: a process that
// ends before any renderer is set still gets its early entries out. It checks
// delivery, not rendering: the test binary registers an entry view for the
// whole process, so the struct dump a real early line would get is not
// exercised here.
func TestCloseWritesEntriesHeldForARenderer(t *testing.T) {
	setRenderer(t, false)

	var out lockedBuffer
	var l = newTestLogger(&out)

	logN(l, 0, 3)
	l.Close()

	// Close returns once the writer has them: no waiting here.
	wantLines(t, strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"), 0, 3)
}

// TestNoRendererNoReleaseWritesNothing documents the bound: with no renderer
// ever set and neither Release nor Close called, entries stay held and nothing
// is written. This is why astrald releases the hold after module loading.
func TestNoRendererNoReleaseWritesNothing(t *testing.T) {
	setRenderer(t, false)

	var out lockedBuffer
	var l = newTestLogger(&out)

	logN(l, 0, heldCap+10)

	time.Sleep(20 * time.Millisecond)
	if got := out.String(); got != "" {
		t.Fatalf("wrote %q with no renderer and no release, want nothing", got)
	}

	if len(l.held) != heldCap {
		t.Errorf("held %v entries, want %v", len(l.held), heldCap)
	}
}

// TestReleaseWritesHeldEntriesAndStopsHolding records that Release writes the
// held entries without a renderer, in order, and that later entries are not
// held again.
func TestReleaseWritesHeldEntriesAndStopsHolding(t *testing.T) {
	setRenderer(t, false)

	var out lockedBuffer
	var l = newTestLogger(&out)

	logN(l, 0, 3)
	l.SetPrefix().Release() // a child releases the root

	logN(l, 3, 5)

	wantLines(t, out.lines(t, 5), 0, 5)

	if len(l.held) != 0 {
		t.Errorf("held %v entries after release, want 0", len(l.held))
	}
}

// TestCloseReportsRefusedEntries records that the drop line is written even
// when the queue is empty by the time Close flushes.
func TestCloseReportsRefusedEntries(t *testing.T) {
	setRenderer(t, true)

	var out lockedBuffer
	var l = newTestLogger(&out)

	l.dropped.Store(3)
	l.firstDrop.Store(time.Now().UnixNano())
	l.Close()

	if got := out.String(); !strings.Contains(got, "log: dropped 3 entries since") {
		t.Errorf("wrote %q, want the drop line", got)
	}
}

// TestHeldBufferOverflowKeepsTheEarliest records the bound: past heldCap the
// newest entries are refused, counted and named in one line after the kept ones.
func TestHeldBufferOverflowKeepsTheEarliest(t *testing.T) {
	setRenderer(t, false)

	var out lockedBuffer
	var l = newTestLogger(&out)

	logN(l, 0, heldCap+5)
	l.Close()

	var got = strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(got) != heldCap+1 {
		t.Fatalf("wrote %v lines, want %v", len(got), heldCap+1)
	}

	wantLines(t, got[:heldCap], 0, heldCap)

	if !strings.HasPrefix(got[heldCap], "log: dropped 5 entries") {
		t.Errorf("last line is %q, want the drop report", got[heldCap])
	}
}

// TestHeldEntriesKeepOrderUnderConcurrentLogging records that entries logged
// while the held ones are written out, from several goroutines, never overtake
// them, and that none is lost.
func TestHeldEntriesKeepOrderUnderConcurrentLogging(t *testing.T) {
	var ready = setRenderer(t, false)
	var out lockedBuffer
	var l = newTestLogger(&out)

	const early, writers, each = 200, 4, 100

	logN(l, 0, early)

	var wg sync.WaitGroup
	ready.Store(true)

	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range each {
				l.Log("%v", astral.NewString32("w"+strconv.Itoa(w)+" "+strconv.Itoa(i)))
			}
		}()
	}

	wg.Wait()

	var got = out.lines(t, early+writers*each)
	if len(got) != early+writers*each {
		t.Fatalf("wrote %v lines, want %v", len(got), early+writers*each)
	}

	wantLines(t, got[:early], 0, early)

	var next [writers]int
	for _, line := range got[early:] {
		var w, i int
		var f = strings.Fields(line)
		w, _ = strconv.Atoi(strings.TrimPrefix(f[len(f)-2], "w"))
		i, _ = strconv.Atoi(f[len(f)-1])
		if i != next[w] {
			t.Fatalf("writer %v line %v came at position %v", w, i, next[w])
		}
		next[w]++
	}
}

// TestRenderMayLogWhileHeldEntriesDrain records that a render which logs does
// not deadlock the drain, and the entry it logs is written, not lost.
func TestRenderMayLogWhileHeldEntriesDrain(t *testing.T) {
	var ready = setRenderer(t, false)
	var out lockedBuffer
	var l = newTestLogger(&out)

	reentrantLogger.Store(l)
	defer reentrantArmed.Store(false)

	l.Log("%v", astral.NewString32("line 0"))

	reentrantArmed.Store(true)
	ready.Store(true)

	var done = make(chan struct{})
	go func() {
		defer close(done)
		l.Log("%v", astral.NewString32("line 1"))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("draining held entries deadlocked while a view logged")
	}

	if got := out.lines(t, 3); len(got) != 3 {
		t.Fatalf("wrote %v lines, want 3: %q", len(got), got)
	}
}
