package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

// jobMaxAttempts is some limit as protection from endless retries of outbox jobs.
const jobMaxAttempts = 30

type Job struct {
	ent.Schema
}

func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", types.JobID{}).Default(types.NewJobID).Unique().Immutable(),
		field.Text("name").NotEmpty().Immutable(),
		field.Text("payload").NotEmpty().Immutable(),
		field.Int("attempts").Max(jobMaxAttempts).Default(0),
		field.Time("available_at").Immutable(),
		field.Time("reserved_until").Default(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Job) Indexes() []ent.Index {
	return nil
}

type FailedJob struct {
	ent.Schema
}

func (FailedJob) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", types.FailedJobID{}).Default(types.NewFailedJobID).Unique().Immutable(),
		field.Text("name").NotEmpty().Immutable(),
		field.Text("payload").NotEmpty().Immutable(),
		field.Text("reason").NotEmpty().Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
