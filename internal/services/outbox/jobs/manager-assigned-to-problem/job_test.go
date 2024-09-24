package managerassignedtoproblemjob_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	messagesrepo "github.com/pershin-daniil/ninja-chat-bank/internal/repositories/messages"
	eventstream "github.com/pershin-daniil/ninja-chat-bank/internal/services/event-stream"
	managerassignedtoproblemjob "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/manager-assigned-to-problem"
	managerassignedtoproblemjobmocks "github.com/pershin-daniil/ninja-chat-bank/internal/services/outbox/jobs/manager-assigned-to-problem/mocks"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func TestJobSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(JobSuite))
}

type JobSuite struct {
	testingh.ContextSuite

	ctrl *gomock.Controller

	messageRepo *managerassignedtoproblemjobmocks.MockmessageRepository
	problemRepo *managerassignedtoproblemjobmocks.MockproblemRepository
	managerLoad *managerassignedtoproblemjobmocks.MockmanagerLoadService
	eventStream *managerassignedtoproblemjobmocks.MockeventStream

	job *managerassignedtoproblemjob.Job
}

func (s *JobSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.messageRepo = managerassignedtoproblemjobmocks.NewMockmessageRepository(s.ctrl)
	s.problemRepo = managerassignedtoproblemjobmocks.NewMockproblemRepository(s.ctrl)
	s.managerLoad = managerassignedtoproblemjobmocks.NewMockmanagerLoadService(s.ctrl)
	s.eventStream = managerassignedtoproblemjobmocks.NewMockeventStream(s.ctrl)

	var err error
	options := managerassignedtoproblemjob.NewOptions(s.messageRepo, s.problemRepo, s.managerLoad, s.eventStream)
	s.job, err = managerassignedtoproblemjob.New(options)
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *JobSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *JobSuite) TestHandle_InvalidOptionsError() {
	job, err := managerassignedtoproblemjob.New(managerassignedtoproblemjob.NewOptions(nil, nil, nil, nil))

	s.Require().Error(err)
	s.Nil(job)
}

func (s *JobSuite) TestHandle_InvalidPayload() {
	// Arrange
	payload := "123"

	// Action
	err := s.job.Handle(s.Ctx, payload)

	// Assert
	s.Require().Error(err)
}

func (s *JobSuite) TestHandle_MessageRepoError() {
	// Arrange
	messageID := types.NewMessageID()
	managerID := types.NewUserID()
	clientID := types.NewUserID()

	payload := s.createPayload(messageID, managerID, clientID)

	errMsgRepo := errors.New("GetMessageByID error")
	s.messageRepo.EXPECT().GetMessageByID(s.Ctx, messageID).Return(nil, errMsgRepo)

	// Action
	err := s.job.Handle(s.Ctx, payload)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), errMsgRepo.Error())
}

func (s *JobSuite) TestHandle_ManagerLoadError() {
	// Arrange
	messageID := types.NewMessageID()
	managerID := types.NewUserID()
	clientID := types.NewUserID()

	payload := s.createPayload(messageID, managerID, clientID)

	message := createServiceMessage(messageID)
	s.messageRepo.EXPECT().GetMessageByID(s.Ctx, messageID).Return(message, nil)

	errManagerLoad := errors.New("CanManagerTakeProblem error")
	s.managerLoad.EXPECT().CanManagerTakeProblem(s.Ctx, managerID).Return(false, errManagerLoad)

	// Action
	err := s.job.Handle(s.Ctx, payload)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), errManagerLoad.Error())
}

func (s *JobSuite) TestHandle_NewChatEventError() {
	// Arrange
	messageID := types.NewMessageID()
	managerID := types.NewUserID()
	clientID := types.NewUserID()

	payload := s.createPayload(messageID, managerID, clientID)

	message := createServiceMessage(messageID)
	s.messageRepo.EXPECT().GetMessageByID(s.Ctx, messageID).Return(message, nil)

	s.managerLoad.EXPECT().CanManagerTakeProblem(s.Ctx, managerID).Return(false, nil)

	errNewChatEvent := errors.New("publish NewChatEvent error")
	chatEventMatcher := &eventstream.NewChatEventMatcher{NewChatEvent: createNewChatEvent(message, managerID)}
	s.eventStream.EXPECT().Publish(s.Ctx, managerID, chatEventMatcher).Return(errNewChatEvent)

	// Action
	err := s.job.Handle(s.Ctx, payload)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), errNewChatEvent.Error())
}

func (s *JobSuite) TestHandle_NewMessageEventError() {
	// Arrange
	messageID := types.NewMessageID()
	managerID := types.NewUserID()
	clientID := types.NewUserID()

	payload := s.createPayload(messageID, managerID, clientID)

	message := createServiceMessage(messageID)
	s.messageRepo.EXPECT().GetMessageByID(s.Ctx, messageID).Return(message, nil)

	s.managerLoad.EXPECT().CanManagerTakeProblem(s.Ctx, managerID).Return(false, nil)

	chatEventMatcher := &eventstream.NewChatEventMatcher{NewChatEvent: createNewChatEvent(message, managerID)}
	s.eventStream.EXPECT().Publish(s.Ctx, managerID, chatEventMatcher).Return(nil)

	errNewMessageEvent := errors.New("publish NewMessageEvent error")
	messageEventMatcher := &eventstream.NewMessageEventMatcher{NewMessageEvent: createNewMessageEvent(message, clientID)}
	s.eventStream.EXPECT().Publish(s.Ctx, clientID, messageEventMatcher).Return(errNewMessageEvent)

	// Action
	err := s.job.Handle(s.Ctx, payload)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), errNewMessageEvent.Error())
}

func (s *JobSuite) TestHandle() {
	for _, canTakeMoreProblem := range []bool{true, false} {
		s.Run(fmt.Sprintf("can take more problem %v", canTakeMoreProblem), func() {
			// Arrange
			messageID := types.NewMessageID()
			managerID := types.NewUserID()
			clientID := types.NewUserID()

			payload := s.createPayload(messageID, managerID, clientID)

			message := createServiceMessage(messageID)
			s.messageRepo.EXPECT().GetMessageByID(s.Ctx, messageID).Return(message, nil)

			s.managerLoad.EXPECT().CanManagerTakeProblem(s.Ctx, managerID).Return(canTakeMoreProblem, nil)

			chatEventMatcher := &eventstream.NewChatEventMatcher{NewChatEvent: createNewChatEvent(message, managerID)}
			chatEventMatcher.NewChatEvent.CanTakeMoreProblems = canTakeMoreProblem
			s.eventStream.EXPECT().Publish(s.Ctx, managerID, chatEventMatcher).Return(nil)

			messageEventMatcher := &eventstream.NewMessageEventMatcher{NewMessageEvent: createNewMessageEvent(message, clientID)}
			s.eventStream.EXPECT().Publish(s.Ctx, clientID, messageEventMatcher).Return(nil)

			// Action
			err := s.job.Handle(s.Ctx, payload)

			// Assert
			s.Require().NoError(err)
		})
	}
}

func createServiceMessage(messageID types.MessageID) *messagesrepo.Message {
	return &messagesrepo.Message{
		ID:                  messageID,
		ChatID:              types.NewChatID(),
		AuthorID:            types.UserIDNil,
		RequestID:           types.NewRequestID(),
		ProblemID:           types.NewProblemID(),
		Body:                "Вам ответит менеджер Вася.",
		CreatedAt:           time.Now(),
		IsVisibleForClient:  true,
		IsVisibleForManager: true,
		IsService:           true,
	}
}

func (s *JobSuite) createPayload(messageID types.MessageID, managerID types.UserID, clientID types.UserID) string {
	payload := managerassignedtoproblemjob.Payload{
		MessageID: messageID,
		ManagerID: managerID,
		ClientID:  clientID,
	}
	payloadStr, err := managerassignedtoproblemjob.Marshal(payload)
	s.Require().NoError(err)

	return payloadStr
}

func createNewChatEvent(message *messagesrepo.Message, managerID types.UserID) *eventstream.NewChatEvent {
	return &eventstream.NewChatEvent{
		EventID:             types.NewEventID(),
		ChatID:              message.ChatID,
		ClientID:            managerID,
		RequestID:           message.RequestID,
		CanTakeMoreProblems: false,
	}
}

func createNewMessageEvent(message *messagesrepo.Message, clientID types.UserID) *eventstream.NewMessageEvent {
	return &eventstream.NewMessageEvent{
		EventID:     types.NewEventID(),
		RequestID:   message.RequestID,
		ChatID:      message.ChatID,
		MessageID:   message.ID,
		AuthorID:    clientID,
		CreatedAt:   message.CreatedAt,
		MessageBody: message.Body,
		IsService:   message.IsService,
	}
}
