package mcp

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

var errInvalidMessageID = errors.New("invalid message id")

// MessageID names one message. The sender mints it, and it is the message's
// name on both sides: the recipient reads by it, and a delivery that arrives
// twice collides on it and is stored once. The zero value names no message.
//
// why 128 bits: an inbox keeps a message and a reply names it long after
// delivery, so the identifier competes against every message a node has stored
// rather than against the ones in flight. 64 bits reaches a one-in-a-million
// collision at six million messages, which a node outlives.
type MessageID [16]byte

// NewMessageID mints a random MessageID.
func NewMessageID() (id MessageID) {
	_, _ = rand.Read(id[:])
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
