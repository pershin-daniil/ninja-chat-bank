package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

func newCreateAtField() ent.Field {
	return field.Time("created_at").
		Default(time.Now).
		Immutable()
}
