package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Envelope is a StoredMessage without its body: the row a listing, a wait or a
// read that withholds bodies answers. Its fields are the record's, in the
// record's order, less Content.
//
// why a second type and not a StoredMessage with Content left empty: a reader
// cannot tell an empty body from a withheld one, and handing a body out stamps
// the message read. A type that carries no body cannot hand one out.
type Envelope struct {
	Cursor          astral.Uint64
	ID              MessageID
	Box             astral.String8
	Sender          *astral.Identity
	Recipient       *astral.Identity
	ParentID        MessageID
	CreatedAt       astral.Time
	ArchivedAt      *astral.Time
	ReadAt          *astral.Time
	ReceiptDueAt    *astral.Time
	ReceiptStoredAt *astral.Time
	LandedAt        *astral.Time
	FailedAt        *astral.Time
	FetchedAt       *astral.Time
	Err             *astral.String16
}

// Envelope answers the record without its body. It shares the record's
// pointers, so a stamp set through one is seen through the other.
func (m *StoredMessage) Envelope() *Envelope {
	if m == nil {
		return nil
	}
	return &Envelope{
		Cursor:          m.Cursor,
		ID:              m.ID,
		Box:             m.Box,
		Sender:          m.Sender,
		Recipient:       m.Recipient,
		ParentID:        m.ParentID,
		CreatedAt:       m.CreatedAt,
		ArchivedAt:      m.ArchivedAt,
		ReadAt:          m.ReadAt,
		ReceiptDueAt:    m.ReceiptDueAt,
		ReceiptStoredAt: m.ReceiptStoredAt,
		LandedAt:        m.LandedAt,
		FailedAt:        m.FailedAt,
		FetchedAt:       m.FetchedAt,
		Err:             m.Err,
	}
}

// astral

var _ astral.Object = &Envelope{}

func (e Envelope) ObjectType() string { return "messaging.envelope" }

func (e Envelope) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *Envelope) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

// json

func (e Envelope) MarshalJSON() ([]byte, error) {
	type alias Envelope
	return json.Marshal(alias(e))
}

func (e *Envelope) UnmarshalJSON(bytes []byte) error {
	type alias Envelope
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*e = Envelope(v)
	return nil
}

func init() {
	astral.MustAdd(&Envelope{})
}
