package problemsrepo

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store/problem"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	"github.com/pershin-daniil/ninja-chat-bank/pkg/pointer"
)

func (r *Repo) ProblemsWithoutManager(ctx context.Context, limit int) ([]types.ProblemID, error) {
	ids, err := r.db.Problem(ctx).
		Query().
		Where(
			problem.ManagerIDIsNil(),
			problem.ResolvedAtIsNil(),
			problem.HasMessagesWith(
				message.IsVisibleForManager(true),
			),
		).
		Order(problem.ByCreatedAt(sql.OrderAsc())).
		Limit(limit).
		IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("query problems: %v", err)
	}

	return ids, nil
}

func (r *Repo) AssignManagerToProblem(ctx context.Context, managerID types.UserID, problemID types.ProblemID) (*Problem, error) {
	p, err := r.db.Problem(ctx).UpdateOneID(problemID).SetManagerID(managerID).Save(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("update problem: %v", err)
	}

	return pointer.Ptr(adaptStoreProblem(p)), nil
}

func (r *Repo) ProblemInitialMessageRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error) {
	msg, err := r.db.Message(ctx).
		Query().
		Where(
			message.HasChatWith(chat.HasProblemsWith(problem.ID(problemID))),
			message.IsVisibleForManager(true),
		).
		First(ctx)
	if err != nil {
		return types.RequestIDNil, fmt.Errorf("find message: %v", err)
	}

	return msg.InitialRequestID, nil
}

func (r *Repo) GetProblemByID(ctx context.Context, id types.ProblemID) (*Problem, error) {
	msg, err := r.db.Problem(ctx).Get(ctx, id)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, ErrProblemNotFound
		}
		return nil, fmt.Errorf("get problem: %v", err)
	}

	return pointer.Ptr(adaptStoreProblem(msg)), nil
}
