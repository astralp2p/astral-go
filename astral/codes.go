package astral

const (
	CodeSuccess = iota
	CodeRejected
	CodeInvalidQuery
	CodeCanceled
	CodeInternalError
)

// note: 5 is not a generic code; a messaging node that predates ErrNotAdmitted sends it as RejectNotAdmitted.
const (
	// CodeTimeout - the query was not answered in time (an HTTP 504).
	CodeTimeout = 6
	// CodeUnavailable - the target or the next hop could not be reached (an HTTP 502 or 503).
	CodeUnavailable = 7
)

// DefaultRejectCode is the code returned when a query is rejected without an explicit
// reason; mirrors CodeRejected.
const DefaultRejectCode = 1
