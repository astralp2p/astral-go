package messaging

// ListMessagesRequest names one of the caller's lists and how to narrow it.
// It is carried as MethodListMessages arguments, not as an object.
//
// List is ListInbox, ListOutbox or ListArchive; empty reads as ListInbox. From
// and To narrow to one correspondent by hex identity or alias: From on the
// inbox, To on the outbox. Since pages the inbox only: it is a previous
// answer's cursor, and 0 pages from the start. UnreadOnly narrows the inbox to
// bodies never handed out, and AwaitingPickup the outbox to deliveries that
// landed and were never collected. A narrowing a list cannot apply is refused,
// never ignored.
type ListMessagesRequest struct {
	List, From, To             string
	Since                      uint64
	UnreadOnly, AwaitingPickup bool
}

// NextSince is the Since that pages past list: the furthest Cursor in it, or
// since itself when nothing in list is newer. A caller passing it back sees
// only rows written after.
//
// why since and not zero when nothing is newer: a caller that stores the
// answer as its position would otherwise rewind to the start of the inbox.
func NextSince(list []*Envelope, since uint64) uint64 {
	furthest := since
	for _, m := range list {
		// why a nil row is skipped: the list may have crossed the wire, and a
		// slice element is a pointer a peer may leave unset.
		if m != nil && uint64(m.Cursor) > furthest {
			furthest = uint64(m.Cursor)
		}
	}
	return furthest
}
