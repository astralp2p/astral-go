package mcp

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
	"github.com/google/uuid"
)

var errInvalidMessageID = errors.New("invalid message id")

// MessageID names one message. The sender mints it, and it is the message's
// name on both sides: the recipient reads by it, and a delivery that arrives
// twice collides on it and is stored once. The zero value names no message.
//
// why 128 bits: an inbox keeps a message and a reply names it long after
// delivery, so the identifier competes against every message a node has stored
// rather than against the ones in flight. 64 bits reaches a one-in-a-million
// collision at six million messages, which a node outlives. A mint is a uuid
// v7: 48 of the 128 carry a millisecond timestamp, 4 the version, 2 the
// variant, and 12 a sequence counting the mints one process makes within one
// millisecond. The remaining 62 are random, and they are what separates two
// mints of one millisecond made by different processes.
type MessageID [16]byte

// NewMessageID mints a MessageID from a uuid v7. The timestamp leads the value,
// so a process's mints sort in the order it made them, as bytes and as the hex
// String writes alike. Two processes share no counter, so mints of one
// millisecond made by different senders order arbitrarily between themselves.
//
// A failing random source panics rather than answering, which is what reading
// crypto/rand did here before: that read never returns an error and crashes the
// program instead. The zero value names no message, so a mint that answered it
// on a failure would give a message the name of the absence of one.
func NewMessageID() (id MessageID) {
	u := uuid.Must(uuid.NewV7())
	copy(id[:], u[:])
	return
}

// ParseMessageID reads a MessageID from the hex form String writes.
func ParseMessageID(s string) (id MessageID, err error) {
	if len(s) != hex.EncodedLen(len(id)) {
		return id, errInvalidMessageID
	}
	if _, err = hex.Decode(id[:], []byte(s)); err != nil {
		return id, errInvalidMessageID
	}
	return
}

// IsZero reports the identifier that names no message.
func (id MessageID) IsZero() bool {
	return id == MessageID{}
}

func (id MessageID) String() string {
	return hex.EncodeToString(id[:])
}

// astral

var _ astral.Object = &MessageID{}

func (MessageID) ObjectType() string { return "mcp.message_id" }

func (id MessageID) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(id[:])
	return int64(m), err
}

func (id *MessageID) ReadFrom(r io.Reader) (n int64, err error) {
	m, err := io.ReadFull(r, id[:])
	return int64(m), err
}

// json

func (id MessageID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *MessageID) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	parsed, err := ParseMessageID(s)
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// text

func (id MessageID) MarshalText() (text []byte, err error) {
	return []byte(id.String()), nil
}

func (id *MessageID) UnmarshalText(text []byte) error {
	parsed, err := ParseMessageID(string(text))
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// sql

func (id MessageID) Value() (driver.Value, error) {
	return id.String(), nil
}

// Scan reads the hex form from either shape a driver hands a text column.
func (id *MessageID) Scan(src any) error {
	switch v := src.(type) {
	case string:
		return id.UnmarshalText([]byte(v))
	case []byte:
		return id.UnmarshalText(v)
	default:
		return errInvalidMessageID
	}
}

func init() {
	var id MessageID
	astral.MustAdd(&id)
}
