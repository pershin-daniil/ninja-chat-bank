package getchats_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	chatsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/chats"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	getchats "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/get-chats"
	getchatsmocks "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/get-chats/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl      *gomock.Controller
	chatsRepo *getchatsmocks.MockchatsRepository
	uCase     getchats.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.chatsRepo = getchatsmocks.NewMockchatsRepository(s.ctrl)

	var err error
	s.uCase, err = getchats.New(getchats.NewOptions(s.chatsRepo))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestHandle_RequestValidationError() {
	// Arrange.
	req := getchats.Request{}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Require().ErrorIs(err, getchats.ErrInvalidRequest)
	s.Empty(resp.Chats)
}

func (s *UseCaseSuite) TestHandle_ChatsRepoError() {
	// Arrange.
	req := getchats.Request{ID: types.NewRequestID(), ManagerID: types.NewUserID()}
	errChatsRepo := errors.New("chats repo error")
	s.chatsRepo.EXPECT().GetOpenProblemChatsForManager(s.Ctx, req.ManagerID).Return(nil, errChatsRepo)

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Contains(err.Error(), errChatsRepo.Error())
	s.Empty(resp.Chats)
}

func (s *UseCaseSuite) TestHandle_NoError() {
	// Arrange.
	req := getchats.Request{ID: types.NewRequestID(), ManagerID: types.NewUserID()}
	chats := []chatsrepo.Chat{
		{ID: types.NewChatID(), ClientID: types.NewUserID()},
		{ID: types.NewChatID(), ClientID: types.NewUserID()},
	}
	s.chatsRepo.EXPECT().GetOpenProblemChatsForManager(s.Ctx, req.ManagerID).Return(chats, nil)

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	expectedResponse := getchats.Response{Chats: []getchats.Chat{
		{ID: chats[0].ID, ClientID: chats[0].ClientID},
		{ID: chats[1].ID, ClientID: chats[1].ClientID},
	}}
	s.Equal(expectedResponse, resp)
}
