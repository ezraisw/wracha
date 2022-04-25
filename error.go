package wracha

import "fmt"

type (
	baseError struct {
		category    string
		message     string
		previousErr error
	}

	preActionError struct {
		baseError
	}

	postActionError[T any] struct {
		baseError
		result ActionResult[T]
	}
)

func newPreActionError(category string, message string, previousErr error) *preActionError {
	return &preActionError{
		baseError: baseError{
			category:    category,
			message:     message,
			previousErr: previousErr,
		},
	}
}

func newPostActionError[T any](category string, message string, result ActionResult[T], previousErr error) *postActionError[T] {
	return &postActionError[T]{
		baseError: baseError{
			category:    category,
			message:     message,
			previousErr: previousErr,
		},
		result: result,
	}
}

func (e baseError) Error() string {
	return fmt.Sprintf("%s (%s)", e.message, e.previousErr.Error())
}

func (a baseError) Unwrap() error {
	return a.previousErr
}
