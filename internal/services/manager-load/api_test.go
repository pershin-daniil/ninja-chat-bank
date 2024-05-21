package managerload_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	managerload "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load"
	managerloadmocks "github.com/pershin-daniil/ninja-chat-bank/internal/services/manager-load/mocks"
	"github.com/pershin-daniil/ninja-chat-bank/internal/testingh"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

type ServiceSuite struct {
	testingh.ContextSuite

	ctrl *gomock.Controller

	problemsRepo *managerloadmocks.MockproblemsRepository
	managerLoad  *managerload.Service
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.problemsRepo = managerloadmocks.NewMockproblemsRepository(s.ctrl)

	s.ContextSuite.SetupTest()
}

func (s *ServiceSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *ServiceSuite) TestInitManagerLoad() {
	var err error

	s.Run("0 max problem at time", func() {
		s.managerLoad, err = managerload.New(managerload.NewOptions(0, s.problemsRepo))
		s.Require().Error(err)
	})

	s.Run("valid max problem at time", func() {
		for i := 1; i <= 30; i++ {
			s.managerLoad, err = managerload.New(managerload.NewOptions(i, s.problemsRepo))
			s.Require().NoError(err)
		}
	})

	s.Run("limit exceeded", func() {
		s.managerLoad, err = managerload.New(managerload.NewOptions(math.MaxInt, s.problemsRepo))
		s.Require().Error(err)
	})
}

func (s *ServiceSuite) TestProblemsForManager() {
	var err error
	s.managerLoad, err = managerload.New(managerload.NewOptions(5, s.problemsRepo))
	s.Require().NoError(err)

	managerID := types.NewUserID()

	for i := 0; i <= 6; i++ {
		s.Run("get open problems", func() {
			s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(s.Ctx, managerID).Return(i, nil)
			can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)
			s.Require().NoError(err)
			s.Require().Equal(i < 5, can)
		})
	}

	s.Run("failed to get problems", func() {
		s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(s.Ctx, managerID).Return(0, errors.New("error"))
		can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)
		s.Require().Error(err)
		s.Require().False(can)
	})
}
