//go:build integration

package problemsrepo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	"github.com/pershin-daniil/ninja-chat-bank/internal/store"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type ProblemsScheduleRepoSuite struct {
	testingh.DBSuite
	repo *problemsrepo.Repo
}

func TestProblemsScheduleRepoSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProblemsScheduleRepoSuite{DBSuite: testingh.NewDBSuite("TestProblemsScheduleRepoSuite")})
}

func (s *ProblemsScheduleRepoSuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = problemsrepo.New(problemsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ProblemsScheduleRepoSuite) SetupSubTest() {
	s.removeOldData()
}

func (s *ProblemsScheduleRepoSuite) Test_ProblemsWithoutManager() {
	s.Run("problem with manager should be skipped", func() {
		c := s.createChat(types.NewUserID())
		s.createProblemForManager(c.ID, types.NewUserID())

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 1)

		s.Require().NoError(err)
		s.Empty(problemIDs)
	})

	s.Run("problem with no messages should be skipped", func() {
		c := s.createChat(types.NewUserID())
		s.createProblem(c.ID)

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 1)

		s.Require().NoError(err)
		s.Empty(problemIDs)
	})

	s.Run("problem with no messages visible for manager should be skipped", func() {
		c := s.createChat(types.NewUserID())
		p := s.createProblem(c.ID)

		// Create message
		err := s.Database.Message(s.Ctx).Create().
			SetChatID(c.ID).
			SetProblemID(p.ID).
			SetInitialRequestID(types.NewRequestID()).
			SetIsVisibleForManager(false).
			SetBody("Hello").
			Exec(s.Ctx)
		s.Require().NoError(err)

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 1)

		s.Require().NoError(err)
		s.Empty(problemIDs)
	})

	s.Run("resolved problem should be skipped", func() {
		c := s.createChat(types.NewUserID())
		// Create resolved problem
		p, err := s.Database.Problem(s.Ctx).Create().SetChatID(c.ID).SetResolvedAt(time.Now()).Save(s.Ctx)
		s.Require().NoError(err)
		s.createMessageVisibleForManager(c.ID, p.ID)

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 1)

		s.Require().NoError(err)
		s.Empty(problemIDs)
	})

	s.Run("unresolved problem should be returned", func() {
		c := s.createChat(types.NewUserID())
		p := s.createProblem(c.ID)
		s.createMessageVisibleForManager(c.ID, p.ID)

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 1)

		s.Require().NoError(err)
		s.Equal([]types.ProblemID{p.ID}, problemIDs)
	})

	s.Run("problems should be ordered by create time", func() {
		// Create chats
		chat1 := s.createChat(types.NewUserID())
		chat2 := s.createChat(types.NewUserID())

		// Create problems
		problem1 := s.createProblem(chat1.ID)
		problem2 := s.createProblem(chat2.ID)

		// Create messages
		s.createMessageVisibleForManager(chat1.ID, problem1.ID)
		// time.After(100) // wait
		s.createMessageVisibleForManager(chat2.ID, problem2.ID)

		problemIDs, err := s.repo.ProblemsWithoutManager(s.Ctx, 2)
		s.Require().NoError(err)
		s.Equal([]types.ProblemID{problem1.ID, problem2.ID}, problemIDs)
	})
}

func (s *ProblemsScheduleRepoSuite) Test_AssignManagerToProblem() {
	s.Run("problem not found", func() {
		managerID := types.NewUserID()
		problemID := types.NewProblemID()

		_, err := s.repo.AssignManagerToProblem(s.Ctx, managerID, problemID)

		s.Require().ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})

	s.Run("manager assigned", func() {
		managerID := types.NewUserID()
		c := s.createChat(types.NewUserID())
		p := s.createProblem(c.ID)

		pr, err := s.repo.AssignManagerToProblem(s.Ctx, managerID, p.ID)

		s.Require().NoError(err)
		s.Equal(managerID, pr.ManagerID)
	})
}

func (s *ProblemsScheduleRepoSuite) Test_ProblemInitialMessageRequestID() {
	c := s.createChat(types.NewUserID())
	p := s.createProblem(c.ID)

	s.createMessageInvisibleForManager(c.ID, p.ID)
	message := s.createMessageVisibleForManager(c.ID, p.ID)
	s.createMessageVisibleForManager(c.ID, p.ID)

	requestID, err := s.repo.ProblemInitialMessageRequestID(s.Ctx, p.ID)

	s.Require().NoError(err)
	s.Equal(message.InitialRequestID, requestID)
}

func (s *ProblemsScheduleRepoSuite) createMessageVisibleForManager(
	chatID types.ChatID,
	problemID types.ProblemID,
) *store.Message {
	msg, err := s.Database.Message(s.Ctx).
		Create().
		SetChatID(chatID).
		SetInitialRequestID(types.NewRequestID()).
		SetIsVisibleForManager(true).
		SetProblemID(problemID).
		SetBody("Hello").
		Save(s.Ctx)

	s.Require().NoError(err)

	return msg
}

func (s *ProblemsScheduleRepoSuite) createMessageInvisibleForManager(
	chatID types.ChatID,
	problemID types.ProblemID,
) *store.Message {
	msg, err := (s.Database.Message(s.Ctx).
		Create().
		SetChatID(chatID)).
		SetInitialRequestID(types.NewRequestID()).
		SetIsVisibleForManager(false).
		SetProblemID(problemID).
		SetBody("My CVC is 123").
		Save(s.Ctx)

	s.Require().NoError(err)

	return msg
}

func (s *ProblemsScheduleRepoSuite) removeOldData() {
	// Delete messages
	_, err := s.Database.Message(s.Ctx).Delete().Exec(s.Ctx)
	s.Require().NoError(err)

	// Delete problems
	_, err = s.Database.Problem(s.Ctx).Delete().Exec(s.Ctx)
	s.Require().NoError(err)

	// Delete chats
	_, err = s.Database.Chat(s.Ctx).Delete().Exec(s.Ctx)
	s.Require().NoError(err)
}

func (s *ProblemsScheduleRepoSuite) createChat(clientID types.UserID) *store.Chat {
	c, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
	s.Require().NoError(err)

	return c
}

func (s *ProblemsScheduleRepoSuite) createProblem(chatID types.ChatID) *store.Problem {
	problem, err := s.Database.Problem(s.Ctx).Create().SetChatID(chatID).Save(s.Ctx)
	s.Require().NoError(err)

	return problem
}

func (s *ProblemsScheduleRepoSuite) createProblemForManager(chatID types.ChatID, managerID types.UserID) *store.Problem {
	problem, err := s.Database.Problem(s.Ctx).Create().
		SetChatID(chatID).
		SetManagerID(managerID).
		SetResolvedAt(time.Now()).Save(s.Ctx)
	s.Require().NoError(err)

	return problem
}
