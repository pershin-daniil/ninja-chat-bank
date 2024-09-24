package sendmessage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	"github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/payload/simpleid"
	sendmanagermessagejob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/send-manager-message"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	sendmessage "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/send-message"
	sendmessagemocks "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/send-message/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl        *gomock.Controller
	msgRepo     *sendmessagemocks.MockmessagesRepository
	problemRepo *sendmessagemocks.MockproblemsRepository
	txtor       *sendmessagemocks.Mocktransactor
	outBoxSvc   *sendmessagemocks.MockoutboxService
	uCase       sendmessage.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.msgRepo = sendmessagemocks.NewMockmessagesRepository(s.ctrl)
	s.outBoxSvc = sendmessagemocks.NewMockoutboxService(s.ctrl)
	s.problemRepo = sendmessagemocks.NewMockproblemsRepository(s.ctrl)
	s.txtor = sendmessagemocks.NewMocktransactor(s.ctrl)

	var err error
	s.uCase, err = sendmessage.New(sendmessage.NewOptions(s.msgRepo, s.outBoxSvc, s.problemRepo, s.txtor))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestInvalidRequestError() {
	// Arrange.
	req := sendmessage.Request{}

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Require().ErrorIs(err, sendmessage.ErrInvalidRequest)
	s.Require().Empty(response)
}

func (s *UseCaseSuite) TestProblemRepoError() {
	// Arrange.
	req := sendmessage.Request{
		ID:          types.NewRequestID(),
		ManagerID:   types.NewUserID(),
		ChatID:      types.NewChatID(),
		MessageBody: "Hello",
	}

	errProblemsRepo := errors.New("problems repo error")
	s.problemRepo.EXPECT().GetAssignedProblemID(s.Ctx, req.ManagerID, req.ChatID).Return(types.ProblemIDNil, errProblemsRepo)

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Contains(err.Error(), errProblemsRepo.Error())
	s.Empty(response)
}

func (s *UseCaseSuite) TestMessagesRepoError() {
	// Arrange.
	req := sendmessage.Request{
		ID:          types.NewRequestID(),
		ManagerID:   types.NewUserID(),
		ChatID:      types.NewChatID(),
		MessageBody: "Hello",
	}

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(s.Ctx, req.ManagerID, req.ChatID).Return(problemID, nil)

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		},
	)

	errMessagesRepo := errors.New("messages repo error")
	s.msgRepo.EXPECT().
		CreateFullVisible(s.Ctx, req.ID, problemID, req.ChatID, req.ManagerID, req.MessageBody).
		Return(nil, errMessagesRepo)

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Contains(err.Error(), errMessagesRepo.Error())
	s.Empty(response)
}

func (s *UseCaseSuite) TestOutboxServiceError() {
	// Arrange.
	req := sendmessage.Request{
		ID:          types.NewRequestID(),
		ManagerID:   types.NewUserID(),
		ChatID:      types.NewChatID(),
		MessageBody: "Hello",
	}

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(s.Ctx, req.ManagerID, req.ChatID).Return(problemID, nil)

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		},
	)

	message := &messagesrepo.Message{
		ID:                  types.NewMessageID(),
		ChatID:              req.ChatID,
		AuthorID:            req.ManagerID,
		RequestID:           req.ID,
		ProblemID:           problemID,
		Body:                req.MessageBody,
		CreatedAt:           time.Now(),
		IsVisibleForClient:  true,
		IsVisibleForManager: true,
	}
	s.msgRepo.EXPECT().
		CreateFullVisible(s.Ctx, req.ID, problemID, req.ChatID, req.ManagerID, req.MessageBody).
		Return(message, nil)

	payload, err := simpleid.Marshal[types.MessageID](message.ID)
	s.Require().NoError(err)
	errOutbox := errors.New("outbox service error")
	s.outBoxSvc.EXPECT().Put(s.Ctx, sendmanagermessagejob.Name, payload, gomock.Any()).Return(types.JobIDNil, errOutbox)

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Contains(err.Error(), errOutbox.Error())
	s.Empty(response)
}

func (s *UseCaseSuite) TestTransactionCommitError() {
	// Arrange.
	req := sendmessage.Request{
		ID:          types.NewRequestID(),
		ManagerID:   types.NewUserID(),
		ChatID:      types.NewChatID(),
		MessageBody: "Hello",
	}

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(s.Ctx, req.ManagerID, req.ChatID).Return(problemID, nil)

	errTxCommit := errors.New("commit error")
	s.txtor.EXPECT().RunInTx(s.Ctx, gomock.Any()).Return(errTxCommit)

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Contains(err.Error(), errTxCommit.Error())
}

func (s *UseCaseSuite) TestSuccess() {
	// Arrange.
	req := sendmessage.Request{
		ID:          types.NewRequestID(),
		ManagerID:   types.NewUserID(),
		ChatID:      types.NewChatID(),
		MessageBody: "Hello",
	}

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(s.Ctx, req.ManagerID, req.ChatID).Return(problemID, nil)

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		},
	)

	message := &messagesrepo.Message{
		ID:                  types.NewMessageID(),
		ChatID:              req.ChatID,
		AuthorID:            req.ManagerID,
		RequestID:           req.ID,
		ProblemID:           problemID,
		Body:                req.MessageBody,
		CreatedAt:           time.Now(),
		IsVisibleForClient:  true,
		IsVisibleForManager: true,
	}
	s.msgRepo.EXPECT().
		CreateFullVisible(s.Ctx, req.ID, problemID, req.ChatID, req.ManagerID, req.MessageBody).
		Return(message, nil)

	payload, err := simpleid.Marshal[types.MessageID](message.ID)
	s.Require().NoError(err)
	s.outBoxSvc.EXPECT().Put(s.Ctx, sendmanagermessagejob.Name, payload, gomock.Any()).Return(types.NewJobID(), nil)

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Equal(response.MessageID, message.ID)
	s.Equal(response.CreatedAt, message.CreatedAt)
}
