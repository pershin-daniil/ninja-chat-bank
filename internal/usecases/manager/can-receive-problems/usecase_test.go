package canreceiveproblems_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	canreceiveproblems "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems"
	canreceiveproblemsmocks "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/can-receive-problems/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl      *gomock.Controller
	mLoadMock *canreceiveproblemsmocks.MockmanagerLoadService
	mPoolMock *canreceiveproblemsmocks.MockmanagerPool
	uCase     canreceiveproblems.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mLoadMock = canreceiveproblemsmocks.NewMockmanagerLoadService(s.ctrl)
	s.mPoolMock = canreceiveproblemsmocks.NewMockmanagerPool(s.ctrl)

	var err error
	s.uCase, err = canreceiveproblems.New(canreceiveproblems.NewOptions(s.mPoolMock, s.mLoadMock))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := canreceiveproblems.Request{}
	s.Run("request id validation", func() {
		// Action.
		resp, err := s.uCase.Handle(s.Ctx, req)
		// Assert.
		s.Require().Error(err)
		s.Empty(resp.Result)
	})
	s.Run("manager id validation", func() {
		req.ID = types.NewRequestID()

		// Action.
		resp, err := s.uCase.Handle(s.Ctx, req)
		// Assert.
		s.Require().Error(err)
		s.Empty(resp.Result)
	})
}

func (s *UseCaseSuite) TestManagerPoolError() {
	// Arrange.
	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: types.NewUserID(),
	}

	s.mPoolMock.EXPECT().Contains(s.Ctx, req.ManagerID).
		Return(false, errors.New("error"))

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Empty(resp)
}

func (s *UseCaseSuite) TestManagerAlreadyInPool() {
	// Arrange.
	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: types.NewUserID(),
	}

	s.mPoolMock.EXPECT().Contains(s.Ctx, req.ManagerID).
		Return(true, nil)

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().False(resp.Result)
}

func (s *UseCaseSuite) TestManagerLoadError() {
	// Arrange.
	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: types.NewUserID(),
	}

	s.mPoolMock.EXPECT().Contains(s.Ctx, req.ManagerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).
		Return(false, errors.New("error"))

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Empty(resp)
}

func (s *UseCaseSuite) TestManagerLoadOK_True() {
	// Arrange.
	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: types.NewUserID(),
	}

	s.mPoolMock.EXPECT().Contains(s.Ctx, req.ManagerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).
		Return(true, nil)

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().True(resp.Result)
}

func (s *UseCaseSuite) TestManagerLoadOK_False() {
	// Arrange.
	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: types.NewUserID(),
	}

	s.mPoolMock.EXPECT().Contains(s.Ctx, req.ManagerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).
		Return(false, nil)

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().False(resp.Result)
}
