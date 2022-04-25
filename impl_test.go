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
	Records    map[string]any
}

type tCase[T any] struct {
	key           any
	action        wracha.ActionFunc[T]
	actionResult  wracha.ActionResult[T]
	err           error
	mustRun       bool
	expectedErr   error
	expectedValue testStruct
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

func makeAction[T any](run *bool, result wracha.ActionResult[T], err error) wracha.ActionFunc[T] {
	return func(context.Context) (wracha.ActionResult[T], error) {
		*run = true
		return result, err
	}
}

func makeActionFrom[T any](run *bool, action wracha.ActionFunc[T]) wracha.ActionFunc[T] {
	return func(ctx context.Context) (wracha.ActionResult[T], error) {
		*run = true
		return action(ctx)
	}
}

func (c tCase[T]) run(ctx context.Context, sut subtestRunner, actor wracha.Actor[T]) {
	run := false
	var value T
	var err error

	if c.action == nil {
		value, err = actor.Do(ctx, c.key, makeAction(&run, c.actionResult, c.err))
	} else {
		value, err = actor.Do(ctx, c.key, makeActionFrom(&run, c.action))
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

func runCases[T any](ctx context.Context, sut subtestRunner, actor wracha.Actor[T], cases []tCase[T]) {
	for i, c := range cases {
		sut.Run(fmt.Sprintf("Test Case #%d", i), func() {
			c.run(ctx, sut, actor)
		})
	}
}

type ManagerTestSuite struct {
	suite.Suite
	adapter *proxiedAdapter
	codec   codec.Codec
	logger  logger.Logger
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
		Records: map[string]any{
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
		Records: map[string]any{
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
}

func (s ManagerTestSuite) TestActionError() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key:         "testing-key",
			err:         errMock,
			expectedErr: errMock,
			mustRun:     true,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithNonCachedValues() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithCachedValues() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
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

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithExpiredTTLCachedValues() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	duration := time.Duration(2) * time.Second

	cases := []tCase[testStruct]{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
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
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestActionWithInvalidatedCachedValues() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: true,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
			postAction: func() {
				actor.Invalidate(context.Background(), "testing-key")
			},
		},
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue2,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue2,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestDefaultPreActionErrorHandler() {
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key: badKeyable("bad"),
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForKey() {
	run := false
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	}).SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs[testStruct]) (testStruct, error) {
			run = true
			s.Assert().Equal("key", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return testStruct{}, errMock
		},
	)

	cases := []tCase[testStruct]{
		{
			key:           badKeyable("bad"),
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: testStruct{},
		},
	}

	runCases(context.Background(), &s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForGet() {
	s.adapter.getOverride = func(context.Context, string) ([]byte, error) {
		return nil, errMock
	}

	run := false
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	}).SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs[testStruct]) (testStruct, error) {
			run = true
			s.Assert().Equal("get", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return testStruct{}, errMock
		},
	)

	cases := []tCase[testStruct]{
		{
			key:           "testing",
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: testStruct{},
		},
	}

	runCases(context.Background(), &s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPreActionErrorHandlerForLock() {
	s.adapter.lockOverride = func(context.Context, string) error {
		return errMock
	}

	run := false
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	}).SetPreActionErrorHandler(
		func(ctx context.Context, args wracha.PreActionErrorHandlerArgs[testStruct]) (testStruct, error) {
			run = true
			s.Assert().Equal("lock", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return testStruct{}, errMock
		},
	)

	cases := []tCase[testStruct]{
		{
			key:           "testing",
			mustRun:       false,
			expectedErr:   errMock,
			expectedValue: testStruct{},
		},
	}

	runCases(context.Background(), &s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestDefaultPostActionErrorHandler() {
	run := false
	s.adapter.setOverride = func(context.Context, string, time.Duration, []byte) error {
		run = true
		return errMock
	}

	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	})

	cases := []tCase[testStruct]{
		{
			key: "testing",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: true,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(context.Background(), &s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestPostActionErrorHandlerForStore() {
	s.adapter.setOverride = func(context.Context, string, time.Duration, []byte) error {
		return errMock
	}

	result := wracha.ActionResult[testStruct]{
		Cache: true,
		Value: dummyValue1,
	}

	run := false
	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	}).SetPostActionErrorHandler(
		func(ctx context.Context, args wracha.PostActionErrorHandlerArgs[testStruct]) (testStruct, error) {
			run = true
			s.Assert().Equal(result, args.Result)
			s.Assert().Equal("store", args.ErrCategory)
			s.Assert().ErrorIs(args.Err, errMock)
			return testStruct{}, errMock
		},
	)

	cases := []tCase[testStruct]{
		{
			key:           "testing",
			actionResult:  result,
			mustRun:       true,
			expectedErr:   errMock,
			expectedValue: testStruct{},
		},
	}

	runCases(context.Background(), &s, actor, cases)

	s.Assert().True(run)
}

func (s ManagerTestSuite) TestSetTTL() {
	expectedTtl := time.Duration(2) * time.Hour

	s.adapter.setOverride = func(_ context.Context, _ string, ttl time.Duration, _ []byte) error {
		s.Assert().Equal(expectedTtl, ttl)
		return nil
	}

	actor := wracha.NewActor[testStruct]("testing", wracha.ActorOptions{
		Adapter: s.adapter,
		Codec:   s.codec,
		Logger:  s.logger,
	}).SetTTL(expectedTtl)

	cases := []tCase[testStruct]{
		{
			key: "testing-key",
			actionResult: wracha.ActionResult[testStruct]{
				Cache: false,
				Value: dummyValue1,
			},
			mustRun:       true,
			expectedErr:   nil,
			expectedValue: dummyValue1,
		},
	}

	runCases(context.Background(), &s, actor, cases)
}

func TestRunManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
