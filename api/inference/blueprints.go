package inference

import (
	"fmt"

	"github.com/astralp2p/astral-go/astral"
)

// Objects returns a zero value of every inference type, each after the types
// it references, the order a node registers their blueprints in.
func Objects() []astral.Object {
	return []astral.Object{
		&EffortLevel{}, &Model{}, &Models{}, &Provider{},
		&Tool{}, &ToolCall{}, &MediaRef{}, &ContentPart{}, &Message{}, &Request{},
		&ToolCallDelta{}, &Chunk{}, &Usage{}, &Error{},
	}
}

// Blueprints derives the blueprint of every inference type, in the order of
// Objects.
//
// why: a node that hosts no inference app knows none of these types, and it
// cannot decode an offering or an answer that crosses it until an app pushes
// them with objects.register_blueprint.
func Blueprints() ([]*astral.Blueprint, error) {
	objects := Objects()
	list := make([]*astral.Blueprint, 0, len(objects))
	for _, o := range objects {
		bp, err := astral.BlueprintOf(o)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", o.ObjectType(), err)
		}
		list = append(list, bp)
	}
	return list, nil
}
