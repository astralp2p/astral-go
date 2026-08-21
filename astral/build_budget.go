package astral

// MaxBlueprintNodes caps the total RuntimeObject frames one construction allocates.
//
// why: MaxBlueprintDepth bounds the stack but not the work. A Blueprint carrying k
// reference fields per level costs k^depth frames, so 27 levels of a two-field
// Blueprint — 27 registrations, every one of them valid and acyclic — makes a single
// New() allocate 2^27 objects. Registration is reachable by a peer, so a depth cap
// alone leaves an amplified denial of service; the node budget converts it into
// ErrDepthExceeded at a fixed cost.
const MaxBlueprintNodes = 4096

// buildBudget is the per-construction node allowance. It is held by pointer and shared
// across the whole construction so sibling branches draw on one budget, while depth
// stays a value parameter so it unwinds on return.
type buildBudget struct{ left int }

func newBuildBudget() *buildBudget { return &buildBudget{left: MaxBlueprintNodes} }

// take charges one frame. A nil budget is unmetered, which keeps the internal
// constructors usable from paths that do not start at a public entry point.
func (b *buildBudget) take() bool {
	if b == nil {
		return true
	}
	if b.left <= 0 {
		return false
	}
	b.left--
	return true
}
