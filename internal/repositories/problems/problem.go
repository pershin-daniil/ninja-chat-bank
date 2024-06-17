package problemsrepo

import (
	"time"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type Problem struct {
	ID         types.ProblemID
	ChatID     types.ChatID
	ManagerID  types.UserID
	CreatedAt  time.Time
	ResolvedAt time.Time
}

func adaptStoreProblem(p *store.Problem) Problem {
	return Problem{
		ID:         p.ID,
		ChatID:     p.ChatID,
		ManagerID:  p.ManagerID,
		CreatedAt:  p.CreatedAt,
		ResolvedAt: p.ResolvedAt,
	}
}
