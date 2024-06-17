package problemsrepo

import (
	"context"
	"fmt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/problem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error) {
	p, err := r.db.Problem(ctx).Query().Where(problem.ChatID(chatID), problem.ResolvedAtIsNil()).First(ctx)
	if nil == err {
		return p.ID, nil
	}

	if !store.IsNotFound(err) {
		return types.ProblemIDNil, fmt.Errorf("failed to query: %v", err)
	}

	p, err = r.db.Problem(ctx).Create().SetChatID(chatID).Save(ctx)
	if err != nil {
		return types.ProblemIDNil, fmt.Errorf("failed to create problem: %v", err)
	}

	return p.ID, nil
}

func (r *Repo) GetManagerOpenProblemsCount(ctx context.Context, managerID types.UserID) (int, error) {
	count, err := r.db.Problem(ctx).Query().Where(
		problem.ManagerID(managerID),
		problem.ResolvedAtIsNil(),
	).Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count manager's problems: %v", err)
	}

	return count, nil
}

func (r *Repo) GetUnresolvedProblem(ctx context.Context, chatID types.ChatID, managerID types.UserID) (*Problem, error) {
	p, err := r.db.Problem(ctx).
		Query().
		Where(
			problem.HasChatWith(chat.ID(chatID)),
			problem.ManagerID(managerID),
			problem.ResolvedAtIsNil(),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("get problem: %v", err)
	}
	if nil == p {
		return nil, ErrProblemNotFound
	}

	return pointer.Ptr(adaptStoreProblem(p)), nil
}
