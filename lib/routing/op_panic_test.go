package routing

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// A panicking op must not take the process down: the goroutine RouteQuery spawns is
// the root of its own stack, so nothing outside it can recover.

type panicOps struct{ sentinel error }

func (o panicOps) Boom(_ *astral.Context, _ *IncomingQuery) error {
	panic("boom")
}

func (o panicOps) BoomError(_ *astral.Context, _ *IncomingQuery) error {
	panic(o.sentinel)
}

func (o panicOps) AcceptThenBoom(_ *astral.Context, q *IncomingQuery) error {
	q.AcceptRaw()
	panic("boom after accept")
}

func (o panicOps) RejectThenBoom(_ *astral.Context, q *IncomingQuery) error {
	q.RejectWithCode(astral.CodeInvalidQuery)
	panic("boom after reject")
}

// routerWithReports mounts panicOps and captures every Report the ops produce.
func routerWithReports(t *testing.T) (*OpRouter, chan *Report) {
	t.Helper()

	reports := make(chan *Report, 1)
	router := NewOpRouter()
	if err := router.AddStruct(&panicOps{sentinel: errSentinel}); err != nil {
		t.Fatalf("AddStruct: %v", err)
	}

	for _, spec := range router.Spec() {
		op, err := router.GetOp(spec.Name)
		if err != nil {
			t.Fatalf("GetOp(%q): %v", spec.Name, err)
		}
		op.LogFunc = func(r *Report) { reports <- r }
	}

	return router, reports
}

var errSentinel = errors.New("sentinel")

func TestOp_PanicIsRejectedAsInternalError(t *testing.T) {
	router, _ := routerWithReports(t)

	w, err := routeTo(t, router, "boom")

	if w != nil {
		t.Errorf("want no writer, got %v", w)
	}

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want *astral.ErrRejected, got %v", err)
	}
	if rejected.Code != astral.CodeInternalError {
		t.Errorf("want code %d, got %d", astral.CodeInternalError, rejected.Code)
	}
}

func TestOp_PanicReportsErrPanicWithStack(t *testing.T) {
	router, reports := routerWithReports(t)

	routeTo(t, router, "boom")

	r := <-reports

	var p *ErrPanic
	if !errors.As(r.Err, &p) {
		t.Fatalf("want *ErrPanic on the report, got %v", r.Err)
	}
	if p.Value != "boom" {
		t.Errorf("want panic value %q, got %v", "boom", p.Value)
	}
	if !strings.Contains(string(p.Stack), "panicOps") {
		t.Errorf("want the panicking function in the stack, got:\n%s", p.Stack)
	}
	if r.Time <= 0 {
		t.Error("want a non-zero duration on the report")
	}
}

func TestOp_PanicUnwrapsAnErrorValue(t *testing.T) {
	router, reports := routerWithReports(t)

	routeTo(t, router, "boom_error")

	r := <-reports

	if !errors.Is(r.Err, errSentinel) {
		t.Errorf("want errors.Is to reach the panicked error, got %v", r.Err)
	}
}

// An op that accepted before panicking keeps its connection: the recovery must not
// downgrade a live query to a rejection.
func TestOp_PanicAfterAcceptKeepsTheConnection(t *testing.T) {
	router, reports := routerWithReports(t)

	w, err := routeTo(t, router, "accept_then_boom")

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if w == nil {
		t.Fatal("want a writer, got nil")
	}

	r := <-reports
	var p *ErrPanic
	if !errors.As(r.Err, &p) {
		t.Errorf("want *ErrPanic on the report, got %v", r.Err)
	}
}

// An op that rejected with its own code before panicking keeps that code: the CAS in
// RejectWithCode is what guarantees the first resolve wins.
func TestOp_PanicAfterRejectKeepsTheOpsCode(t *testing.T) {
	router, _ := routerWithReports(t)

	_, err := routeTo(t, router, "reject_then_boom")

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want *astral.ErrRejected, got %v", err)
	}
	if rejected.Code != astral.CodeInvalidQuery {
		t.Errorf("want the op's own code %d, got %d", astral.CodeInvalidQuery, rejected.Code)
	}
}

// The survival assertion. Asserting "the process did not die" in-process is
// trivially true when the recovery works and takes the whole test binary down when it
// does not, so the routing runs in a re-executed child: a regression then fails this
// one named test instead of crashing every test in the package.
func TestOp_PanickingOpDoesNotKillTheProcess(t *testing.T) {
	const marker = "SURVIVED"

	if os.Getenv("ASTRAL_TEST_PANICKING_OP") == "1" {
		router, reports := routerWithReports(t)
		routeTo(t, router, "boom")

		// bound the wait: a child that blocks would otherwise hang the parent's
		// CombinedOutput and report as a timeout rather than as this assertion.
		select {
		case <-reports:
		case <-time.After(10 * time.Second):
			t.Fatal("no report from the panicking op")
		}

		os.Stdout.WriteString(marker + "\n")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestOp_PanickingOpDoesNotKillTheProcess$")
	cmd.Env = append(os.Environ(), "ASTRAL_TEST_PANICKING_OP=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child process died on a panicking op: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), marker) {
		t.Errorf("child did not reach the marker:\n%s", out)
	}
}
