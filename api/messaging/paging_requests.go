package messaging

// The paging requests are carried as operation arguments, not as objects.
//
// Mailbox names the mailbox read, as on ListMessagesRequest: empty or the
// caller's own identity reads the caller's own, and any other identity is a
// delegated read. Limit 0 takes PageLimitDefault; a Limit over PageLimitMax is
// refused, never clamped.
//
// Generation is the generation the answer that gave the position carried. It
// is a pointer where a request may start a traversal without one: nil sends no
// generation, and a node answers a request that holds a position without one
// as malformed.

// PageMessagesRequest reads one page newest first: one list, or one peer's
// unarchived rows in both boxes. List and Peer are exclusive; both empty reads
// the inbox. Before 0 reads the newest page.
type PageMessagesRequest struct {
	List, Peer, Mailbox string
	Before, Limit       uint64
	Generation          *uint64
}

// ListMessageChangesRequest reads what changed after Since, narrowed to one peer
// when Peer is set. Since 0 reads from the start of the order.
type ListMessageChangesRequest struct {
	Peer, Mailbox string
	Since, Limit  uint64
	Generation    uint64
}

// PageConversationsRequest reads one page of conversations newest first, or the
// one conversation with Peer. Before and Peer are exclusive.
type PageConversationsRequest struct {
	Peer, Mailbox string
	Before, Limit uint64
	Generation    *uint64
}

// ListConversationChangesRequest reads the conversations that changed after
// Since.
type ListConversationChangesRequest struct {
	Mailbox      string
	Since, Limit uint64
	Generation   uint64
}

// ErrGeneration is the error a node answers a position from another generation
// of the mailbox: the rows it named were deleted.
const ErrGeneration = "generation changed: the mailbox was deleted since this position was read"
