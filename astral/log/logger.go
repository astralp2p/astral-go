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
	dropped   atomic.Uint64
	firstDrop atomic.Int64 // unix nanoseconds of the oldest unreported drop
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
		id:    id,
		w:     os.Stdout,
		queue: make(chan string, rootQueueCap),
	}

	go l.pump()

	return l
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
		root.enqueue(fmt.Sprintf("%v\n", e))
	}

	root.mu.Lock()
	defer root.mu.Unlock()

	for _, el := range root.loggers.Clone() {
		el.LogEntry(e)
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
// note: entries still queued when the process dies are lost. The queue empties
// within microseconds unless the writer is already stalled.
func (l *Logger) pump() {
	var w = bufio.NewWriterSize(l.w, rootBufferSize)

	for line := range l.queue {
		l.reportDrops(w)

		w.WriteString(line)

		// why: a burst coalesces into one write, and the console flushes as
		// soon as the queue drains, so a quiet node never sits on a line.
		if len(l.queue) == 0 {
			w.Flush()
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
