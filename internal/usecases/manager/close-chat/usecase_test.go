package closechat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	problemsrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/problems"
	closechatjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/close-chat"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	closechat "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/close-chat"
	closechatsmocks "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/close-chat/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	problemsRepo  *closechatsmocks.MockproblemsRepository
	messagesRepo  *closechatsmocks.MockmessagesRepository
	chatsRepo     *closechatsmocks.MockchatsRepository
	outboxService *closechatsmocks.MockoutboxService
	transactor    *closechatsmocks.Mocktransactor
	txMockFunc    func(context.Context, func(context.Context) error) error

	ctrl  *gomock.Controller
	uCase closechat.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.problemsRepo = closechatsmocks.NewMockproblemsRepository(s.ctrl)
	s.messagesRepo = closechatsmocks.NewMockmessagesRepository(s.ctrl)
	s.chatsRepo = closechatsmocks.NewMockchatsRepository(s.ctrl)
	s.outboxService = closechatsmocks.NewMockoutboxService(s.ctrl)
	s.transactor = closechatsmocks.NewMocktransactor(s.ctrl)
	s.txMockFunc = func(ctx context.Context, f func(context.Context) error) error { return f(ctx) }

	var err error
	s.uCase, err = closechat.New(
		closechat.NewOptions(s.problemsRepo, s.messagesRepo, s.chatsRepo, s.outboxService, s.transactor),
	)
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestInvalidRequest() {
	// Arrange
	req := closechat.Request{ChatID: types.ChatID{}, ManagerID: types.UserID{}}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert
	s.Require().ErrorIs(err, closechat.ErrInvalidRequest)
}

func (s *UseCaseSuite) TestChatsRepoError() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}

	errChats := errors.New("db error")
	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(types.UserIDNil, errChats)

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert
	s.Require().Error(err)
	s.Require().Contains(err.Error(), errChats.Error())
}

func (s *UseCaseSuite) TestProblemRepoError() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}

	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(types.UserIDNil, nil)

	s.transactor.EXPECT().RunInTx(s.Ctx, gomock.Any()).DoAndReturn(s.txMockFunc)

	errProblems := errors.New("db error")
	s.problemsRepo.EXPECT().ResolveAssignedProblem(s.Ctx, req.ChatID, req.ManagerID).Return(nil, errProblems)

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), errProblems.Error())
}

func (s *UseCaseSuite) TestProblemNotFoundError() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}

	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(types.UserIDNil, nil)

	s.transactor.EXPECT().RunInTx(s.Ctx, gomock.Any()).DoAndReturn(s.txMockFunc)

	s.problemsRepo.EXPECT().ResolveAssignedProblem(s.Ctx, req.ChatID, req.ManagerID).Return(nil, problemsrepo.ErrProblemNotFound)

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert
	s.ErrorIs(err, closechat.ErrNoActiveProblemInChat)
}

func (s *UseCaseSuite) TestOutboxError() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}
	clientID := types.NewUserID()

	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(clientID, nil)

	s.transactor.EXPECT().RunInTx(s.Ctx, gomock.Any()).DoAndReturn(s.txMockFunc)

	problem := &problemsrepo.Problem{ID: types.NewProblemID()}
	s.problemsRepo.EXPECT().ResolveAssignedProblem(s.Ctx, req.ChatID, req.ManagerID).Return(problem, nil)

	msg := &messagesrepo.Message{ID: types.NewMessageID()}
	s.messagesRepo.EXPECT().
		CreateServiceClientVisible(s.Ctx, req.RequestID, problem.ID, req.ChatID, gomock.Any()).
		Return(msg, nil)

	errOutbox := errors.New("outbox error")
	payload, err := closechatjob.Marshal(closechatjob.Payload{
		RequestID: req.RequestID,
		ManagerID: req.ManagerID,
		MessageID: msg.ID,
		ClientID:  clientID,
	})
	s.Require().NoError(err)
	s.outboxService.EXPECT().Put(s.Ctx, closechatjob.Name, payload, gomock.Any()).Return(types.JobIDNil, errOutbox)

	// Action
	err = s.uCase.Handle(s.Ctx, req)

	// Assert
	s.Require().Error(err)
	s.Require().Contains(err.Error(), errOutbox.Error())
}

func (s *UseCaseSuite) TestTransactionCommitError() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}

	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(types.UserIDNil, nil)

	errTxCommit := errors.New("tx commit error")
	s.transactor.EXPECT().RunInTx(s.Ctx, gomock.Any()).Return(errTxCommit)

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert
	s.Require().Error(err)
	s.Require().Contains(err.Error(), errTxCommit.Error())
}

func (s *UseCaseSuite) TestSuccess() {
	// Arrange
	req := closechat.Request{
		ChatID:    types.NewChatID(),
		ManagerID: types.NewUserID(),
		RequestID: types.NewRequestID(),
	}
	clientID := types.NewUserID()

	s.chatsRepo.EXPECT().GetClientIDByChatID(s.Ctx, req.ChatID).Return(clientID, nil)

	s.transactor.EXPECT().RunInTx(s.Ctx, gomock.Any()).DoAndReturn(s.txMockFunc)

	problem := &problemsrepo.Problem{ID: types.NewProblemID()}
	s.problemsRepo.EXPECT().ResolveAssignedProblem(s.Ctx, req.ChatID, req.ManagerID).Return(problem, nil)

	msg := &messagesrepo.Message{ID: types.NewMessageID()}
	s.messagesRepo.EXPECT().
		CreateServiceClientVisible(s.Ctx, req.RequestID, problem.ID, req.ChatID, gomock.Any()).
		Return(msg, nil)

	payload, err := closechatjob.Marshal(closechatjob.Payload{
		RequestID: req.RequestID,
		ManagerID: req.ManagerID,
		MessageID: msg.ID,
		ClientID:  clientID,
	})
	s.Require().NoError(err)
	s.outboxService.EXPECT().Put(s.Ctx, closechatjob.Name, payload, gomock.Any()).Return(types.NewJobID(), nil)

	// Action
	err = s.uCase.Handle(s.Ctx, req)

	// Assert
	s.NoError(err)
}
