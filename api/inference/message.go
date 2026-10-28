package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Message is one turn of a conversation.
type Message struct {
	// Role is RoleSystem, RoleUser, RoleAssistant or RoleTool.
	Role astral.String8
	// Parts may be empty on an assistant turn that only calls tools.
	Parts []*ContentPart
	// ToolCalls lists the calls an assistant turn made. Assistant turns only.
	ToolCalls []*ToolCall
	// ToolCallID names the call a tool turn answers. Tool turns only.
	ToolCallID astral.String8
}

func (Message) ObjectType() string { return "inference.message" }

func (v Message) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Message) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Message) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Message) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Message{}) }
