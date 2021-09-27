package wracha_test

import (
	"context"
	"errors"
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

var errMock = errors.New("mock error")

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
	action        wracha.ActionFunc
	actionResult  wracha.ActionResult
	err           error
	mustRun       bool
	expectedErr   error
	expectedValue interface{}
	postAction    func()
}

type ctxKey string

type badKeyable string

func (t badKeyable) Key() (string, error) {
	return "", errMock
}

type proxiedAdapter struct {
	adapter adapter.Adapter

	existsOverride func(context.Context, string) (bool, error)
	getOverride    func(context.Context, string) ([]byte, error)
	setOverride    func(context.Context, string, time.Duration, []byte) error
	deleteOverride func(context.Context, string) error
	lockOverride   func(context.Context, string) error
	unlockOverride func(context.Context, string) error
}

func (a proxiedAdapter) Exists(ctx context.Context, key string) (bool, error) {
	if a.existsOverride != nil {
		return a.existsOverride(ctx, key)
	}

	return a.adapter.Exists(ctx, key)
}

func (a proxiedAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	if a.getOverride != nil {
		return a.getOverride(ctx, key)
	}

	return a.adapter.Get(ctx, key)
}

func (a proxiedAdapter) Set(ctx context.Context, key string, ttl time.Duration, data []byte) error {
	if a.setOverride != nil {
		return a.setOverride(ctx, key, ttl, data)
	}

	return a.adapter.Set(ctx, key, ttl, data)
}

func (a proxiedAdapter) Delete(ctx context.Context, key string) error {
	if a.deleteOverride != nil {
		return a.deleteOverride(ctx, key)
	}

	return a.adapter.Delete(ctx, key)
}

func (a proxiedAdapter) Lock(ctx context.Context, key string) error {
	if a.lockOverride != nil {
		return a.lockOverride(ctx, key)
	}

	return a.adapter.Lock(ctx, key)
}

func (a proxiedAdapter) Unlock(ctx context.Context, key string) error {
	if a.unlockOverride != nil {
		return a.unlockOverride(ctx, key)
	}

	return a.adapter.Unlock(ctx, key)
}

func makeAction(run *bool, result wracha.ActionResult, err error) wracha.ActionFunc {
	return func(context.Context) (wracha.ActionResult, error) {
		*run = true
		return result, err
	}
}

func makeActionFrom(run *bool, action wracha.ActionFunc) wracha.ActionFunc {
	return func(ctx context.Context) (wracha.ActionResult, error) {
		*run = true
		return action(ctx)
	}
}

func (c tCase) run(sut subtestRunner, actor wracha.Actor) {
	run := false
	var value interface{}
	var err error

	if c.action == nil {
		value, err = actor.Do(c.key, makeAction(&run, c.actionResult, c.err))
	} else {
		value, err = actor.Do(c.key, makeActionFrom(&run, c.action))
	}

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
	adapter *proxiedAdapter
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
	s.adapter = &proxiedAdapter{
		adapter: memory.NewAdapter(),
	}
	s.codec = msgpack.NewCodec()
	s.logger = std.NewLogger()
	s.manager = wracha.NewManager(s.adapter, s.codec, s.logger)
}

func (s ManagerTestSuite) TestActionError() {
	actor := s.manager.On("testing")

	cases := []tCase{
		{
			key:         "testing-key",
			err:         errMock,
			expectedErr: errMock,
			mustRun:     true,
		},
	}

	runCases(&s, actor, cases)
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

func (s ManagerTestSuite) TestDefaultPreActionErrorHandler() {
	actor := s.manager.On("testing")

	cases := []tCase{
		{
			key: badKeyable("bad"),
			actionResult: wracha.ActionResult{
				Cache: false,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(&s, actor, cases)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForKey() {
	run := false
	actor := s.manager.On("testing").SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs) (interface{}, error) {
			run = true
			s.Assert().Equal("key", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return nil, errMock
		},
	)

	cases := []tCase{
		{
			key:           badKeyable("bad"),
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: nil,
		},
	}

	runCases(&s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForGet() {
	s.adapter.getOverride = func(context.Context, string) ([]byte, error) {
		return nil, errMock
	}

	run := false
	actor := s.manager.On("testing").SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs) (interface{}, error) {
			run = true
			s.Assert().Equal("get", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return nil, errMock
		},
	)

	cases := []tCase{
		{
			key:           "testing",
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: nil,
		},
	}

	runCases(&s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForLock() {
	s.adapter.lockOverride = func(context.Context, string) error {
		return errMock
	}

	run := false
	actor := s.manager.On("testing").SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs) (interface{}, error) {
			run = true
			s.Assert().Equal("lock", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return nil, errMock
		},
	)

	cases := []tCase{
		{
			key:           "testing",
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: nil,
		},
	}

	runCases(&s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestDefaultPostActionErrorHandler() {
	run := false
	s.adapter.setOverride = func(context.Context, string, time.Duration, []byte) error {
		run = true
		return errMock
	}

	actor := s.manager.On("testing")

	cases := []tCase{
		{
			key: "testing",
			actionResult: wracha.ActionResult{
				Cache: true,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(&s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPostActionErrorHandlerForStore() {
	s.adapter.setOverride = func(context.Context, string, time.Duration, []byte) error {
		return errMock
	}

	result := wracha.ActionResult{
		Cache: true,
		Value: dummyValue1,
	}

	run := false
	actor := s.manager.On("testing").SetPostActionErrorHandler(
		func(ctx context.Context, args wracha.PostActionErrorHandlerArgs) (interface{}, error) {
			run = true
			s.Assert().Equal(result, args.Result)
			s.Assert().Equal("store", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return nil, errMock
		},
	)

	cases := []tCase{
		{
			key:           "testing",
			actionResult:  result,
			mustRun:       true,
			expectedErr:   errMock,
			expectedValue: nil,
		},
	}

	runCases(&s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestSetTTL() {
	expectedTtl := time.Duration(2) * time.Hour

	s.adapter.setOverride = func(_ context.Context, _ string, ttl time.Duration, _ []byte) error {
		s.Assert().Equal(expectedTtl, ttl)
		return nil
	}

	actor := s.manager.On("testing").
		SetTTL(expectedTtl).
		SetReturnType(new(testStruct))

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
	}

	runCases(&s, actor, cases)
}

func (s ManagerTestSuite) TestSetContext() {
	expectedCtx := context.WithValue(context.Background(), ctxKey("ctxkey"), "value")

	s.adapter.getOverride = func(ctx context.Context, _ string) ([]byte, error) {
		s.Assert().Equal(expectedCtx, ctx)
		return nil, adapter.ErrNotFound
	}

	s.adapter.setOverride = func(ctx context.Context, _ string, _ time.Duration, _ []byte) error {
		s.Assert().Equal(expectedCtx, ctx)
		return nil
	}

	s.adapter.lockOverride = func(ctx context.Context, _ string) error {
		s.Assert().Equal(expectedCtx, ctx)
		return nil
	}

	s.adapter.unlockOverride = func(ctx context.Context, _ string) error {
		s.Assert().Equal(expectedCtx, ctx)
		return nil
	}

	actor := s.manager.On("testing").
		SetContext(expectedCtx).
		SetReturnType(new(testStruct))

	cases := []tCase{
		{
			key: "testing-key",
			action: func(ctx context.Context) (wracha.ActionResult, error) {
				s.Assert().Equal(expectedCtx, ctx)

				return wracha.ActionResult{
					Cache: false,
					Value: dummyValue1,
				}, nil
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(&s, actor, cases)
}

func TestRunManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
