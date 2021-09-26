package wracha_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pwnedgod/wracha"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/memory"
	"github.com/pwnedgod/wracha/codec"
	"github.com/pwnedgod/wracha/codec/msgpack"
	"github.com/pwnedgod/wracha/logger"
	"github.com/pwnedgod/wracha/logger/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type subtestRunner interface {
	Run(name string, subtest func()) bool
	Assert() *assert.Assertions
}

type testSubStruct struct {
	Number string
	By     string
}

type testStruct struct {
	Name       string
	Time       time.Time
	Age        int
	Percentage float64
	Account    testSubStruct
	Scopes     []string
	Records    map[string]interface{}
}

type tCase struct {
	key           interface{}
	actionResult  wracha.ActionResult
	mustRun       bool
	expectedErr   error
	expectedValue interface{}
	postAction    func()
}

func makeAction(run *bool, result wracha.ActionResult) wracha.ActionFunc {
	return func(context.Context) (wracha.ActionResult, error) {
		*run = true
		return result, nil
	}
}

func (c tCase) run(sut subtestRunner, actor wracha.Actor) {
	run := false
	value, err := actor.Do(c.key, makeAction(&run, c.actionResult))

	if c.expectedErr == nil {
		sut.Assert().Nil(err)
	} else {
		sut.Assert().ErrorIs(err, c.expectedErr)
	}
	sut.Assert().Equal(c.expectedValue, value)
	if c.mustRun {
		sut.Assert().True(run)
	} else {
		sut.Assert().False(run)
	}

	if c.postAction != nil {
		c.postAction()
	}
}

func runCases(sut subtestRunner, actor wracha.Actor, cases []tCase) {
	for i, c := range cases {
		sut.Run(fmt.Sprintf("Test Case #%d", i), func() {
			c.run(sut, actor)
		})
	}
}

type ManagerTestSuite struct {
	suite.Suite
	adapter adapter.Adapter
	codec   codec.Codec
	logger  logger.Logger
	manager wracha.Manager
}

var (
	now = time.Date(2020, 10, 06, 8, 59, 39, 0, time.Local)

	dummyValue1 = testStruct{
		Name:       "testing-1",
		Time:       now,
		Age:        50,
		Percentage: 0.5,
		Account: testSubStruct{
			Number: "1234567890",
			By:     "John Doe",
		},
		Scopes: []string{"edit", "create", "view"},
		Records: map[string]interface{}{
			"device":           "Android",
			"usingRecognition": false,
		},
	}

	dummyValue2 = testStruct{
		Name:       "testing-2",
		Time:       now.AddDate(0, 0, 1),
		Age:        30,
		Percentage: 0.75,
		Scopes:     []string{"*"},
		Account: testSubStruct{
			Number: "0987654321",
			By:     "Jane Dee",
		},
		Records: map[string]interface{}{
			"status":           "Disabled",
			"usingRecognition": true,
		},
	}
)

func (s *ManagerTestSuite) SetupTest() {
	s.adapter = memory.NewAdapter()
	s.codec = msgpack.NewCodec()
	s.logger = std.NewLogger()
	s.manager = wracha.NewManager(s.adapter, s.codec, s.logger)
}

func (s ManagerTestSuite) TestSameActorInstance() {
	actor1 := s.manager.On("testing1")
	actor2 := s.manager.On("testing2")
	actor1D := s.manager.On("testing1")
	actor2D := s.manager.On("testing2")

	s.False(actor1 == actor2)
	s.True(actor1 == actor1D)
	s.True(actor2 == actor2D)
}

func (s ManagerTestSuite) TestActionWithNonCachedValues() {
	actor := s.manager.On("testing").SetReturnType(new(testStruct))

	cases := []tCase{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: false,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(&s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithCachedValues() {
	actor := s.manager.On("testing").SetReturnType(new(testStruct))

	cases := []tCase{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: true,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
		{
			key:           "testing-key",
			mustRun:       false,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(&s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithExpiredTTLCachedValues() {
	actor := s.manager.On("testing").SetReturnType(new(testStruct))

	duration := time.Duration(2) * time.Second

	cases := []tCase{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: true,
				TTL:   duration,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
			postAction: func() {
				time.Sleep(duration * 2)
			},
		},
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(&s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithInvalidatedCachedValues() {
	actor := s.manager.On("testing").SetReturnType(new(testStruct))

	cases := []tCase{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: true,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
			postAction: func() {
				actor.Invalidate("testing-key")
			},
		},
		{
			key: "testing-key",
			actionResult: wracha.ActionResult{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(&s, actor, cases)
}

func TestRunManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
