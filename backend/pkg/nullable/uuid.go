package nullable

import (
	"encoding/json"

	"github.com/google/uuid"
)

// UUID wraps *uuid.UUID with a custom JSON unmarshaler that distinguishes
// JSON null from a missing field.
//
//   - JSON {"id": null} -> UUID{Valid: true, UUID: nil}  (explicit null)
//   - JSON {"id": "..."} -> UUID{Valid: true, UUID: &id} (present)
//   - JSON {} -> UUID{Valid: false}                       (omitted)
type UUID struct {
	Valid bool
	UUID  *uuid.UUID
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	u.Valid = true

	if string(data) == "null" {
		u.UUID = nil
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	u.UUID = &id
	return nil
}
