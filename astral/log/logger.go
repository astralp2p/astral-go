package log

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/fmt"
	"github.com/astralp2p/astral-go/sig"
)

// rootQueueCap bounds the rendered lines waiting for the root writer.
//
// note: a console write costs microseconds, so the queue absorbs bursts and
// transient stalls only; a console nobody reads drops.
const rootQueueCap = 1024

// rootBufferSize sizes the buffer the pump writes through.
const rootBufferSize = 64 << 10

// heldCap bounds the entries held back while no renderer is set.
//
// note: only the lines before the first module registers its entry view wait
// here, a few dozen in practice. A full buffer refuses the newest entry and
// counts it, the way the queue does, and the count prints as one line after
// the entries that were kept. The earliest lines are the ones a failed start
// needs, so they are the ones kept. It sits well below rootQueueCap so a full
// drain fits the queue of a writer that is keeping up.
const heldCap = 256

// closeTimeout bounds how long Close waits for a stalled writer.
const closeTimeout = 2 * time.Second

// rendererSet reports whether entries have a registered view. Until it does,
// fmt falls back to a struct dump of *Entry.
//
// note: a variable so a test can run before and after the registration; the
// test binary registers a view for the whole process.
var rendererSet = func() bool { return fmt.HasView(&Entry{}) }

// Logger writes log entries. Child loggers created via SetPrefix/Tag share the
// root's filter, registered EntryLoggers and writer; only the root holds that
// state and only the root runs the writer's pump.
type Logger struct {
	id      *astral.Identity
	w       io.Writer
	mu      sync.Mutex
	parent  *Logger
	prefix  []astral.Object
	filter  func(*Entry) bool
	loggers sig.Set[EntryLogger]

	queue     chan string
	flushc    chan chan struct{} // Close asks the pump to flush through it
	dropped   atomic.Uint64
	firstDrop atomic.Int64 // unix nanoseconds of the oldest unreported drop

	// why: an entry renders only once the node's modules have registered the
	// entry view, so earlier entries wait here in arrival order.
	holdMu      sync.Mutex // guards held, heldDropped, draining and the release of the hold; never held across a render
	held        []*Entry
	heldDropped uint64
	draining    bool       // a goroutine is writing the held entries out
	drainMu     sync.Mutex // one drain at a time; held across renders

	// released is set for good once a drain empties the hold with a renderer
	// set, or by Release/Close. From then on an entry never waits.
	released atomic.Bool
}

// SetFilter sets the filter on the root; an entry is emitted only when filter is
// nil or returns true.
func (l *Logger) SetFilter(filter func(*Entry) bool) {
	if l.parent != nil {
		l.parent.SetFilter(filter)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.filter = filter
}

// EntryLogger receives every entry the root logger emits. LogEntry is called
// synchronously under the root logger mutex and must not block: a subscriber
// with a slow or stalling sink buffers or drops, never waits — a blocked
// LogEntry stops every goroutine that logs. The entry is shared across
// subscribers and must be treated as immutable.
type EntryLogger interface {
	LogEntry(*Entry)
}

// New returns a root logger writing to os.Stdout through a pump goroutine.
//
// note: New is the only constructor that starts the pump; a Logger built as a
// struct literal queues nothing and drops every line.
func New(id *astral.Identity) *Logger {
	var l = &Logger{
		id:     id,
		w:      os.Stdout,
		queue:  make(chan string, rootQueueCap),
		flushc: make(chan chan struct{}),
	}

	go l.pump()

	return l
}

// Release writes out the entries held for a renderer, whether or not one was
// ever set, and stops holding: an entry logged afterwards goes straight to the
// queue. Call it when module loading is over, so a node that never registers
// the entry view, or that fails to start, still prints its early lines.
//
// note: held entries render with whatever view is registered, the struct dump
// of *Entry when none is. A late dump is a lossy line, not a lost one.
func (l *Logger) Release() {
	l.root().drain(true)
}

// Close releases the hold (see Release), then waits, up to closeTimeout, for
// the writer to take everything queued. Call it before the process exits.
//
// note: without Release or Close, and with no renderer ever set, entries stay
// held: the first heldCap are kept in memory, the rest are counted, and
// nothing is written. Close does not stop the logger.
func (l *Logger) Close() {
	var root = l.root()

	root.Release()

	if root.flushc == nil {
		return
	}

	var ack = make(chan struct{})
	var timeout = time.After(closeTimeout)

	select {
	case root.flushc <- ack:
	case <-timeout:
		return
	}

	select {
	case <-ack:
	case <-timeout:
	}
}

func (l *Logger) Log(format string, v ...interface{}) {
	l.logf(0, format, v...)
}

func (l *Logger) Logv(level int, format string, v ...interface{}) {
	l.logf(uint8(level), format, v...)
}

// note: Info/Infov/Error/Errorv carry no severity; they are aliases of Log/Logv.
func (l *Logger) Info(format string, v ...interface{}) {
	l.logf(0, format, v...)
}

func (l *Logger) Infov(level int, format string, v ...interface{}) {
	l.logf(uint8(level), format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.logf(0, format, v...)
}

func (l *Logger) Errorv(level int, format string, v ...interface{}) {
	l.logf(uint8(level), format, v...)
}

// SetPrefix returns a new child logger that prepends obj to every entry; the
// receiver is unchanged.
func (l *Logger) SetPrefix(obj ...astral.Object) *Logger {
	return &Logger{
		parent: l,
		id:     l.id,
		prefix: obj,
	}
}

func (l *Logger) Tag(tag Tag) *Logger {
	return l.SetPrefix(&tag)
}

func (l *Logger) AppendTag(tag Tag) *Logger {
	return l.SetPrefix(append(l.prefix, &tag)...)
}

func (l *Logger) AddLogger(el EntryLogger) {
	l.root().loggers.Add(el)
}

func (l *Logger) RemoveLogger(el EntryLogger) {
	l.root().loggers.Remove(el)
}

func (l *Logger) logf(level uint8, f string, v ...interface{}) {
	var items = l.prefix

	f = strings.ReplaceAll(f, "\n", "\\\\n")

	for _, a := range fmt.Format(f, v...) {
		obj := astral.Adapt(a)
		if obj == nil {
			if view, ok := a.(fmt.View); ok {
				obj = astral.NewString32(view.Render())
			} else {
				obj = astral.NewString32(astral.Stringify(a))
			}
		}
		items = append(items, obj)
	}

	entry := NewEntry(l.id, level, items...)

	l.root().logEntry(entry)
}

func (l *Logger) logEntry(e *Entry) {
	var root = l.root()

	root.mu.Lock()
	var f = root.filter
	root.mu.Unlock()

	// why: the filter gates the writer alone. A subscriber receives every entry
	// regardless of the console's verbosity.
	if f == nil || f(e) {
		// why: a view renders arbitrary code — astrald's mod/log resolves an
		// identity's display name through a database read — so rendering under
		// the root mutex serializes every goroutine that logs behind that read,
		// and lets the render re-enter the non-reentrant mutex.
		// note: one Write per entry; Printer.Printf writes once per token.
		if !root.hold(e) {
			root.enqueue(fmt.Sprintf("%v\n", e))
		}
	}

	root.mu.Lock()
	defer root.mu.Unlock()

	for _, el := range root.loggers.Clone() {
		el.LogEntry(e)
	}
}

// hold keeps e back and reports true when it cannot render yet: no renderer is
// set, or earlier entries are still held or being written out and e must not
// pass them. When the renderer is set and held entries remain, the first caller
// writes them all out in arrival order before returning.
//
// why: once a renderer is set it stays set, so the held entries are the only
// ones that can be ahead of a new entry; with none, an entry takes the
// unchanged path.
func (l *Logger) hold(e *Entry) bool {
	if l.released.Load() {
		return false
	}

	l.holdMu.Lock()

	// note: released is read again under holdMu, where drain sets it, so an
	// entry cannot be appended after the drain emptied the hold for good.
	var ready = l.released.Load() || rendererSet()

	if ready && len(l.held) == 0 && !l.draining {
		l.holdMu.Unlock()
		return false
	}

	if len(l.held) < heldCap {
		l.held = append(l.held, e)
	} else {
		l.heldDropped++
	}

	var start = ready && !l.draining
	l.holdMu.Unlock()

	if start {
		l.drain(false)
	}

	return true
}

// drain renders the held entries and queues them in arrival order, looping
// until none are left, then releases the hold for good. Without force it stops
// while no renderer is set.
//
// why: a render may log (see TestRenderMayReEnterLogger), and an entry logged
// meanwhile finds draining set, joins the held entries and is picked up by the
// next pass, so nothing renders under holdMu and nothing overtakes the batch.
func (l *Logger) drain(force bool) {
	l.drainMu.Lock()
	defer l.drainMu.Unlock()

	for {
		l.holdMu.Lock()

		var ready = force || l.released.Load() || rendererSet()

		if !ready {
			l.draining = false
			l.holdMu.Unlock()
			return
		}

		if len(l.held) == 0 && l.heldDropped == 0 {
			l.draining = false
			l.released.Store(true)
			l.holdMu.Unlock()
			return
		}

		l.draining = true
		var batch, n = l.held, l.heldDropped
		l.held, l.heldDropped = nil, 0
		l.holdMu.Unlock()

		for _, e := range batch {
			l.enqueue(fmt.Sprintf("%v\n", e))
		}

		if n > 0 {
			l.enqueue("log: dropped " + strconv.FormatUint(n, 10) +
				" entries held before the log renderer was set\n")
		}
	}
}

// enqueue hands a rendered line to the pump, dropping it when the queue is
// full.
//
// why: the root writer defaults to os.Stdout, and a full pipe blocks write(2).
// Holding the root mutex across that write stopped every goroutine that logs,
// which is the whole-node freeze an unread subscriber used to cause.
func (l *Logger) enqueue(line string) {
	select {
	case l.queue <- line:
	default:
		if l.dropped.Add(1) == 1 {
			l.firstDrop.Store(time.Now().UnixNano())
		}
	}
}

// pump drains the queue for the process lifetime and is the writer's sole
// user.
//
// note: entries still queued when the process dies without Close are lost. The
// queue empties within microseconds unless the writer is already stalled.
func (l *Logger) pump() {
	var w = bufio.NewWriterSize(l.w, rootBufferSize)

	for {
		select {
		case line := <-l.queue:
			l.reportDrops(w)

			w.WriteString(line)

			// why: a burst coalesces into one write, and the console flushes as
			// soon as the queue drains, so a quiet node never sits on a line.
			if len(l.queue) == 0 {
				w.Flush()
			}

		case ack := <-l.flushc:
			l.drainQueue(w)
			l.reportDrops(w)
			w.Flush()
			close(ack)
		}
	}
}

// drainQueue writes the lines already queued.
func (l *Logger) drainQueue(w *bufio.Writer) {
	for {
		select {
		case line := <-l.queue:
			l.reportDrops(w)
			w.WriteString(line)
		default:
			return
		}
	}
}

// reportDrops writes one line naming the entries the queue refused since the
// last report.
//
// note: the line is built without the formatter, which renders views and would
// re-enter the logger.
func (l *Logger) reportDrops(w *bufio.Writer) {
	var n = l.dropped.Swap(0)
	if n == 0 {
		return
	}

	var since = time.Unix(0, l.firstDrop.Swap(0))

	w.WriteString("log: dropped " + strconv.FormatUint(n, 10) +
		" entries since " + since.Format(time.RFC3339) + "\n")
}

func (l *Logger) root() *Logger {
	if l.parent != nil {
		return l.parent.root()
	}
	return l
}
