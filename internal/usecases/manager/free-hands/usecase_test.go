package freehands_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
	freehands "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/free-hands"
	freehandsmocks "github.com/pershin-daniil/ninja-chat-bank/internal/usecases/manager/free-hands/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl      *gomock.Controller
	mLoadMock *freehandsmocks.MockmanagerLoadService
	mPoolMock *freehandsmocks.MockmanagerPool
	uCase     freehands.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	var err error
	s.ctrl = gomock.NewController(s.T())
	s.mPoolMock = freehandsmocks.NewMockmanagerPool(s.ctrl)
	s.mLoadMock = freehandsmocks.NewMockmanagerLoadService(s.ctrl)

	s.uCase, err = freehands.New(freehands.NewOptions(s.mPoolMock, s.mLoadMock))
	s.Require().NoError(err)
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *UseCaseSuite) TestHandler() {
	req := freehands.Request{
		ID:        types.RequestID{},
		ManagerID: types.UserID{},
	}

	s.Run("invalid request", func() {
		err := s.uCase.Handle(s.Ctx, req)
		s.Require().Error(err)
	})

	req.ID = types.NewRequestID()
	req.ManagerID = types.NewUserID()
	s.Run("managerPool.Contains err", func() {
		s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).Return(false, errors.New("bang"))
		err := s.uCase.Handle(s.Ctx, req)
		s.Require().Error(err)
	})

	s.Run("manager capacity is exceeded", func() {
		s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).Return(false, nil)
		err := s.uCase.Handle(s.Ctx, req)
		s.Require().ErrorIs(err, freehands.ErrManagerOverloaded)
	})

	s.Run("put manager returns error", func() {
		s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).Return(true, nil)
		s.mPoolMock.EXPECT().Put(s.Ctx, req.ManagerID).Return(errors.New("bang"))
		err := s.uCase.Handle(s.Ctx, req)
		s.Require().Error(err)
	})

	s.Run("success", func() {
		s.mLoadMock.EXPECT().CanManagerTakeProblem(s.Ctx, req.ManagerID).Return(true, nil)
		s.mPoolMock.EXPECT().Put(s.Ctx, req.ManagerID).Return(nil)
		err := s.uCase.Handle(s.Ctx, req)
		s.Require().NoError(err)
	})
}
