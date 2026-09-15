package astral

import (
	"fmt"
	"reflect"
	"sync"
)

// fieldShapes caches checkFieldShapes per struct type. A nil value records a struct whose
// every field is describable.
var fieldShapes sync.Map // reflect.Type -> error

// checkFieldShapes returns ErrUndescribableField naming the first exported field of the
// struct type t that no Blueprint can describe, or nil.
//
// why: the reflection codec and BlueprintFromType read the same struct, so they accept the
// same fields. The codec encoded plain Go kinds — string, bool, uint8, []byte, an embedded
// plain struct — that BlueprintFromType rejects, so a registered type could send bytes no
// peer could decode from its Blueprint. The verdict comes from specFromType itself, so the
// two sides cannot drift apart again.
func checkFieldShapes(t reflect.Type) error {
	if v, ok := fieldShapes.Load(t); ok {
		err, _ := v.(error)
		return err
	}

	err := firstUndescribableField(t)
	fieldShapes.Store(t, err)
	return err
}

// firstUndescribableField skips the same fields BlueprintFromType skips: unexported fields
// and the embedded EmptyObject marker.
func firstUndescribableField(t reflect.Type) error {
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() || sf.Type == emptyObjectReflectType {
			continue
		}
		if _, err := specFromType(sf.Type); err != nil {
			return fmt.Errorf("%w: %s.%s: %v", ErrUndescribableField, t, sf.Name, err)
		}
	}
	return nil
}
