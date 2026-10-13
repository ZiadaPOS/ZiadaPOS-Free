package uuidx

import (
	"fmt"

	"github.com/google/uuid"
)

// Parse extracts a uuid from values that may arrive through event payloads
// (map[string]interface{}) where the same key can be a string, a uuid.UUID or
// any type implementing fmt.Stringer. Returns nil when no valid id is found.
func Parse(v interface{}) *uuid.UUID {
	var s string
	switch t := v.(type) {
	case uuid.UUID:
		id := t
		return &id
	case string:
		s = t
	case fmt.Stringer:
		s = t.String()
	default:
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}
