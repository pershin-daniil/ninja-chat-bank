//go:build integration

package chatsrepo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type ChatsRepoSuite struct {
	testingh.DBSuite
	repo *chatsrepo.Repo
}

func TestChatsRepoSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ChatsRepoSuite{DBSuite: testingh.NewDBSuite("TestChatsRepoSuite")})
}

func (s *ChatsRepoSuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = chatsrepo.New(chatsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ChatsRepoSuite) Test_CreateIfNotExists() {
	s.Run("chat does not exist, should be created", func() {
		clientID := types.NewUserID()

		chatID, err := s.repo.CreateIfNotExists(s.Ctx, clientID)
		s.Require().NoError(err)
		s.NotEmpty(chatID)
	})

	s.Run("chat already exists", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		chatID, err := s.repo.CreateIfNotExists(s.Ctx, clientID)
		s.Require().NoError(err)
		s.Require().NotEmpty(chatID)
		s.Equal(chat.ID, chatID)
	})
}

func (s *ChatsRepoSuite) Test_GetOpenProblemChatsForManager() {
	s.Run("manager has open problem chats", func() {
		managerID := types.NewUserID()
		chatID1 := s.createChatWithProblem(managerID)
		chatID2 := s.createChatWithProblem(managerID)

		chats, err := s.repo.GetOpenProblemChatsForManager(s.Ctx, managerID)
		chatIDs := make([]types.ChatID, len(chats))
		for i, chat := range chats {
			chatIDs[i] = chat.ID
		}

		s.Require().NoError(err)
		s.Equal([]types.ChatID{chatID2, chatID1}, chatIDs)
	})

	s.Run("manager has resolved problem chats", func() {
		managerID := types.NewUserID()
		s.createChatWithResolvedProblem(managerID)
		s.createChatWithResolvedProblem(managerID)

		chtIDs, err := s.repo.GetOpenProblemChatsForManager(s.Ctx, managerID)

		s.Require().NoError(err)
		s.Empty(chtIDs)
	})

	s.Run("another manager has open problem chats", func() {
		managerID := types.NewUserID()
		s.createChatWithProblem(types.NewUserID())
		s.createChatWithProblem(types.NewUserID())

		chtIDs, err := s.repo.GetOpenProblemChatsForManager(s.Ctx, managerID)

		s.Require().NoError(err)
		s.Empty(chtIDs)
	})

	s.Run("another manager has resolved problem chats", func() {
		managerID := types.NewUserID()
		s.createChatWithResolvedProblem(types.NewUserID())
		s.createChatWithResolvedProblem(types.NewUserID())

		chtIDs, err := s.repo.GetOpenProblemChatsForManager(s.Ctx, managerID)

		s.Require().NoError(err)
		s.Empty(chtIDs)
	})

	s.Run("no problem in chat", func() {
		managerID := types.NewUserID()
		s.createChat()
		s.createChat()

		chtIDs, err := s.repo.GetOpenProblemChatsForManager(s.Ctx, managerID)

		s.Require().NoError(err)
		s.Empty(chtIDs)
	})
}

func (s *ChatsRepoSuite) createChatWithProblem(managerID types.UserID) types.ChatID {
	chatID := s.createChat()
	s.createProblem(chatID, managerID)
	return chatID
}

func (s *ChatsRepoSuite) createChatWithResolvedProblem(managerID types.UserID) {
	chatID := s.createChat()
	s.createResolvedProblem(chatID, managerID)
}

func (s *ChatsRepoSuite) createChat() types.ChatID {
	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(types.NewUserID()).Save(s.Ctx)
	s.Require().NoError(err)

	return chat.ID
}

func (s *ChatsRepoSuite) createProblem(chatID types.ChatID, managerID types.UserID) types.ProblemID {
	problem, err := s.Database.Problem(s.Ctx).Create().SetChatID(chatID).SetManagerID(managerID).Save(s.Ctx)
	s.Require().NoError(err)

	return problem.ID
}

func (s *ChatsRepoSuite) createResolvedProblem(chatID types.ChatID, managerID types.UserID) types.ProblemID {
	problem, err := s.Database.Problem(s.Ctx).
		Create().
		SetChatID(chatID).
		SetManagerID(managerID).
		SetResolvedAt(time.Now()).
		Save(s.Ctx)
	s.Require().NoError(err)

	return problem.ID
}
